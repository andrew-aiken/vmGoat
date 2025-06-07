package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
	"infrasec.sh/vmGoat/pkg/types"
)

// Purge handles the purge command
func Purge(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	// Read the config directory from the context.
	configDir, _ := ctx.Value("configDirectory").(string)

	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	deployedScenarios := listDeployedScenarios(config)

	if !(cli.Bool("auto-approve") || approveDestruction(deployedScenarios)) {
		return nil
	}

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
	for _, scenario := range deployedScenarios {
		terraformOptions.TerraformCodePath = filepath.Join(scenariosPath, scenario, "terraform")
		terraformOptions.TerraformStateFilePath = filepath.Join(configDir, "state", "scenario", scenario, "terraform.tfstate")
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

func approveDestruction(scenarios []string) bool {
	var approveInput string

	if len(scenarios) != 0 {
		log.Info().Msgf("Deployed scenarios:")
		for _, s := range scenarios {
			log.Info().Msgf(" - %s", s)
		}
	}

	log.Debug().Msg("Prompting for purge approval")
	fmt.Print("Type 'Yes' to confirm that you want to destroy everything: ")
	fmt.Scanln(&approveInput)

	if approveInput != "Yes" {
		log.Warn().Msg("Purge not approved. Exiting.")
		return false
	}
	return true
}
