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
	mysqlJSON := flag.String("mysql", os.Getenv("MYSQL"), "JSON array of MySQL configurations")
	mssqlJSON := flag.String("mssql", os.Getenv("MSSQL"), "JSON array of MSSQL configurations")
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

	// Validate Dropbox configuration
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

	// Process container backups if configured
	if *containersJSON != "" {
		var configs ContainerConfigs
		if err := json.Unmarshal([]byte(*containersJSON), &configs); err != nil {
			logStep("❌ Failed to parse container configurations: %v", err)
			os.Exit(1)
		}
		logStep("📋 Found %d containers to process", len(configs))
		if err := processContainers(configs, uploader, *dropboxPath, retentionPolicy); err != nil {
			logStep("❌ Failed to process containers: %v", err)
			os.Exit(1)
		}
		logStep("✅ Container backups completed successfully")
	}

	// Process MySQL backups if configured
	if *mysqlJSON != "" {
		var mysqlConfigs MySQLConfigs
		if err := json.Unmarshal([]byte(*mysqlJSON), &mysqlConfigs); err != nil {
			logStep("❌ Failed to parse MySQL configurations: %v", err)
			os.Exit(1)
		}

		for i := range mysqlConfigs {
			if mysqlConfigs[i].Port == 0 {
				mysqlConfigs[i].Port = 3306
			}
		}

		logStep("📋 Found %d MySQL configurations to backup", len(mysqlConfigs))
		if err := processMySQLBackups(mysqlConfigs, uploader, *dropboxPath, retentionPolicy); err != nil {
			logStep("❌ Failed to process MySQL backups: %v", err)
			os.Exit(1)
		}
		logStep("✅ MySQL backups completed successfully")
	}

	// Process MSSQL backups if configured
	if *mssqlJSON != "" {
		var mssqlConfigs MSSQLConfigs
		if err := json.Unmarshal([]byte(*mssqlJSON), &mssqlConfigs); err != nil {
			logStep("❌ Failed to parse MSSQL configurations: %v", err)
			os.Exit(1)
		}

		for i := range mssqlConfigs {
			if mssqlConfigs[i].Port == 0 {
				mssqlConfigs[i].Port = 1433
			}
		}

		logStep("📋 Found %d MSSQL configurations to backup", len(mssqlConfigs))
		if err := processMSSQLBackups(mssqlConfigs, uploader, *dropboxPath, retentionPolicy); err != nil {
			logStep("❌ Failed to process MSSQL backups: %v", err)
			os.Exit(1)
		}
		logStep("✅ MSSQL backups completed successfully")
	}

	// Check if at least one backup type was processed
	if *containersJSON == "" && *mysqlJSON == "" && *mssqlJSON == "" {
		logStep("❌ No backup configurations provided (containers, MySQL, or MSSQL)")
		os.Exit(1)
	}

	logHeader("✨ Backup process completed successfully!")
}
