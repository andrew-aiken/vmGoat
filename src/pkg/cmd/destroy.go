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

// Destroy handles the destroy command
func Destroy(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	scenariosPath := "/Users/aaiken/Private/vmGoat/scenarios"

	scenario := cli.Args().First()
	if !validateScenario(scenario, scenariosPath) {
		log.Info().Msgf("\nUsage: %s", cli.UsageText)
		return nil
	}

	approve := cli.Bool("auto-approve")
	var approveInput string

	if approve == false {
		log.Debug().Msg("Prompting for destroy approval")
		fmt.Printf("Type 'Yes' to confirm that you want to destroy the %s scenario: ", scenario)
		fmt.Scanln(&approveInput)

		if approveInput != "Yes" {
			log.Warn().Msg("Destroy not approved. Exiting.")
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

	log.Info().Msgf("Destroying Scenario: %s", scenario)
	// Initialize the scenarios init container
	if err := LaunchInitScenarioContainer(ctx, containerOptions, scenario); err != nil {
		return fmt.Errorf("failed to launch init container: %v", err)
	}
	log.Debug().Msg("Scenario container successfully initialized")

	// Deploy the scenario
	if err := LaunchScenarioContainer(ctx, containerOptions, "destroy", scenario); err != nil {
		return fmt.Errorf("failed to launch base container: %v", err)
	}
	log.Debug().Msg("Scenario container successfully destroyed")

	delete(config.Scenarios, scenario)

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}

	return nil
}
