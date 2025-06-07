package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// TODO
// Check if in a valid vmGoat file structure

// Validate the existence of the configuration file
func InitializeConfig(ctx context.Context, cli *cli.Command) (context.Context, error) {
	ctx = context.WithValue(ctx, "debug", cli.Root().Bool("debug"))

	ctx, configDirectory, err := setupConfigDirectory(ctx, cli.Root().Name, cli.Bool("containerized"))
	if err != nil {
		return ctx, fmt.Errorf("Error setting up config directory: %s", err)
	}

	if err := setupConfigfile(filepath.Join(configDirectory, "config.yaml")); err != nil {
		return ctx, fmt.Errorf("Error setting up config file: %s", err)
	}

	if err := setupStateDirectory(filepath.Join(configDirectory, "state")); err != nil {
		return ctx, fmt.Errorf("Error setting up state directory: %s", err)
	}

	return ctx, nil
}

func setupConfigDirectory(ctx context.Context, appName string, containerized bool) (context.Context, string, error) {
	homeDirName, err := os.UserHomeDir()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get home directory")
	}

	configPath := filepath.Join(homeDirName, "/.config", appName)
	if containerized {
		configPath = "/mnt/config"
	}

	ctx = context.WithValue(ctx, "configDirectory", configPath)

	if stat, err := os.Stat(configPath); err == nil {
		if !stat.IsDir() {
			return ctx, configPath, fmt.Errorf("file '%s' exists but is not a directory", configPath)
		}
		log.Debug().Msgf("Directory '%s' exists", configPath)
		// If the file does not exist
	} else if os.IsNotExist(err) {
		log.Debug().Msgf("Directory '%s' does not exist", configPath)
		// Make the directories
		if err := os.MkdirAll(configPath, 0755); err != nil {
			return ctx, configPath, fmt.Errorf("Error creating directory '%s': %v", configPath, err)
		}
	} else {
		return ctx, configPath, fmt.Errorf("Error checking file '%s'", configPath)
	}
	return ctx, configPath, nil
}

func setupConfigfile(filePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		log.Debug().Msgf("File '%s' exists", filePath)
	} else if os.IsNotExist(err) {
		log.Info().Msgf("File '%s' does not exist, initializing", filePath)
		_, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("Error creating file '%s': %v", filePath, err)
		} else {
			log.Debug().Msgf("File '%s' created", filePath)
		}
	} else {
		return fmt.Errorf("Error checking file '%s': %v", filePath, err)
	}
	return nil
}

func setupStateDirectory(path string) error {
	// Check if state directory exists
	if stat, err := os.Stat(path); err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("file '%s' exists but is not a directory", path)
		}
		log.Debug().Msgf("Directory '%s' exists", path)
		// If the file does not exist
	} else if os.IsNotExist(err) {
		log.Debug().Msgf("Directory '%s' does not exist", path)
		// Make the directories
		if err := os.Mkdir(path, 0755); err != nil {
			return fmt.Errorf("Error creating directory '%s': %v", path, err)
		}
	} else {
		return fmt.Errorf("Error checking file '%s'", path)
	}
	return nil
}
