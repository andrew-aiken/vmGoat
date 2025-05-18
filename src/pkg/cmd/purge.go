package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
	"infrasec.sh/vmGoat/pkg/types"
)

// Purge handles the purge command
func Purge(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	approve := cli.Bool("auto-approve")
	var approveInput string

	if approve == false {
		log.Debug().Msg("Prompting for purge approval")
		fmt.Print("Type 'Yes' to confirm that you want to destroy everything: ")
		fmt.Scanln(&approveInput)

		if approveInput != "Yes" {
			log.Warn().Msg("Purge not approved. Exiting.")
			return nil
		}
		approve = true
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
		ConfigDir:  configDir,
		HomeDir:    homeDir,
		AwsProfile: awsProfile,
		AwsRegion:  awsRegion,
	}

	// TODO
	for _, s := range listDeployedScenarios(config) {
		log.Info().Msgf("Destroying Scenario: %s", s)
		// Initialize the scenarios init container
		if err := LaunchInitScenarioContainer(ctx, containerOptions, s); err != nil {
			return fmt.Errorf("failed to launch init container: %v", err)
		}
		log.Debug().Msg("Scenario container successfully initialized")

		// Deploy the scenario
		if err := LaunchScenarioContainer(ctx, containerOptions, "destroy", s); err != nil {
			return fmt.Errorf("failed to launch base container: %v", err)
		}
		log.Debug().Msg("Scenario container successfully destroyed")

		delete(config.Scenarios, s)
	}

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}
	log.Debug().Msg("Removed scenarios from the config")

	if err := LaunchInitContainer(ctx, containerOptions); err != nil {
		return fmt.Errorf("failed to launch init container: %v", err)
	}

	if err := LaunchBaseContainer(ctx, containerOptions, "destroy"); err != nil {
		return fmt.Errorf("failed to launch base container: %v", err)
	}

	log.Info().Msg("Base Infrastructure destroyed successfully")

	return nil
}

func listDeployedScenarios(config types.Config) []string {
	var scenarios []string
	for scenario := range config.Scenarios {
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}
