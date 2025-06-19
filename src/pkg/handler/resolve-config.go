package handler

import (
	"fmt"

	"github.com/andrew-aiken/vmGoat/pkg/logger"
)

// ResolveConfigValue handles the resolution of a value from command line flags, falling back to config,
// and updating the config if needed. Returns an error if the value is required but not found.
func ResolveConfigValue(flagValue *string, configValue *string) error {
	log := logger.Get()

	if *flagValue == "" {
		*flagValue = *configValue
		if *flagValue == "" {
			return fmt.Errorf("")
		}
	}

	if *configValue == "" {
		log.Debug().Msgf("Updating config with flag value: %s", *flagValue)
		*configValue = *flagValue
	}

	return nil
}
