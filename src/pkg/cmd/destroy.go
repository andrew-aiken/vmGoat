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

// Destroy handles the destroy command
func Destroy(ctx context.Context, cli *cli.Command) error {
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

	projectPath, _ := ctx.Value("projectPath").(string)
	scenariosPath := filepath.Join(projectPath, "scenarios")

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
		Destroy:                true,
		TerraformCodePath:      filepath.Join(scenariosPath, scenario, "terraform"),
		TerraformVersion:       "1.12.0",
		TerraformStateFilePath: filepath.Join(configDir, "state", "scenario", scenario, "terraform.tfstate"),
	}

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

	if err := handler.WriteConfig(configDir, config); err != nil {
		return err
	}

	return nil
}
