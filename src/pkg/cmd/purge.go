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

// Purge handles the purge command
func Purge(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	approve := cli.Bool("auto-approve")
	var approveInput string

	// TODO: list all deployed scenarios

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
	containerized := cli.Bool("containerized")

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

	projectPath := "/Users/aaiken/Private/vmGoat"
	scenariosPath := filepath.Join(projectPath, "scenarios")

	// Set AWS paths depending if running inside a container or not
	awsConfigPath := filepath.Join(homeDir, ".aws", "config")
	awsCredentialsPath := filepath.Join(homeDir, ".aws", "credentials")

	if containerized {
		awsConfigPath = filepath.Join("/mnt/aws", "config")
		awsCredentialsPath = filepath.Join("/mnt/aws", "credentials")
	}

	// Setup the configurations that are passed when deploying the Terraform
	terraformOptions := types.TerraformOptions{
		Allowlist:          config.IpAddresses,
		AWSConfigPath:      awsConfigPath,
		AWSCredentialsPath: awsCredentialsPath,
		AwsProfile:         awsProfile,
		AwsRegion:          awsRegion,
		ConfigDir:          configDir,
		Destroy:            true,
		TerraformVersion:   "1.12.0",
	}

	// Loop over scenarios and destroy them
	for _, scenario := range listDeployedScenarios(config) {
		terraformOptions.TerraformCodePath = filepath.Join(scenariosPath, scenario, "terraform")
		terraformOptions.TerraformStateFilePath = filepath.Join(configDir, "state", "scenario", scenario)
		log.Info().Msgf("Destroying Scenario: %s", scenario)
		tf, err := initializeTerraform(ctx, terraformOptions)
		if err != nil {
			return fmt.Errorf("Failed to initialized the scenario %s Terraform: %v", scenario, err)
		}

		err = applyTerraform(ctx, tf, terraformOptions)
		if err != nil {
			log.Fatal().Msgf("Error destroying the scenario %s: %s", scenario, err)
		}

		delete(config.Scenarios, scenario)
	}

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}
	log.Debug().Msg("Removed scenarios from the config")

	log.Info().Msg("Removing the base infrastructure")

	terraformOptions.TerraformCodePath = filepath.Join(projectPath, "base", "aws")
	terraformOptions.TerraformStateFilePath = filepath.Join(configDir, "state", "terraform.tfstate")

	tf, err := initializeTerraform(ctx, terraformOptions)
	if err != nil {
		return fmt.Errorf("Failed to initialized the base Terraform: %v", err)
	}

	err = applyTerraform(ctx, tf, terraformOptions)
	if err != nil {
		log.Fatal().Msgf("Error destroying the base Terraform: %s", err)
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
