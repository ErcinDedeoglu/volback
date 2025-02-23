package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func executeCommand(cmdPath string, args ...string) ([]byte, error) {
	cmd := exec.Command(cmdPath, args...)
	logSubStep("⚙️  Executing: %s %v", cmdPath, args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logStep("❌ Command failed with output: %s", string(output))
		return nil, err
	}
	return output, nil
}

func getContainerVolumes(containerName string) (*VolumeResult, error) {
	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("error initializing Docker client: %v", err)
	}
	defer cli.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	volumeInfo := &VolumeResult{
		ContainerName: containerName,
		Volumes:       make([]Volume, 0),
	}

	container, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		volumeInfo.Status = "Failed"
		volumeInfo.Error = err.Error()
		return volumeInfo, nil
	}

	volumeInfo.Status = "Success"
	for _, mount := range container.Mounts {
		volume := Volume{
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        mount.Mode,
			RW:          mount.RW,
			Type:        string(mount.Type),
			Name:        mount.Name,
		}
		volumeInfo.Volumes = append(volumeInfo.Volumes, volume)
	}

	return volumeInfo, nil
}

func processContainer(containerName string, action string) (*ControlResult, error) {
	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("error initializing Docker client: %v", err)
	}
	defer cli.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := &ControlResult{
		Name:   containerName,
		Action: action,
	}

	// Create stop options with timeout in seconds
	timeoutSeconds := 10
	stopOptions := container.StopOptions{
		Timeout: &timeoutSeconds,
	}

	// Process container
	var actionErr error
	if action == "stop" {
		actionErr = cli.ContainerStop(ctx, containerName, stopOptions)
	} else {
		actionErr = cli.ContainerStart(ctx, containerName, container.StartOptions{})
	}

	if actionErr != nil {
		result.Status = "Failed"
		result.Error = actionErr.Error()
	} else {
		result.Status = "Success"
	}

	return result, nil
}

func stopDockerContainer(containerName string) error {
	result, err := processContainer(containerName, "stop")
	if err != nil || result.Status == "Failed" {
		logStep("❌ Failed to stop container: %v", err)
		return err
	}
	return nil
}

func startDockerContainer(containerName string) error {
	result, err := processContainer(containerName, "start")
	if err != nil || result.Status == "Failed" {
		logStep("❌ Failed to start container: %v", err)
		return err
	}
	return nil
}
