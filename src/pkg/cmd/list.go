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
)

func List(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	configDir, _ := ctx.Value("configDirectory").(string)

	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	deployedScenarios := listDeployedScenarios(config)

	if cli.Bool("deployed") {
		// If no deployed scenarios are found, print a message
		if len(deployedScenarios) == 0 {
			log.Info().Msgf("No deployed scenarios found.")
		} else {
			log.Info().Msgf("Deployed scenarios:")
			for _, s := range deployedScenarios {
				log.Info().Msgf(" - %s", s)
			}
		}
	} else {
		scenariosPath := filepath.Join("/Users/aaiken/Private/vmGoat/", "scenarios")

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
	}
	return nil
}
