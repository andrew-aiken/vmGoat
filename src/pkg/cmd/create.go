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

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/execute/workflow"
	galaxy "github.com/apenella/go-ansible/v2/pkg/galaxy/role/install"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
	"infrasec.sh/vmGoat/pkg/types"
)

// Create handles the create command
func Create(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	containerized := cli.Bool("containerized")
	localExecution := cli.Bool("local")

	// Get user's home directory for AWS credentials
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	if !(containerized || localExecution) {
		return handler.LaunchContainerizedVersion(ctx, cli, homeDir)
	}

	projectPath := "/mnt"
	if !containerized {
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
	}

	scenariosPath := filepath.Join(projectPath, "scenarios")

	// TODO: Error if not scenarios are found
	scenario := cli.Args().First()
	if !validateScenario(scenario, scenariosPath) {
		log.Info().Msgf("\nUsage: %s", cli.UsageText)
		return nil
	}

	// Read the config directory from the context
	configDir, _ := ctx.Value("configDirectory").(string)

	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

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

	config.Scenarios[scenario] = types.Scenario{
		Provider: "aws",
		Path:     filepath.Join(projectPath, "scenarios", scenario),
	}

	// Set AWS paths depending if running inside a container or not
	awsConfigPath := filepath.Join(homeDir, ".aws", "config")
	awsCredentialsPath := filepath.Join(homeDir, ".aws", "credentials")

	if containerized && !localExecution {
		awsConfigPath = filepath.Join("/mnt/aws", "config")
		awsCredentialsPath = filepath.Join("/mnt/aws", "credentials")
	}

	// Setup the configurations that are passed when deploying the Terraform
	terraformOptions := types.TerraformOptions{
		Allowlist:              config.IpAddresses,
		AWSConfigPath:          awsConfigPath,
		AWSCredentialsPath:     awsCredentialsPath,
		AwsProfile:             awsProfile,
		AwsRegion:              awsRegion,
		ConfigDir:              configDir,
		Destroy:                false,
		TerraformCodePath:      filepath.Join(projectPath, "base", "aws"),
		TerraformVersion:       "1.12.0",
		TerraformStateFilePath: filepath.Join(configDir, "state", "terraform.tfstate"),
	}

	log.Info().Msg("Deploying base infrastructure")
	tf, err := initializeTerraform(ctx, terraformOptions)
	if err != nil {
		return fmt.Errorf("Failed to initialize the base Terraform: %v", err)
	}

	err = applyTerraform(ctx, tf, terraformOptions)
	if err != nil {
		log.Fatal().Msgf("Error applying Terraform: %s", err)
	}

	log.Debug().Msg("Base resources successfully deployed")

	// Deploy Scenario
	log.Info().Msg("Deploying scenario infrastructure")

	// Update the Terraform path options for the scenario
	terraformOptions.TerraformCodePath = filepath.Join(scenariosPath, scenario, "terraform")
	terraformOptions.TerraformStateFilePath = filepath.Join(configDir, "state", "scenario", scenario, "terraform.tfstate")

	tf, err = initializeTerraform(ctx, terraformOptions)
	if err != nil {
		return fmt.Errorf("Failed to initialize the base Terraform: %v", err)
	}

	err = applyTerraform(ctx, tf, terraformOptions)
	if err != nil {
		log.Fatal().Msgf("Error applying Terraform: %s", err)
	}

	log.Debug().Msg("Scenario resources successfully deployed")

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}
	log.Debug().Msg("Config updated successfully with scenario")

	ansiblePath := filepath.Join(scenariosPath, scenario, "ansible")

	// Create a temporary directory for Ansible files
	temporaryDirectory, err := os.MkdirTemp("", "vmgoat-ansible-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	log.Debug().Msgf("Temporary directory created: %s", temporaryDirectory)

	inventoryPath, entrypoint, serverIps, err := generateAnsibleInventory(types.AnsibleInventoryOptions{
		ScenarioAnsiblePath: ansiblePath,
		ScenarioStatePath:   terraformOptions.TerraformStateFilePath,
		TemporaryDirPath:    temporaryDirectory,
	})
	if err != nil {
		return fmt.Errorf("failed to generate Ansible inventory: %v", err)
	}

	if err := waitForSSH(serverIps, 60*time.Second); err != nil {
		return fmt.Errorf("failed to wait for SSH: %v", err)
	}

	err = runAnsible(types.AnsibleOptions{
		AnsiblePath:   ansiblePath,
		ConfigDir:     configDir,
		InventoryPath: inventoryPath,
	})
	if err != nil {
		return fmt.Errorf("failed to run Ansible playbook: %v", err)
	}

	log.Info().Msgf("Entrypoint: %s", entrypoint)

	return nil
}

