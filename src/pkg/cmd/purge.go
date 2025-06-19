package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/andrew-aiken/vmGoat/pkg/handler"
	"github.com/andrew-aiken/vmGoat/pkg/logger"
	"github.com/andrew-aiken/vmGoat/pkg/types"
)

// Purge handles the purge command
func Purge(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	// Read the config directory from the context.
	configDir, _ := ctx.Value("configDirectory").(string)

	projectPath, _ := ctx.Value("projectPath").(string)
	scenariosPath := filepath.Join(projectPath, "scenarios")

	config, err := handler.ValidateConfigInitiator(types.ValidateConfigInputs{
		CliInputs: types.CliInputs{
			AwsProfile: cli.String("aws-profile"),
			AwsRegion:  cli.String("aws-region"),
		},
		ConfigDirectory: configDir,
	})

	deployedScenarios := listDeployedScenarios(config.Config)

	if !(cli.Bool("auto-approve") || approveDestruction(log, deployedScenarios, "purge")) {
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

	awsConfigPath, awsCredentialsPath := handler.AwsPathLocation(homeDir, (containerized && !localExecution))

	// Setup the configurations that are passed when deploying the Terraform
	terraformOptions := types.TerraformOptions{
		Allowlist:          config.IpAddresses,
		AWSConfigPath:      awsConfigPath,
		AWSCredentialsPath: awsCredentialsPath,
		AwsProfile:         config.CliInputs.AwsProfile,
		AwsRegion:          config.CliInputs.AwsRegion,
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

		delete(config.Config.Scenarios, scenario)
	}

	if err := handler.WriteConfig(configDir, config.Config); err != nil {
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

// Prompt the user for approval before proceeding
// List the deployed scenario[s] and the action being taken
func approveDestruction(log logger.Logger, scenarios []string, action string) bool {
	var approveInput string

	displayDeployedScenarios(log, scenarios)

	log.Debug().Msgf("Prompting for %s approval", action)
	fmt.Printf("Type 'Yes' to confirm that you want to %s: ", action)
	fmt.Scanln(&approveInput)

	if approveInput != "Yes" {
		log.Warn().Msgf("%s not approved. Exiting.", cases.Title(language.English, cases.Compact).String(action))
		return false
	}
	return true
}
