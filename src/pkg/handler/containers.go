package handler

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/andrew-aiken/vmGoat/pkg/logger"
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
	WorkingDir  string // Working directory for the container
	AutoRemove  bool
	Entrypoint  []string // Optional entrypoint override
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

// PullImage pulls a Docker image with progress reporting
func (d *DockerContainer) PullImage(ctx context.Context, imageName string) error {
	log := logger.Get()
	log.Info().Str("image", imageName).Msg("Pulling Docker image")

	// Pull the image
	reader, err := d.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %v", err)
	}
	defer reader.Close()

	// Read the pull progress
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("error while pulling image: %v", err)
	}

	log.Info().Str("image", imageName).Msg("Successfully pulled Docker image")
	return nil
}

// EnsureImageExists checks if the image exists locally and pulls it if necessary
func (d *DockerContainer) EnsureImageExists(ctx context.Context, image string) error {
	// Check if image exists locally
	_, err := d.client.ImageInspect(ctx, image)
	if err == nil {
		return nil // Image exists locally
	}

	if errdefs.IsNotFound(err) {
		// Image doesn't exist, pull it
		return d.PullImage(ctx, image)
	}

	return fmt.Errorf("error checking image: %v", err)
}

// Launch starts a new container with the specified configuration
func (d *DockerContainer) Launch(ctx context.Context, config ContainerConfig) error {
	log := logger.Get()

	// Ensure the image exists
	if err := d.EnsureImageExists(ctx, config.Image); err != nil {
		return fmt.Errorf("failed to ensure image exists: %v", err)
	}

	// Convert volume mounts to Docker format
	binds := make([]string, 0, len(config.Volumes))
	for _, volume := range config.Volumes {
		volumeStr := fmt.Sprintf("%s:%s", volume.Source, volume.Destination)
		if volume.ReadOnly {
			volumeStr += ":ro"
		}
		binds = append(binds, volumeStr)
	}

	// Create container configuration
	containerConfig := &container.Config{
		Image:      config.Image,
		Cmd:        config.Args,
		Env:        config.Environment,
		WorkingDir: config.WorkingDir,
		Entrypoint: config.Entrypoint,
	}

	// Create host configuration
	hostConfig := &container.HostConfig{
		Binds: binds,
		// PortBindings: portBindings,
		AutoRemove: config.AutoRemove,
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
	if err := d.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	log.Debug().
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
	return d.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true,
	})
}

// GetLogs retrieves container logs
func (d *DockerContainer) GetLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return d.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
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

// DeleteContainer is a helper function to delete a container by name
func DeleteContainer(ctx context.Context, containerName string) error {
	docker, err := NewDockerContainer()
	if err != nil {
		return err
	}

	// Get container ID by name
	containers, err := docker.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	var containerID string
	for _, container := range containers {
		for _, name := range container.Names {
			if name == "/"+containerName {
				containerID = container.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		return fmt.Errorf("container %s not found", containerName)
	}

	// Stop and remove the container
	if err := docker.Stop(ctx, containerID); err != nil {
		return fmt.Errorf("failed to stop container: %v", err)
	}

	return docker.Remove(ctx, containerID)
}

// GetContainerLogs is a helper function to get container logs by name
func GetContainerLogs(ctx context.Context, containerName string) error {
	docker, err := NewDockerContainer()
	if err != nil {
		return err
	}

	// Get container ID by name
	containers, err := docker.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	var containerID string
	for _, container := range containers {
		for _, name := range container.Names {
			if name == "/"+containerName {
				containerID = container.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		return fmt.Errorf("container %s not found", containerName)
	}

	// Get logs without timestamps
	logs, err := docker.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return fmt.Errorf("failed to get container logs: %v", err)
	}
	defer logs.Close()

	// Demultiplex Docker logs to remove binary headers and properly separate stdout/stderr
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
	if err != nil {
		return fmt.Errorf("failed to stream container logs: %v", err)
	}

	return nil
}
