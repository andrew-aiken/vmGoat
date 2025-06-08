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

	if !(cli.Bool("auto-approve") || approveDestruction(log, []string{scenario}, "destroy")) {
		return nil
	}

	// Read the config directory from the context
	configDir, _ := ctx.Value("configDirectory").(string)

	config, err := handler.ValidateConfigInitiator(types.ValidateConfigInputs{
		CliInputs: types.CliInputs{
			AwsProfile: cli.String("aws-profile"),
			AwsRegion:  cli.String("aws-region"),
		},
		ConfigDirectory: configDir,
	})

	awsConfigPath, awsCredentialsPath := handler.AwsPathLocation(homeDir, (containerized && !localExecution))

	// Setup the configurations that are passed when deploying the Terraform
	terraformOptions := types.TerraformOptions{
		Allowlist:              config.IpAddresses,
		AWSConfigPath:          awsConfigPath,
		AWSCredentialsPath:     awsCredentialsPath,
		AwsProfile:             config.CliInputs.AwsProfile,
		AwsRegion:              config.CliInputs.AwsRegion,
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

	delete(config.Config.Scenarios, scenario)

	if err := handler.WriteConfig(configDir, config.Config); err != nil {
		return err
	}

	return nil
}