func generateAnsibleInventory(options types.AnsibleInventoryOptions) (inventory string, entrypoint string, ips []string, error error) {
	file, err := os.Open(options.ScenarioStatePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Parse the raw structure into a map
	var data struct {
		Outputs map[string]types.TerraformOutput `json:"outputs"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		panic(err)
	}

	// Copy inventory template to temp directory
	srcPath := filepath.Join(options.ScenarioAnsiblePath, "inventory.tmpl")
	inventoryPath := filepath.Join(options.TemporaryDirPath, "inventory")

	// Copy the file using io.Copy
	src, err := os.Open(srcPath)
	if err != nil {
		return "", "", []string{}, fmt.Errorf("failed to open source file: %v", err)
	}
	defer src.Close()

	// Read the entire file content
	content, err := io.ReadAll(src)
	if err != nil {
		return "", "", []string{}, fmt.Errorf("failed to read source file: %v", err)
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
	dst, err := os.Create(inventoryPath)
	if err != nil {
		return "", "", []string{}, fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dst.Close()

	// Write the modified content to the destination file
	if _, err := dst.Write([]byte(tmpl)); err != nil {
		return "", "", []string{}, fmt.Errorf("failed to write modified content: %v", err)
	}

	return inventoryPath, data.Outputs["entrypoint"].Value, serverIps, nil
}

func runAnsible(options types.AnsibleOptions) error {
	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		Become:        true,
		Inventory:     options.InventoryPath,
		SSHCommonArgs: "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
		ExtraVars: map[string]interface{}{
			"ansible_ssh_private_key_file": filepath.Join(options.ConfigDir, "output", "ssh", "id_rsa"),
		},
	}

	timeoutContext, cancel := context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()

	playbookCmd := playbook.NewAnsiblePlaybookExecute(filepath.Join(options.AnsiblePath, "playbook.yaml")).
		WithPlaybookOptions(ansiblePlaybookOptions)

	// Check if requirements.yaml exists before trying to install roles
	requirementsPath := filepath.Join(options.AnsiblePath, "requirements.yaml")
	var err error
	if _, statErr := os.Stat(requirementsPath); statErr == nil {
		// Requirements file exists, install roles first
		galaxyInstallRolesCmd := galaxy.NewAnsibleGalaxyRoleInstallCmd(
			galaxy.WithGalaxyRoleInstallOptions(&galaxy.AnsibleGalaxyRoleInstallOptions{
				Force:    true,
				RoleFile: requirementsPath,
			}),
		)

		galaxyInstallRolesExec := execute.NewDefaultExecute(
			execute.WithCmd(galaxyInstallRolesCmd),
		)

		err = workflow.NewWorkflowExecute(galaxyInstallRolesExec, playbookCmd).
			WithTrace().
			Execute(timeoutContext)
	} else {
		// No requirements file, just run the playbook
		err = playbookCmd.Execute(timeoutContext)
	}

	if err != nil {
		return err
	}

	return nil
}

func initializeTerraform(ctx context.Context, options types.TerraformOptions) (*tfexec.Terraform, error) {
	log.Debug().Msg("Initializing Terraform")
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion(options.TerraformVersion)),
	}

	execPath, err := installer.Install(ctx)
	if err != nil {
		log.Fatal().Msgf("error installing Terraform: %s", err)
	}

	tf, err := tfexec.NewTerraform(options.TerraformCodePath, execPath)
	if err != nil {
		log.Fatal().Msgf("error setting up go terraform client: %s", err)
	}

	debug, _ := ctx.Value("debug").(bool)
	if debug {
		tf.SetStdout(os.Stdout)
		tf.SetStderr(os.Stderr)
	}

	err = tf.Init(
		ctx,
		tfexec.Upgrade(true),
		tfexec.Reconfigure(true),
	)

	if err != nil {
		log.Fatal().Msgf("error running Init: %s", err)
	}

	return tf, nil
}

// Converts a list of IPs to a json list string of the IPs
func generateAllowlistString(allowlist []net.IP) string {
	allowlistStrings := make([]string, len(allowlist))
	for i, ip := range allowlist {
		allowlistStrings[i] = fmt.Sprintf("\"%s\"", ip.String())
	}
	allowlistString := "[" + strings.Join(allowlistStrings, ", ") + "]"

	log.Debug().Msgf("Allowlist: %s", allowlistString)
	return allowlistString
}

func applyTerraform(ctx context.Context, tf *tfexec.Terraform, options types.TerraformOptions) error {
	log.Debug().Msg("Applying Terraform")

	allowlist := generateAllowlistString(options.Allowlist)

	os.Setenv("AWS_CONFIG_FILE", options.AWSConfigPath)
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", options.AWSCredentialsPath)
	os.Setenv("AWS_PROFILE", options.AwsProfile)
	os.Setenv("AWS_REGION", options.AwsRegion)
	// Variables to be used in the Terraform
	os.Setenv("TF_VAR_allowlist", allowlist)
	os.Setenv("TF_VAR_output_path", filepath.Join(options.ConfigDir, "output"))

	err := tf.Apply(
		ctx,
		tfexec.Refresh(true),
		tfexec.State(options.TerraformStateFilePath), // TODO: See if there is a modern way to do this
		tfexec.StateOut(options.TerraformStateFilePath),
		tfexec.Destroy(options.Destroy),
	)

	if err != nil {
		return err
	}
	return nil
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

// WaitForSSH waits for SSH connectivity to be available on all provided IP addresses
func waitForSSH(ips []string, timeout time.Duration) error {
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
