package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
)

func processVolumes(container string, volumes []Volume, outputDir string) error {
	tempDir := filepath.Join(outputDir, "temp", container)
	os.MkdirAll(tempDir, 0755)

	for i, volume := range volumes {
		logHeader("🔸 Volume %d/%d:", i+1, len(volumes))
		if err := backupVolume(volume, tempDir); err != nil {
			return err
		}
	}

	if err := createFinalArchive(container, tempDir, outputDir); err != nil {
		return err
	}

	return os.RemoveAll(filepath.Join(outputDir, "temp"))
}

func backupVolume(volume Volume, tempDir string) error {
	logSubStep("Source: %s", volume.Source)
	logSubStep("Destination: %s", volume.Destination)
	logSubStep("Type: %s", volume.Type)
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

	_, err := executeCommand("docker", args...)
	return err
}

func createFinalArchive(container, tempDir, outputDir string) error {
	args := []string{
		"run", "--rm",
		"-v", tempDir + ":/source:ro",
		"-v", outputDir + ":/output",
		"dublok/packmate:latest",
		"--name", container,
		"--compression=0",
		"--method=copy",
		"--multithreading=true",
		"--extra=-ms=off",
	}

	_, err := executeCommand("docker", args...)
	return err
}
