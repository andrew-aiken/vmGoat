package flags

import "github.com/urfave/cli/v3"

var (
	// AWSProfile defines the AWS profile flag
	AWSProfile = &cli.StringFlag{
		Name:     "aws-profile",
		Required: false,
		OnlyOnce: true,
		Usage:    "Override the configured AWS profile",
		Sources:  cli.EnvVars("AWS_PROFILE"),
	}

	// AWSRegion defines the AWS region flag
	AWSRegion = &cli.StringFlag{
		Name:     "aws-region",
		Required: false,
		Value:    "us-east-1",
		OnlyOnce: true,
		Usage:    "Override the configured AWS region",
		Sources:  cli.EnvVars("AWS_REGION", "AWS_DEFAULT_REGION"),
	}

	// AutoApprove defines the auto-approve flag
	AutoApprove = &cli.BoolFlag{
		Name:     "auto-approve",
		Required: false,
		OnlyOnce: true,
		Usage:    "Automatically approve all actions without prompting",
	}

	// ConfigAWSProfile defines the AWS profile flag for config command
	ConfigAWSProfile = &cli.StringFlag{
		Name:     "profile",
		Required: false,
		OnlyOnce: true,
		Usage:    "Set the default AWS profile",
	}

	// ConfigAWSRegion defines the AWS region flag for config command
	ConfigAWSRegion = &cli.StringFlag{
		Name:     "region",
		Required: false,
		OnlyOnce: true,
		Usage:    "Set the default AWS region",
	}

	// Debug defines the debug flag
	Debug = &cli.BoolFlag{
		Name:     "debug",
		Aliases:  []string{"d"},
		Required: false,
		OnlyOnce: true,
		Usage:    "Display debug information",
	}
)
