package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func processVolumes(containerConfig ContainerConfig, volumes []Volume, outputDir string) error {
	tempDir := filepath.Join(outputDir, "temp", containerConfig.Container)
	os.MkdirAll(tempDir, 0755)

	// Ensure the latest version of the Packmate image is pulled
	if err := pullLatestPackmateImage(); err != nil {
		return fmt.Errorf("failed to pull the latest Packmate image: %w", err)
	}

	for i, volume := range volumes {
		logHeader("🔸 Volume %d/%d:", i+1, len(volumes))
		logSubStep("Source: %s", volume.Source)
		logSubStep("Destination: %s", volume.Destination)
		logSubStep("Type: %s", volume.Type)
		logSubStep("RW: %t", volume.RW)
		logSubStep("Mode: %s", volume.Mode)
		logSubStep("Name: %s", volume.Name)

		// Skip tmpfs volumes
		if volume.Type == "tmpfs" {
			logSubStep("⏭️  Skipping tmpfs volume")
			continue
		}

		// Skip volumes with empty source
		if volume.Source == "" {
			logSubStep("⏭️  Skipping volume with empty source")
			continue
		}

		if !volume.RW && (containerConfig.IncludeRO == nil || !*containerConfig.IncludeRO) {
			logSubStep("⏭️  Skipping read-only volume")
			continue
		}

		if err := backupVolume(volume, tempDir); err != nil {
			return err
		}
	}

	if err := createFinalArchive(containerConfig.Container, tempDir, outputDir); err != nil {
		return err
	}

	return os.RemoveAll(filepath.Join(outputDir, "temp"))
}

func pullLatestPackmateImage() error {
	logSubStep("⬇️  Pulling the latest version of dublok/packmate...")
	args := []string{"pull", "dublok/packmate:latest"}
	_, err := executeCommand("docker", args...)
	if err != nil {
		return fmt.Errorf("failed to pull the latest Packmate image: %w", err)
	}
	logSubStep("✅ Successfully pulled the latest version of dublok/packmate")
	return nil
}

func backupVolume(volume Volume, tempDir string) error {
	logSubStep("💾 Creating backup with Packmate...")
	args := []string{
		"run", "--rm",
		"-v", volume.Source + ":/source:ro",
		"-v", tempDir + ":/output",
		"dublok/packmate:latest",
		"--name", base64.StdEncoding.EncodeToString([]byte(volume.Source)),
		"--compression=0",
		"--method=copy",
		"--multithreading=true",
		"--extra=-ms=off",
	}
	logSubStep("⚙️  Executing: docker %v", args)
	_, err := executeCommand("docker", args...)
	return err
}

func createFinalArchive(container, tempDir, outputDir string) error {
	finalArchivePath := filepath.Join(outputDir, container+".7z")
	args := []string{
		"run", "--rm",
		"-v", tempDir + ":/source:ro",
		"-v", filepath.Dir(finalArchivePath) + ":/output",
		"dublok/packmate:latest",
		"--name", container,
		"--compression=0",
		"--method=copy",
		"--multithreading=true",
		"--extra=-ms=off",
	}
	_, err := executeCommand("docker", args...)
	if err != nil {
		return err
	}

	// Verify the file exists
	if _, err := os.Stat(finalArchivePath); os.IsNotExist(err) {
		return fmt.Errorf("final archive was not created at %s", finalArchivePath)
	}
	return nil
}
