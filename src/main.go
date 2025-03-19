package main

import (
	"encoding/json"
	"flag"
	"os"
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
