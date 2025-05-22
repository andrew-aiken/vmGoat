package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
	"infrasec.sh/vmGoat/pkg/types"
)

// Create handles the create command
func Create(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	scenariosPath := "/Users/aaiken/Private/vmGoat/scenarios"

	scenario := cli.Args().First()
	if !validateScenario(scenario, scenariosPath) {
		log.Info().Msgf("\nUsage: %s", cli.UsageText)
		return nil
	}

	// Read the config directory from the context.
	// This should be under the home directory of the user. (`~/.config/vmgoat`)
	configDir, _ := ctx.Value("configDirectory").(string)

	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	// Get user's home directory for AWS credentials
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	// USED FOR LATER
	// Get current working directory for base path
	// currentDir, err := os.Getwd()
	// if err != nil {
	// 	return fmt.Errorf("failed to get current directory: %v", err)
	// }

	awsProfile := cli.String("aws-profile")
	awsRegion := cli.String("aws-region")

	if err := handler.ResolveConfigValue(&awsProfile, &config.AWS.Profile); err != nil {
		return fmt.Errorf("failed to resolve AWS profile: %v", err)
	}

	if err := handler.ResolveConfigValue(&awsRegion, &config.AWS.Region); err != nil {
		return fmt.Errorf("failed to resolve AWS profile: %v", err)
	}

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}

	log.Debug().Msg("Config updated successfully")

	containerOptions := types.ContainerOptions{
		Allowlist:  config.IpAddresses,
		ConfigDir:  configDir,
		HomeDir:    homeDir,
		AwsProfile: awsProfile,
		AwsRegion:  awsRegion,
	}

	config.Scenarios[scenario] = types.Scenario{
		Provider: "aws",
		Path:     "tmp",
	}

	if err := LaunchInitContainer(ctx, containerOptions); err != nil {
		return fmt.Errorf("failed to launch init container: %v", err)
	}
	log.Debug().Msg("Base container successfully initialized")

	if err := LaunchBaseContainer(ctx, containerOptions, "apply"); err != nil {
		return fmt.Errorf("failed to launch base container: %v", err)
	}
	log.Debug().Msg("Base container successfully launched")

	// Initialize the scenarios init container
	if err := LaunchInitScenarioContainer(ctx, containerOptions, scenario); err != nil {
		return fmt.Errorf("failed to launch init container: %v", err)
	}
	log.Debug().Msg("Scenario container successfully initialized")

	// Deploy the scenario
	if err := LaunchScenarioContainer(ctx, containerOptions, "apply", scenario); err != nil {
		return fmt.Errorf("failed to launch base container: %v", err)
	}
	log.Debug().Msg("Scenario container successfully launched")

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}
	log.Debug().Msg("Config updated successfully with scenario")

	// Ansible
	statePath := filepath.Join(configDir, "state", "scenario", scenario, "scenario.tfstate")

	file, err := os.Open(statePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Parse the raw structure into a map
	var data struct {
		Outputs map[string]Output `json:"outputs"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		panic(err)
	}

	// // Loop through keys and print them
	// for key, output := range data.Outputs {
	// 	fmt.Printf("Key: %s, Value: %s\n", key, output.Value)
	// }

	// Create a temporary directory for Ansible files
	tmpDir, err := os.MkdirTemp("", "vmgoat-ansible-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log.Debug().Msgf("Temporary directory created: %s", tmpDir)

	// Copy inventory template to temp directory
	srcPath := filepath.Join(scenariosPath, scenario, "ansible", "inventory.tmpl")
	dstPath := filepath.Join(tmpDir, "inventory")

	// Copy the file using io.Copy
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer src.Close()

	// Read the entire file content
	content, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %v", err)
	}

	// Convert to string for replacement
	tmpl := string(content)

	serverIps := []string{}

	// Replace variables in the template with values from tfstate
	for key, output := range data.Outputs {
		if strings.HasPrefix(key, "host_") {
			serverIps = append(serverIps, output.Value)
		}
		tmpl = strings.Replace(tmpl, key, output.Value, -1)
	}

	log.Debug().Msgf("Modified content: %s", tmpl)

	// Create the destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dst.Close()

	// Write the modified content to the destination file
	if _, err := dst.Write([]byte(tmpl)); err != nil {
		return fmt.Errorf("failed to write modified content: %v", err)
	}

	if err := WaitForSSH(serverIps, 60*time.Second); err != nil {
		return fmt.Errorf("failed to wait for SSH: %v", err)
	}

	if err := AnsibleContainer(ctx, containerOptions, scenario, dstPath); err != nil {
		return fmt.Errorf("failed to launch ansible container: %v", err)
	}
	log.Debug().Msg("Scenario configured with Ansible successfully")

	log.Info().Msgf("deployed infrastructure: %s", scenario)

	log.Info().Msgf("Entrypoint:\n%s", data.Outputs["entrypoint"].Value)
	return nil
}

type Output struct {
	Value     string `json:"value"`
	Type      string `json:"type"`
	Sensitive bool   `json:"sensitive"`
}

type Outputs struct {
	Main Output `json:"main"`
}

type Data struct {
	Outputs Outputs `json:"outputs"`
}

// listScenarios lists all the scenarios in the scenarios directory
func listScenarios(path string) ([]string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %v", err)
	}

	var scenarios []string
	for _, file := range files {
		if file.IsDir() {
			scenarios = append(scenarios, file.Name())
		}
	}

	return scenarios, nil
}

// LaunchInitContainer launches the init container that initializes the shared Terraform configuration
func LaunchInitContainer(ctx context.Context, options types.ContainerOptions) error {
	debug, _ := ctx.Value("debug").(bool)

	containerName := "vmgoat-terraform-base-init"

	err := handler.LaunchContainer(ctx, handler.ContainerConfig{
		Image: "hashicorp/terraform:latest",
		Name:  containerName,
		Args: []string{
			"init",
			"--reconfigure",
			"--upgrade",
		},
		WorkingDir: "/mnt/base/aws",
		Volumes: []handler.VolumeMount{
			{
				// TODO switch this to by in the current working directory
				// Source:      filepath.Join(currentDir, "base", "aws"),
				Source:      "/Users/aaiken/Private/vmGoat/base/aws",
				Destination: "/mnt/base/aws",
				ReadOnly:    false,
			},
			{
				Source:      filepath.Join(options.ConfigDir, "state"),
				Destination: "/mnt/state",
				ReadOnly:    false,
			},
		},
		// If debug is enabled, the container will not be automatically removed
		AutoRemove: !debug,
	})

	if err != nil {
		return fmt.Errorf("Failed to launch container: %s", err)
	}

	// Get and print container logs
	err = handler.GetContainerLogs(ctx, containerName)
	if err != nil {
		return fmt.Errorf("Failed to get container logs: %s", err)
	}
	return nil
}

// LaunchBaseContainer launches the base container that deploys infrastructure shared across all scenarios
func LaunchBaseContainer(ctx context.Context, options types.ContainerOptions, cmd string) error {
	debug, _ := ctx.Value("debug").(bool)

	containerName := fmt.Sprintf("vmgoat-terraform-base-%s", cmd)

	allowlistStrings := make([]string, len(options.Allowlist))
	for i, ip := range options.Allowlist {
		allowlistStrings[i] = fmt.Sprintf("\"%s\"", ip.String())
	}
	allowlistString := "[" + strings.Join(allowlistStrings, ", ") + "]"

	log.Debug().Msgf("Allowlist: %s", allowlistString)

	err := handler.LaunchContainer(ctx, handler.ContainerConfig{
		Image: "hashicorp/terraform:latest",
		Name:  containerName,
		Environment: []string{
			"TF_VAR_aws_profile=" + options.AwsProfile,
			"TF_VAR_aws_region=" + options.AwsRegion,
			"TF_VAR_allowlist=" + allowlistString,
		},
		Args: []string{
			cmd,
			"--auto-approve",
		},
		WorkingDir: "/mnt/base/aws",
		Volumes: []handler.VolumeMount{
			{
				Source:      filepath.Join(options.HomeDir, ".aws"),
				Destination: "/mnt/aws",
				ReadOnly:    true,
			},
			{
				// TODO switch this to by in the current working directory
				// Source:      filepath.Join(currentDir, "base", "aws"),
				Source:      "/Users/aaiken/Private/vmGoat/base/aws",
				Destination: "/mnt/base/aws",
				ReadOnly:    false,
			},
			{
				Source:      filepath.Join(options.ConfigDir, "state"),
				Destination: "/mnt/state",
				ReadOnly:    false,
			},
			{
				Source:      filepath.Join(options.ConfigDir, "ssh"),
				Destination: "/mnt/ssh",
				ReadOnly:    false,
			},
		},
		// If debug is enabled, the container will not be automatically removed
		AutoRemove: !debug,
	})

	if err != nil {
		return fmt.Errorf("Failed to launch container: %s", err)
	}

	// Get and print container logs
	err = handler.GetContainerLogs(ctx, containerName)
	if err != nil {
		return fmt.Errorf("Failed to get container logs: %s", err)
	}
	return nil
}

// LaunchInitScenarioContainer launches the init container that initializes a individual scenario
func LaunchInitScenarioContainer(ctx context.Context, options types.ContainerOptions, scenario string) error {
	debug, _ := ctx.Value("debug").(bool)

	containerName := fmt.Sprintf("vmgoat-terraform-scenario-%s-init", scenario)

	err := handler.LaunchContainer(ctx, handler.ContainerConfig{
		Image: "hashicorp/terraform:latest",
		Name:  containerName,
		Args: []string{
			"init",
			"--reconfigure",
			"--upgrade",
		},
		WorkingDir: fmt.Sprintf("/mnt/scenario/%s", scenario),
		Volumes: []handler.VolumeMount{
			{
				// TODO switch this to by in the current working directory
				// Source:      filepath.Join(currentDir, "base", "aws"),
				Source:      fmt.Sprintf("/Users/aaiken/Private/vmGoat/scenarios/%s/terraform", scenario),
				Destination: fmt.Sprintf("/mnt/scenario/%s", scenario),
				ReadOnly:    false,
			},
			{
				Source:      filepath.Join(options.ConfigDir, "state", "scenario", scenario),
				Destination: "/mnt/state",
				ReadOnly:    false,
			},
		},
		// If debug is enabled, the container will not be automatically removed
		AutoRemove: !debug,
	})

	if err != nil {
		return fmt.Errorf("Failed to launch container: %s", err)
	}

	// Get and print container logs
	err = handler.GetContainerLogs(ctx, containerName)
	if err != nil {
		return fmt.Errorf("Failed to get container logs: %s", err)
	}
	return nil
}

// LaunchScenarioContainer launches the scenario container that deploys its infrastructure
func LaunchScenarioContainer(ctx context.Context, options types.ContainerOptions, cmd string, scenario string) error {
	debug, _ := ctx.Value("debug").(bool)

	containerName := fmt.Sprintf("vmgoat-terraform-scenario-%s-%s", scenario, cmd)

	err := handler.LaunchContainer(ctx, handler.ContainerConfig{
		Image: "hashicorp/terraform:latest",
		Name:  containerName,
		Environment: []string{
			"TF_VAR_aws_profile=" + options.AwsProfile,
			"TF_VAR_aws_region=" + options.AwsRegion,
		},
		Args: []string{
			cmd,
			"--auto-approve",
		},
		WorkingDir: fmt.Sprintf("/mnt/scenario/%s", scenario),
		Volumes: []handler.VolumeMount{
			{
				Source:      filepath.Join(options.HomeDir, ".aws"),
				Destination: "/mnt/aws",
				ReadOnly:    true,
			},
			{
				// TODO switch this to by in the current working directory
				// Source:      filepath.Join(currentDir, "base", "aws"),
				Source:      fmt.Sprintf("/Users/aaiken/Private/vmGoat/scenarios/%s/terraform", scenario),
				Destination: fmt.Sprintf("/mnt/scenario/%s", scenario),
				ReadOnly:    false,
			},
			{
				Source:      filepath.Join(options.ConfigDir, "state", "scenario", scenario),
				Destination: "/mnt/state",
				ReadOnly:    false,
			},
		},
		// If debug is enabled, the container will not be automatically removed
		AutoRemove: !debug,
	})

	if err != nil {
		return fmt.Errorf("Failed to launch container: %s", err)
	}

	// Get and print container logs
	err = handler.GetContainerLogs(ctx, containerName)
	if err != nil {
		return fmt.Errorf("Failed to get container logs: %s", err)
	}
	return nil
}

func validateScenario(scenario string, scenariosPath string) bool {
	invalidScenario := false

	if scenario == "" {
		log.Warn().Msg("Deployment name is required")
		invalidScenario = true
	}

	scenarios, _ := listScenarios(scenariosPath)

	if !slices.Contains(scenarios, scenario) {
		invalidScenario = true
	}

	if invalidScenario {
		log.Info().Msgf("Available scenarios:")

		for _, s := range scenarios {
			log.Info().Msgf("  %s", s)
		}
		return false
	}
	return true
}

// AnsibleContainer runs the ansible playbook for the scenario
func AnsibleContainer(ctx context.Context, options types.ContainerOptions, scenario string, inventoryPath string) error {
	debug, _ := ctx.Value("debug").(bool)

	containerName := fmt.Sprintf("vmgoat-ansible-%s", scenario)

	// TODO: Have this be dynamic
	path := "/Users/aaiken/Private/vmGoat/"

	err := handler.LaunchContainer(ctx, handler.ContainerConfig{
		Image: "alpine/ansible:2.18.1",
		Name:  containerName,
		Entrypoint: []string{
			"sh",
			"-c",
		},
		Args: []string{
			"ansible-galaxy install -r requirements.yaml && ansible-playbook playbook.yaml",
		},
		// Environment: []string{
		// 	"TF_VAR_aws_profile=" + options.AwsProfile,
		// },
		WorkingDir: "/mnt/ansible",
		Volumes: []handler.VolumeMount{
			{
				Source:      filepath.Join(path, "scenarios", scenario, "ansible"),
				Destination: "/mnt/ansible",
				ReadOnly:    true,
			},
			{
				Source:      filepath.Join(path, "ansible.cfg"),
				Destination: "/etc/ansible/ansible.cfg",
				ReadOnly:    true,
			},
			{
				Source:      inventoryPath,
				Destination: "/mnt/inventory",
				ReadOnly:    true,
			},
			{
				Source:      filepath.Join(options.ConfigDir, "ssh"),
				Destination: "/mnt/ssh",
				ReadOnly:    true,
			},
		},
		// If debug is enabled, the container will not be automatically removed
		AutoRemove: !debug,
	})

	if err != nil {
		return fmt.Errorf("Failed to launch container: %s", err)
	}

	// Get and print container logs
	err = handler.GetContainerLogs(ctx, containerName)
	if err != nil {
		return fmt.Errorf("Failed to get container logs: %s", err)
	}
	return nil
}

// WaitForSSH waits for SSH connectivity to be available on all provided IP addresses
func WaitForSSH(ips []string, timeout time.Duration) error {
	// Create a channel to receive results from goroutines
	type result struct {
		ip    string
		error error
	}
	results := make(chan result, len(ips))

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Launch a goroutine for each IP
	for _, ip := range ips {
		log.Debug().Msgf("Waiting for SSH on %s", ip)
		go func(ip string) {
			for {
				select {
				case <-ctx.Done():
					results <- result{ip: ip, error: fmt.Errorf("timeout waiting for SSH on %s", ip)}
					return
				default:
					// Try to establish TCP connection to port 22
					conn, err := net.DialTimeout("tcp", ip+":22", 5*time.Second)
					if err == nil {
						conn.Close()
						results <- result{ip: ip, error: nil}
						return
					}
					// Wait a bit before trying again
					time.Sleep(2 * time.Second)
				}
			}
		}(ip)
	}

	// Collect results
	var errors []string
	for i := 0; i < len(ips); i++ {
		result := <-results
		if result.error != nil {
			errors = append(errors, result.error.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to connect to some IPs: %s", strings.Join(errors, "; "))
	}

	return nil
}
