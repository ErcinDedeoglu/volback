package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	logHeader("=== Docker Volume Backup Utility ===")

	// Define flags
	containersJSON := flag.String("containers", os.Getenv("CONTAINERS"), "JSON array of container configurations")
	dropboxRefreshToken := flag.String("dropbox-refresh-token", os.Getenv("DROPBOX_REFRESH_TOKEN"), "Dropbox refresh token")
	dropboxClientID := flag.String("dropbox-client-id", os.Getenv("DROPBOX_CLIENT_ID"), "Dropbox client ID")
	dropboxClientSecret := flag.String("dropbox-client-secret", os.Getenv("DROPBOX_CLIENT_SECRET"), "Dropbox client secret")
	dropboxPath := flag.String("dropbox-path", os.Getenv("DROPBOX_PATH"), "Dropbox destination path (e.g., /backups)")

	// Retention flags
	keepDaily := flag.Int("keep-daily", getEnvInt("KEEP_DAILY", 0), "Number of daily backups to keep")
	keepWeekly := flag.Int("keep-weekly", getEnvInt("KEEP_WEEKLY", 0), "Number of weekly backups to keep")
	keepMonthly := flag.Int("keep-monthly", getEnvInt("KEEP_MONTHLY", 0), "Number of monthly backups to keep")
	keepYearly := flag.Int("keep-yearly", getEnvInt("KEEP_YEARLY", 0), "Number of yearly backups to keep")

	flag.Parse()

	// Parse container configurations
	var configs ContainerConfigs
	if err := json.Unmarshal([]byte(*containersJSON), &configs); err != nil {
		logStep("❌ Failed to parse container configurations: %v", err)
		os.Exit(1)
	}

	// Validate inputs
	if len(configs) == 0 {
		logStep("❌ No container configurations provided")
		os.Exit(1)
	}

	if *dropboxRefreshToken == "" || *dropboxClientID == "" || *dropboxClientSecret == "" {
		logStep("❌ Dropbox configuration is required")
		os.Exit(1)
	}

	// Initialize Dropbox uploader
	uploader := NewDropboxUploader(
		*dropboxRefreshToken,
		*dropboxClientID,
		*dropboxClientSecret,
	)

	// Create retention policy
	retentionPolicy := RetentionPolicy{
		KeepDaily:   *keepDaily,
		KeepWeekly:  *keepWeekly,
		KeepMonthly: *keepMonthly,
		KeepYearly:  *keepYearly,
	}

	logStep("📋 Found %d containers to process", len(configs))

	// Process all containers
	if err := processContainers(configs, uploader, *dropboxPath, retentionPolicy); err != nil {
		logStep("❌ Failed to process containers: %v", err)
		os.Exit(1)
	}

	logHeader("✨ Backup process completed successfully!")
}

func getEnvInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultVal
}

func processContainers(configs ContainerConfigs, uploader *DropboxUploader, dropboxPath string, retentionPolicy RetentionPolicy) error {
	// Create dependency graph
	dependencies := make(map[string][]string)
	for _, config := range configs {
		if len(config.DependsOn) > 0 {
			dependencies[config.Container] = config.DependsOn
		}
	}

	// Process containers in correct order
	processed := make(map[string]bool)
	var processContainer func(config ContainerConfig) error

	processContainer = func(config ContainerConfig) error {
		// Check if already processed
		if processed[config.Container] {
			return nil
		}

		// Process dependencies first
		if deps, ok := dependencies[config.Container]; ok {
			for _, dep := range deps {
				// Find dependency config
				for _, depConfig := range configs {
					if depConfig.Container == dep {
						if err := processContainer(depConfig); err != nil {
							return err
						}
						break
					}
				}
			}
		}

		logHeader("📦 Processing container: %s", config.Container)

		// Stop container if required
		if config.Stop {
			if err := stopDockerContainer(config.Container); err != nil {
				return err
			}
		}

		// Create temporary working directory
		tempDir := filepath.Join("/tmp", "volback-"+config.Container+"-"+time.Now().Format("20060102150405"))
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Get and process volumes
		volumeResult, err := getContainerVolumes(config.Container)
		if err != nil {
			return err
		}
		if volumeResult.Status == "Failed" {
			return fmt.Errorf("failed to get container volumes: %s", volumeResult.Error)
		}

		if err := processVolumes(config.Container, volumeResult.Volumes, tempDir); err != nil {
			return err
		}

		// Start container if it was stopped
		if config.Stop {
			if err := startDockerContainer(config.Container); err != nil {
				return err
			}
		}

		// Upload to Dropbox using existing logic
		if uploader != nil {
			logHeader("📤 Uploading backup to Dropbox...")

			// Generate timestamp-based filename
			timestamp := time.Now().Format("20060102.150405")
			backupFileName := timestamp + ".7z"

			// Construct source and target paths
			localBackupPath := filepath.Join(tempDir, config.Container+".7z")
			dropboxTargetPath := filepath.Join(dropboxPath, config.BackupID, backupFileName)

			// Ensure dropbox path starts with "/"
			if !strings.HasPrefix(dropboxTargetPath, "/") {
				dropboxTargetPath = "/" + dropboxTargetPath
			}

			logStep("📁 Uploading to Dropbox: %s", dropboxTargetPath)
			if err := uploader.Upload(localBackupPath, dropboxTargetPath); err != nil {
				return fmt.Errorf("Dropbox upload failed: %v", err)
			}
			logStep("✅ Backup successfully uploaded to Dropbox")

			// Apply retention policy
			if retentionPolicy.KeepDaily > 0 || retentionPolicy.KeepWeekly > 0 ||
				retentionPolicy.KeepMonthly > 0 || retentionPolicy.KeepYearly > 0 {
				retentionPath := filepath.Join(dropboxPath, config.BackupID)
				if err := manageRetention(uploader, retentionPath, retentionPolicy); err != nil {
					return fmt.Errorf("retention management failed: %v", err)
				}
			}
		}

		processed[config.Container] = true
		return nil
	}

	// Process all containers
	for _, config := range configs {
		if err := processContainer(config); err != nil {
			return err
		}
	}

	return nil
}
