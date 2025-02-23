package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	logHeader("=== Docker Volume Backup Utility ===")

	// Flags
	container := flag.String("container", "", "Container name for volume operation")
	stopContainer := flag.String("stop", "", "Container name to stop and start")
	id := flag.String("id", "", "Unique identifier for the backup (optional)")
	// Dropbox flags
	dropboxRefreshToken := flag.String("dropbox-refresh-token", "", "Dropbox refresh token")
	dropboxClientID := flag.String("dropbox-client-id", "", "Dropbox client ID")
	dropboxClientSecret := flag.String("dropbox-client-secret", "", "Dropbox client secret")
	dropboxPath := flag.String("dropbox-path", "", "Dropbox destination path (e.g., /backups)")
	// Add retention flags
	keepDaily := flag.Int("keep-daily", 0, "Number of daily backups to keep")
	keepWeekly := flag.Int("keep-weekly", 0, "Number of weekly backups to keep")
	keepMonthly := flag.Int("keep-monthly", 0, "Number of monthly backups to keep")
	keepYearly := flag.Int("keep-yearly", 0, "Number of yearly backups to keep")
	flag.Parse()

	if err := validateInputs(*container); err != nil {
		os.Exit(1)
	}

	// Use --id if provided, otherwise default to container name
	backupID := *id
	if backupID == "" {
		backupID = *container
	}

	// Create temporary working directory
	tempDir, err := os.MkdirTemp("", "docker-backup-*")
	if err != nil {
		logStep("❌ Failed to create temporary directory: %v", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory when done

	// Stop container if requested
	if *stopContainer != "" {
		if err := stopDockerContainer(*stopContainer); err != nil {
			os.Exit(1)
		}
	}

	// Get container volumes
	volumeResult, err := getContainerVolumes(*container)
	if err != nil {
		logStep("❌ Failed to get container volumes: %v", err)
		os.Exit(1)
	}
	if volumeResult.Status == "Failed" {
		logStep("❌ Failed to get container volumes: %s", volumeResult.Error)
		os.Exit(1)
	}

	// Process volumes
	if err := processVolumes(*container, volumeResult.Volumes, tempDir); err != nil {
		os.Exit(1)
	}

	// Start container if it was stopped
	if *stopContainer != "" {
		if err := startDockerContainer(*stopContainer); err != nil {
			os.Exit(1)
		}
	}

	uploader := NewDropboxUploader(
		*dropboxRefreshToken,
		*dropboxClientID,
		*dropboxClientSecret,
	)

	// Upload to cloud storage
	if *dropboxRefreshToken != "" && *dropboxClientID != "" && *dropboxClientSecret != "" {
		logHeader("📤 Uploading backup to Dropbox...")

		// Generate timestamp-based filename
		timestamp := time.Now().Format("20060102.150405")
		backupFileName := timestamp + ".7z"

		// Construct source and target paths with .7z extension
		localBackupPath := filepath.Join(tempDir, *container+".7z")
		dropboxTargetPath := filepath.Join(*dropboxPath, backupID, backupFileName)

		// Ensure dropbox path starts with "/"
		if !strings.HasPrefix(dropboxTargetPath, "/") {
			dropboxTargetPath = "/" + dropboxTargetPath
		}

		logStep("📁 Uploading to Dropbox: %s", dropboxTargetPath)
		if err := uploader.Upload(localBackupPath, dropboxTargetPath); err != nil {
			logStep("❌ Dropbox upload failed: %v", err)
			os.Exit(1)
		}
		logStep("✅ Backup successfully uploaded to Dropbox")

		// Add retention management after successful upload
		if *keepDaily > 0 || *keepWeekly > 0 || *keepMonthly > 0 || *keepYearly > 0 {
			policy := RetentionPolicy{
				KeepDaily:   *keepDaily,
				KeepWeekly:  *keepWeekly,
				KeepMonthly: *keepMonthly,
				KeepYearly:  *keepYearly,
			}

			retentionPath := filepath.Join(*dropboxPath, backupID)
			if err := manageRetention(uploader, retentionPath, policy); err != nil {
				logStep("❌ Failed to manage retention: %v", err)
				os.Exit(1)
			}
		}
	} else {
		logStep("❌ No storage provider configured. Please specify a storage provider (e.g., Dropbox)")
		os.Exit(1)
	}

	logHeader("✨ Backup process completed successfully!")
}

func validateInputs(container string) error {
	logHeader("📋 Validating inputs...")
	if container == "" {
		logStep("❌ Container name is required (use --container flag)")
		return fmt.Errorf("container name required")
	}
	logStep("✅ Container: %s", container)
	return nil
}
