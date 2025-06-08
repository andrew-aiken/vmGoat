package handler

import (
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"infrasec.sh/vmGoat/pkg/types"
)

// Read the configuration and resolve differences in values
func ValidateConfigInitiator(configInput types.ValidateConfigInputs) (types.ValidateConfigInputs, error) {
	config, err := ReadConfig(configInput.ConfigDirectory)
	if err != nil {
		return types.ValidateConfigInputs{}, fmt.Errorf("failed to read config: %v", err)
	}

	outputConfig := types.ValidateConfigInputs{
		CliInputs: types.CliInputs{
			AwsProfile: configInput.CliInputs.AwsProfile,
			AwsRegion:  configInput.CliInputs.AwsRegion,
		},
		ConfigDirectory: configInput.ConfigDirectory,
		IpAddresses:     config.IpAddresses,
		Config:          config,
	}

	if err := ResolveConfigValue(&outputConfig.CliInputs.AwsProfile, &config.AWS.Profile); err != nil {
		return outputConfig, fmt.Errorf("failed to resolve AWS profile: %v", err)
	}

	if err := ResolveConfigValue(&outputConfig.CliInputs.AwsRegion, &config.AWS.Region); err != nil {
		return outputConfig, fmt.Errorf("failed to resolve AWS profile: %v", err)
	}

	if err := WriteConfig(configInput.ConfigDirectory, config); err != nil {
		return outputConfig, err
	}

	log.Debug().Msg("Config updated successfully")

	return outputConfig, nil
}

// Set AWS paths depending if running inside a container or not
func AwsPathLocation(homeDir string, containerized bool) (string, string) {
	awsConfigPath := filepath.Join(homeDir, ".aws", "config")
	awsCredentialsPath := filepath.Join(homeDir, ".aws", "credentials")

	if containerized {
		awsConfigPath = filepath.Join("/mnt/aws", "config")
		awsCredentialsPath = filepath.Join("/mnt/aws", "credentials")
	}

	return awsConfigPath, awsCredentialsPath
}
