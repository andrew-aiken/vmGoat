package handler

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"infrasec.sh/vmGoat/internal/types"
)

func ReadConfig(configPath string) (types.Config, error) {
	configFile := fmt.Sprintf("%s/config.yaml", configPath)

	// Read the YAML file
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return types.Config{}, fmt.Errorf("Error reading YAML file: %v\n", err)
	}

	// Unmarshal the YAML into the Config struct
	var config types.Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return types.Config{}, fmt.Errorf("Error unmarshalling YAML: %v\n", err)
	}

	return config, nil
}

func WriteConfig(configPath string, config types.Config) error {
	configFile := fmt.Sprintf("%s/config.yaml", configPath)

	// Marshal the Config struct to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("Error marshalling YAML: %v\n", err)
	}

	// Write the YAML data to the file
	err = os.WriteFile(configFile, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("Error writing YAML file: %v\n", err)
	}

	return nil
}
