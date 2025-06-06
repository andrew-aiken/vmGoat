package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"infrasec.sh/vmGoat/pkg/cmd"
	"infrasec.sh/vmGoat/pkg/flags"
	"infrasec.sh/vmGoat/pkg/handler"
	"infrasec.sh/vmGoat/pkg/logger"
)

var (
	ProjectName = "vmGoat"
	Version     = "0.0.1"
)

func main() {
	app := &cli.Command{
		Name:        ProjectName,
		Version:     Version,
		Usage:       "Deploy insecure VMs to the cloud",
		Description: "foo",

		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Initialize logger with debug mode from flag
			logger.Init(logger.Config{
				Debug: cmd.Bool("debug"),
			})
			return handler.InitializeConfig(ctx, cmd)
		},

		Commands: []*cli.Command{
			{
				Name:      "create",
				Aliases:   []string{"c"},
				Usage:     "Create a new deployment",
				UsageText: "vmgoat create [scenario] [args]",
				Action:    cmd.Create,
				Flags: []cli.Flag{
					flags.AutoApprove,
					flags.AWSProfile,
					flags.AWSRegion,
				},
			},
			{
				Name:      "destroy",
				Aliases:   []string{"d"},
				Usage:     "Delete an existing deployment",
				UsageText: "vmgoat destroy [scenario] [args]",
				Action:    cmd.Destroy,
				Flags: []cli.Flag{
					flags.AutoApprove,
					flags.AWSProfile,
					flags.AWSRegion,
				},
			},
			{
				Name:    "list",
				Aliases: []string{"p"},
				Usage:   "List all scenarios",
				Action:  cmd.List,
				Flags: []cli.Flag{
					flags.DeployedScenarios,
				},
			},
			{
				Name:    "purge",
				Aliases: []string{"p"},
				Usage:   "Destroys all deployed infrastructure",
				Action:  cmd.Purge,
				Flags: []cli.Flag{
					flags.AutoApprove,
					flags.AWSProfile,
					flags.AWSRegion,
				},
			},
			{
				Name:  "config",
				Usage: "Configure persistent settings",
				Commands: []*cli.Command{
					{
						Name:        "aws",
						Usage:       "Setup AWS profile and region",
						Action:      cmd.ConfigAWS,
						Description: "Configure static AWS settings for vmGoat. These can still be overridden by the command line flags and environment variables.",
						Flags: []cli.Flag{
							flags.ConfigAWSProfile,
							flags.ConfigAWSRegion,
						},
					},
					{
						Name:        "allowlist",
						Usage:       "Specify IPs to allow to connect to the infrastructure",
						UsageText:   "vmgoat config allowlist [ip1] [ip2] [Nth]",
						Description: "Will automatically pull your public IP address and write it to the config file.\nAddresses can be specified as arguments to alow access from multiples IPs.",
						Action:      cmd.ConfigAllowlist,
					},
					{
						Name:      "view",
						Usage:     "Printouts out the current config",
						UsageText: "vmgoat config view",
						// Description: "",
						Action: cmd.ConfigView,
					},
				},
			},
		},
		Flags: []cli.Flag{
			flags.Containerized,
			flags.Debug,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log := logger.Get()
		log.Fatal().Err(err).Msg("Application error")
	}
}
