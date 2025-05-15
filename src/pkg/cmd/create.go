package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
)

// Create handles the create command
func Create(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()
	scenario := cli.Args().First()

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

	// err = handler.LaunchContainer(ctx, handler.ContainerConfig{
	// 	Image:       "nginx",
	// 	Environment: []string{"ENV=production"},
	// 	Name:        "nginx-container",
	// 	Ports:       map[string]string{"8080": "80"},
	// })

	if err != nil {
		log.Error().Err(err).Msg("Failed to launch container")
	}

	log.Info().Msgf("deployed infrastructure: %s", scenario)
	return nil
}

// TODO
// Check if the scenario is a valid scenario in the local file system
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
