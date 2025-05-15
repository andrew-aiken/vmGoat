package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/logger"
)

// Validate the existence of the configuration file
func InitializeConfig(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	log := logger.Get()

	homeDirName, err := os.UserHomeDir()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get home directory")
	}

	configDirectory := fmt.Sprintf("%s/.config/%s", homeDirName, cmd.Root().Name)

	// Check if the file exists and is a directory
	if stat, err := os.Stat(configDirectory); err == nil {
		if !stat.IsDir() {
			return ctx, fmt.Errorf("file '%s' exists but is not a directory", configDirectory)
		}
		log.Debug().Msgf("Directory '%s' exists", configDirectory)
		// If the file does not exist
	} else if os.IsNotExist(err) {
		log.Debug().Msgf("Directory '%s' does not exist", configDirectory)
		// Make the directories
		if err := os.MkdirAll(configDirectory, 0755); err != nil {
			return ctx, fmt.Errorf("Error creating directory '%s': %v", configDirectory, err)
		}
	} else {
		return ctx, fmt.Errorf("Error checking file '%s'", configDirectory)
	}

	ctx = context.WithValue(ctx, "configDirectory", configDirectory)

	configFile := fmt.Sprintf("%s/config.yaml", configDirectory)

	if _, err := os.Stat(configFile); err == nil {
		log.Debug().Msgf("File '%s' exists", configFile)
	} else if os.IsNotExist(err) {
		log.Info().Msgf("File '%s' does not exist, initializing", configFile)
		_, err := os.Create(configFile)
		if err != nil {
			return ctx, fmt.Errorf("Error creating file '%s': %v", configFile, err)
		} else {
			log.Debug().Msgf("File '%s' created", configFile)
		}
	} else {
		return ctx, fmt.Errorf("Error checking file '%s': %v", configFile, err)
	}

	return ctx, nil
}
