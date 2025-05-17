package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
	"infrasec.sh/vmGoat/pkg/types"
)

// Create handles the create command
func Create(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()
	scenario := cli.Args().First()

	// var Debug = cli.Root().Bool("debug")

	if scenario == "" {
		log.Warn().Msg("Deployment name is required")
		log.Info().Msgf("Usage: %s", cli.UsageText)
		return nil
	}

	_ = validateScenario(scenario)

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

	if err := resolveConfigValue(&awsProfile, &config.AWS.Profile); err != nil {
		return fmt.Errorf("failed to resolve AWS profile: %v", err)
	}

	if err := resolveConfigValue(&awsRegion, &config.AWS.Region); err != nil {
		return fmt.Errorf("failed to resolve AWS profile: %v", err)
	}

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}

	log.Debug().Msg("Config updated successfully")

	// TODO
	// Actually deploy & configure the infrastructure

	LaunchBaseContainer(ctx, types.ContainerOptions{
		ConfigDir:  configDir,
		HomeDir:    homeDir,
		AwsProfile: awsProfile,
		AwsRegion:  awsRegion,
	})

	// err = handler.LaunchContainer(ctx, handler.ContainerConfig{
	// 	// Image: "hashicorp/terraform:latest",
	// 	Image: "terraform:local",
	// 	Name:  "vmgoat-terraform-base",
	// 	Environment: []string{
	// 		"TF_VAR_aws_profile=" + awsProfile,
	// 		"TF_VAR_aws_region=" + awsRegion,
	// 	},
	// 	Args: []string{
	// 		"init",
	// 	},
	// 	WorkingDir: "/mnt/base/aws",
	// 	Volumes: []handler.VolumeMount{
	// 		{
	// 			Source:      filepath.Join(homeDir, ".aws"),
	// 			Destination: "/mnt/aws",
	// 			ReadOnly:    true,
	// 		},
	// 		{
	// 			// TODO switch this to by in the current working directory
	// 			// Source:      filepath.Join(currentDir, "base", "aws"),
	// 			Source:      "/Users/aaiken/Private/vmGoat/base/aws",
	// 			Destination: "/mnt/base/aws",
	// 			ReadOnly:    false,
	// 		},
	// 		{
	// 			Source:      filepath.Join(configDir, "state"),
	// 			Destination: "/mnt/state",
	// 			ReadOnly:    false,
	// 		},
	// 	},
	// 	// If debug is enabled, the container will not be automatically removed
	// 	AutoRemove: Debug,
	// })

	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to launch container")
	// 	return err
	// }

	// // Get and print container logs
	// err = handler.GetContainerLogs(ctx, "vmgoat-terraform-base")
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to get container logs")
	// }

	log.Info().Msgf("deployed infrastructure: %s", scenario)
	return nil
}

// TODO
// Check if the scenario is a valid scenario in the local file system
// Move to a dedicated handler function
func validateScenario(name string) error {
	if name == "" {
		return fmt.Errorf("Scenario name is required")
	}
	return nil
}

// resolveConfigValue handles the resolution of a value from command line flags, falling back to config,
// and updating the config if needed. Returns an error if the value is required but not found.
func resolveConfigValue(flagValue *string, configValue *string) error {
	log := logger.Get()

	if *flagValue == "" {
		*flagValue = *configValue
		if *flagValue == "" {
			return fmt.Errorf("")
		}
	}

	if *configValue == "" {
		log.Debug().Msgf("Updating config with flag value: %s", *flagValue)
		*configValue = *flagValue
	}

	return nil
}

func LaunchBaseContainer(ctx context.Context, options types.ContainerOptions) error {
	debug, _ := ctx.Value("debug").(bool)

	err := handler.LaunchContainer(ctx, handler.ContainerConfig{
		// Image: "hashicorp/terraform:latest",
		Image: "terraform:local",
		Name:  "vmgoat-terraform-base",
		Environment: []string{
			"TF_VAR_aws_profile=" + options.AwsProfile,
			"TF_VAR_aws_region=" + options.AwsRegion,
		},
		Args: []string{
			"init",
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
	err = handler.GetContainerLogs(ctx, "vmgoat-terraform-base")
	if err != nil {
		return fmt.Errorf("Failed to get container logs: %s", err)
	}
	return nil
}
