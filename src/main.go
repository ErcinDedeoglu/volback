package main

import (
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
	container := flag.String("container", os.Getenv("CONTAINER"), "Container name for volume operation")
	stopContainer := flag.String("stop", os.Getenv("STOP_CONTAINER"), "Container name to stop and start")
	id := flag.String("id", os.Getenv("BACKUP_ID"), "Unique identifier for the backup (optional)")

	// Dropbox flags
	dropboxRefreshToken := flag.String("dropbox-refresh-token", os.Getenv("DROPBOX_REFRESH_TOKEN"), "Dropbox refresh token")
	dropboxClientID := flag.String("dropbox-client-id", os.Getenv("DROPBOX_CLIENT_ID"), "Dropbox client ID")
	dropboxClientSecret := flag.String("dropbox-client-secret", os.Getenv("DROPBOX_CLIENT_SECRET"), "Dropbox client secret")
	dropboxPath := flag.String("dropbox-path", os.Getenv("DROPBOX_PATH"), "Dropbox destination path (e.g., /backups)")

	// Retention flags with environment variable fallbacks
	keepDaily := flag.Int("keep-daily", getEnvInt("KEEP_DAILY", 0), "Number of daily backups to keep")
	keepWeekly := flag.Int("keep-weekly", getEnvInt("KEEP_WEEKLY", 0), "Number of weekly backups to keep")
	keepMonthly := flag.Int("keep-monthly", getEnvInt("KEEP_MONTHLY", 0), "Number of monthly backups to keep")
	keepYearly := flag.Int("keep-yearly", getEnvInt("KEEP_YEARLY", 0), "Number of yearly backups to keep")

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
	tempDir := filepath.Join("/tmp", "volback-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
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

			logHeader("🧹 Starting retention management...")
			logStep("📊 Retention Policy:")
			logSubStep("Daily backups to keep: %d", policy.KeepDaily)
			logSubStep("Weekly backups to keep: %d", policy.KeepWeekly)
			logSubStep("Monthly backups to keep: %d", policy.KeepMonthly)
			logSubStep("Yearly backups to keep: %d", policy.KeepYearly)

			retentionPath := filepath.Join(*dropboxPath, backupID)
			logStep("📁 Processing retention for path: %s", retentionPath)

			if err := manageRetention(uploader, retentionPath, policy); err != nil {
				logStep("❌ Retention management failed: %v", err)
				// Decide if you want to exit here or continue
				// os.Exit(1) // Uncomment if you want to exit on retention failure
			} else {
				logStep("✅ Retention management completed successfully")
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

func getEnvInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultVal
}
