package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
	"infrasec.sh/vmGoat/pkg/types"
)

// Create handles the create command
func Create(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	scenario := cli.Args().First()

	invalidScenario := false

	if scenario == "" {
		log.Warn().Msg("Deployment name is required")
		invalidScenario = true
	}

	scenariosDirs, _ := listScenarios("/Users/aaiken/Private/vmGoat/scenarios")

	if !slices.Contains(scenariosDirs, scenario) {
		invalidScenario = true
	}

	if invalidScenario {
		log.Info().Msgf("Usage: %s\n", cli.UsageText)

		log.Info().Msgf("Available scenarios:")

		for _, s := range scenariosDirs {
			log.Info().Msgf("  %s", s)
		}
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

	// TODO
	// Actually deploy & configure the infrastructure

	containerOptions := types.ContainerOptions{
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

	// TODO
	// Ansible

	log.Info().Msgf("deployed infrastructure: %s", scenario)
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
