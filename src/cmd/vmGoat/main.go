package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)

	// Register the channel to receive specific signals
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to handle signals
	go func() {
		sig := <-sigChan
		log := logger.Get()
		log.Debug().Msgf("Received signal: %v", sig)

		// Run your cleanup functions here
		cleanup(ctx)

		// Cancel the context to signal shutdown
		cancel()

		// Exit gracefully
		os.Exit(0)
	}()

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
				Aliases:   []string{"c", "deploy"},
				Usage:     "Create a new deployment",
				UsageText: "vmgoat create [scenario] [args]",
				Action:    cmd.Create,
				Flags: []cli.Flag{
					flags.AutoApprove,
					flags.AWSProfile,
					flags.AWSRegion,
					flags.Local,
				},
			},
			{
				Name:      "destroy",
				Aliases:   []string{"d", "delete"},
				Usage:     "Delete an existing deployment",
				UsageText: "vmgoat destroy [scenario] [args]",
				Action:    cmd.Destroy,
				Flags: []cli.Flag{
					flags.AutoApprove,
					flags.AWSProfile,
					flags.AWSRegion,
					flags.Local,
				},
			},
			{
				Name:   "list",
				Usage:  "List all scenarios",
				Action: cmd.List,
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
					flags.Local,
				},
			},
			{
				Name:  "config",
				Usage: "Configure persistent settings",
				Commands: []*cli.Command{
					{
						Name:        "aws",
						Usage:       "Set the AWS profile and region",
						Action:      cmd.ConfigAWS,
						Description: "Configure static AWS settings for vmGoat. These can still be overridden by the command line flags and environment variables.",
						Flags: []cli.Flag{
							flags.ConfigAWSProfile,
							flags.ConfigAWSRegion,
						},
					},
					{
						Name:        "allowlist",
						Usage:       "Specify IPv4 addresses to allow to connect to the infrastructure",
						UsageText:   "vmgoat config allowlist [ip1] [ip2] [Nth]",
						Description: "Will automatically pull your public IPv4 address and write it to the config file.\nAddresses can be specified as arguments to alow access from multiples IPs.",
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

	if err := app.Run(ctx, os.Args); err != nil {
		log := logger.Get()
		log.Fatal().Err(err).Msg("Application error")
	}
}

// cleanup runs any necessary cleanup operations before shutdown
func cleanup(ctx context.Context) {
	log := logger.Get()
	log.Info().Msg("Intercepted interruption signal, starting cleanup...")

	handler.DeleteContainer(ctx, "vmGoat")

	log.Info().Msg("Cleanup completed")
}
