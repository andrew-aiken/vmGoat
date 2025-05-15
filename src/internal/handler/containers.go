package handler

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"infrasec.sh/vmGoat/pkg/logger"
)

// ContainerRuntime represents a supported container runtime
type ContainerRuntime string

const (
	// DockerRuntime represents the Docker container runtime
	DockerRuntime ContainerRuntime = "docker"
	// PodmanRuntime represents the Podman container runtime
	PodmanRuntime ContainerRuntime = "podman"
)

// VolumeMount represents a volume mount configuration
type VolumeMount struct {
	Source      string
	Destination string
	ReadOnly    bool
}

// ContainerConfig holds the configuration for launching a container
type ContainerConfig struct {
	Image       string
	Volumes     []VolumeMount
	Args        []string
	Environment []string
	Name        string
	Ports       map[string]string // hostPort:containerPort
}

// DockerContainer manages Docker container operations
type DockerContainer struct {
	client *client.Client
}

// NewDockerContainer creates a new Docker container manager
func NewDockerContainer() (*DockerContainer, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %v", err)
	}
	return &DockerContainer{client: cli}, nil
}

// Launch starts a new container with the specified configuration
func (d *DockerContainer) Launch(ctx context.Context, config ContainerConfig) error {
	log := logger.Get()

	// Convert volume mounts to Docker format
	binds := make([]string, 0, len(config.Volumes))
	for _, volume := range config.Volumes {
		volumeStr := fmt.Sprintf("%s:%s", volume.Source, volume.Destination)
		if volume.ReadOnly {
			volumeStr += ":ro"
		}
		binds = append(binds, volumeStr)
	}

	// Convert port mappings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for hostPort, containerPort := range config.Ports {
		port := nat.Port(containerPort)
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}

	// Create container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Cmd:          config.Args,
		Env:          config.Environment,
		ExposedPorts: exposedPorts,
	}

	// Create host configuration
	hostConfig := &container.HostConfig{
		Binds:        binds,
		PortBindings: portBindings,
	}

	// Create the container
	resp, err := d.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		config.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Start the container
	if err := d.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	log.Info().
		Str("container_id", resp.ID).
		Str("name", config.Name).
		Msg("Container started successfully")

	return nil
}

// Stop stops a running container
func (d *DockerContainer) Stop(ctx context.Context, containerID string) error {
	timeout := 10 // seconds
	return d.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// Remove removes a container
func (d *DockerContainer) Remove(ctx context.Context, containerID string) error {
	return d.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

// GetLogs retrieves container logs
func (d *DockerContainer) GetLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return d.client.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
	})
}

// LaunchContainer is a helper function to launch a container with the specified configuration
func LaunchContainer(ctx context.Context, config ContainerConfig) error {
	docker, err := NewDockerContainer()
	if err != nil {
		return err
	}
	return docker.Launch(ctx, config)
}
