package handler

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func LaunchContainerizedVersion(ctx context.Context, cli *cli.Command, homeDir string) error {
	// debug, _ := ctx.Value("debug").(bool)
	configDir, _ := ctx.Value("configDirectory").(string)

	containerName := cli.Root().Name

	err := LaunchContainer(ctx, ContainerConfig{
		Image:      "local/vmgoat",
		Name:       containerName,
		Entrypoint: []string{"/vmGoat"},
		Args:       append(cli.Root().Args().Slice(), "--containerized", "--auto-approve"),
		WorkingDir: "/",
		Volumes: []VolumeMount{
			{
				Source:      filepath.Join(homeDir, ".aws"),
				Destination: "/mnt/aws",
				ReadOnly:    true,
			},
			{
				Source:      configDir,
				Destination: "/mnt/config",
				ReadOnly:    false,
			},
		},
		// If debug is enabled, the container will not be automatically removed
		// TODO: Might be able to use the same container if the arguments are passed as a file and then dynamically read on startup (ie not a cli tool)
		AutoRemove: true, //!debug,
	})

	if err != nil {
		return fmt.Errorf("Failed to launch container: %s", err)
	}

	// Get and print container logs
	err = GetContainerLogs(ctx, containerName)
	if err != nil {
		return fmt.Errorf("Failed to get container logs: %s", err)
	}

	return nil
}
