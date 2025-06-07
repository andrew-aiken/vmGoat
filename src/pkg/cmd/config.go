package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
)

// ConfigAWS handles the aws configuration command
func ConfigAWS(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	// Read the config directory from the context.
	configDir, _ := ctx.Value("configDirectory").(string)

	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	// Read CLI flags
	config.AWS.Profile = cli.String("profile")
	config.AWS.Region = cli.String("region")

	// If flags are not set, prompt the user for input
	if config.AWS.Profile == "" {
		log.Debug().Msg("Prompting for AWS Profile")
		fmt.Print("AWS Profile: ")
		fmt.Scanln(&config.AWS.Profile)
	}

	if config.AWS.Region == "" {
		log.Debug().Msg("Prompting for AWS Region")
		fmt.Print("AWS Region: ")
		fmt.Scanln(&config.AWS.Region)
	}

	// TODO
	// Validate the AWS profile and region

	// Write the AWS settings to the config file
	if err := handler.WriteConfig(configDir, config); err != nil {
		log.Error().Err(err).Msg("Failed to write AWS config")
		return err
	}

	log.Info().Msg("AWS config successfully updated")
	return nil
}

// ConfigAllowlist handles the allowlist configuration command
func ConfigAllowlist(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	configDir, _ := ctx.Value("configDirectory").(string)
	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	// Remove the existing IP addresses, they will only be replaced if the new ones are valid
	config.IpAddresses = nil

	// Query the user for IP addresses to allowlist if none are provided
	if cli.Args().Len() == 0 {
		log.Info().Msg("No IPv4 addresses provided, automatically pulling IP from ifconfig.me")
		res, err := http.Get("https://ifconfig.me/ip")
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch IP from ifconfig.me")
			return err
		}

		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read response body from ifconfig.me")
			return fmt.Errorf("client: could not read response body: %s", err)
		}
		ip := net.ParseIP(string(resBody))
		if ip != nil {
			log.Info().Str("ip", ip.String()).Msg("Successfully retrieved IP address")
			config.IpAddresses = append(config.IpAddresses, ip)
		} else {
			log.Error().Str("ip", string(resBody)).Msg("Failed to parse IP address from response")
			return fmt.Errorf("invalid IP address received from ifconfig.me: %s", string(resBody))
		}
	}

	// If IP arguments are provided, validate them
	for _, ip := range cli.Args().Slice() {
		parsedIp := net.ParseIP(ip)
		if parsedIp == nil {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
		log.Info().Msg("Adding IP address to allowlist")
		config.IpAddresses = append(config.IpAddresses, parsedIp)
	}

	// Write the IP address(es) to the config file
	if err := handler.WriteConfig(configDir, config); err != nil {
		log.Error().Err(err).Msg("Failed to write allowlist config")
		return err
	}

	log.Info().Msg("Allowlist config successfully updated")
	return nil
}

func ConfigView(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	// Read the config directory from the context.
	// This should be under the home directory of the user. (`~/.config/vmgoat`)
	configDir, _ := ctx.Value("configDirectory").(string)

	config, err := handler.ReadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	yamlConfig, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	log.Info().Msg("Current configuration:\n")
	color.RGB(211, 211, 211).Println(string(yamlConfig))

	return nil
}
