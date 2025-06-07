package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/fatih/color"
	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
	"infrasec.sh/vmGoat/pkg/types"
)

func List(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	configDir, _ := ctx.Value("configDirectory").(string)
	projectPath, _ := ctx.Value("projectPath").(string)

	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	deployedScenarios := listDeployedScenarios(config)

	if cli.Bool("deployed") {
		displayDeployedScenarios(log, deployedScenarios)
	} else {
		return displayAllScenarios(log, deployedScenarios, projectPath)
	}
	return nil
}

// Display a list of deployed scenarios based on the contents of the config file
func displayDeployedScenarios(log logger.Logger, scenarios []string) {
	// If no deployed scenarios are found, print a message
	if len(scenarios) == 0 {
		log.Info().Msgf("No scenarios deployed")
	} else {
		log.Info().Msgf("Deployed scenarios:")
		for _, s := range scenarios {
			log.Info().Msgf(" - %s", s)
		}
	}
}

// Display all the scenarios in the project directory, highlighting those that are deployed
func displayAllScenarios(log logger.Logger, deployedScenarios []string, projectPath string) error {
	scenariosPath := filepath.Join(projectPath, "scenarios")

	scenarios, err := listScenarios(scenariosPath)
	if err != nil {
		return fmt.Errorf("Failed to list scenarios: %s", err)
	}

	log.Info().Msgf("Available scenarios:")
	for _, s := range scenarios {
		if slices.Contains(deployedScenarios, s) {
			log.Info().Msgf(" - %s", color.RedString(s))
		} else {
			log.Info().Msgf(" - %s", s)
		}
	}
	return nil
}

func listDeployedScenarios(config types.Config) []string {
	var scenarios []string
	for scenario := range config.Scenarios {
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}
