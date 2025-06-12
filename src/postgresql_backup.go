package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func getPostgreSQLDatabases(config PostgreSQLConfig) ([]string, error) {
	args := []string{
		"-h", config.Container,
		"-p", fmt.Sprintf("%d", config.Port),
		"-U", config.User,
		"-t",
		"-c", "SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres');",
	}

	cmd := exec.Command("psql", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get PostgreSQL database list: %v, output: %s", err, string(output))
	}

	databases := []string{}
	for _, db := range strings.Split(string(output), "\n") {
		db = strings.TrimSpace(db)
		if db != "" && db != "datname" && !strings.Contains(db, "----") && !strings.Contains(db, "row") {
			databases = append(databases, db)
		}
	}

	logStep("📋 Found PostgreSQL databases: %s", strings.Join(databases, ", "))
	return databases, nil
}

func dumpPostgreSQLDatabase(config PostgreSQLConfig, database string) (*PostgreSQLBackupResult, error) {
	logStep("📦 Backing up PostgreSQL database: %s", database)
	timestamp := time.Now().Format("20060102.150405")
	outputFile := fmt.Sprintf("%s_%s.sql", database, timestamp)
	fullPath := filepath.Join("/tmp", outputFile)

	args := []string{
		"-h", config.Container,
		"-p", fmt.Sprintf("%d", config.Port),
		"-U", config.User,
		"-f", fullPath,
		"--verbose",
		"--clean",
		"--no-owner",
		"--no-privileges",
		database,
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pg_dump failed for database %s: %v, output: %s",
			database, err, strings.TrimSpace(string(output)))
	}

	logStep("✅ Successfully backed up PostgreSQL database: %s", database)
	return &PostgreSQLBackupResult{
		Database:   database,
		BackupPath: fullPath,
		FileName:   outputFile,
		Status:     "completed",
	}, nil
}

func processPostgreSQLBackups(configs PostgreSQLConfigs, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	for _, config := range configs {
		logStep("🗄️ Processing PostgreSQL backup for container: %s", config.Container)
		var databasesToBackup []string
		var err error
		if len(config.Databases) == 0 {
			logStep("📑 No specific databases configured - backing up all PostgreSQL databases")
			databasesToBackup, err = getPostgreSQLDatabases(config)
			if err != nil {
				return err
			}
		} else {
			logStep("📑 Backing up specific PostgreSQL databases: %s", strings.Join(config.Databases, ", "))
			databasesToBackup = config.Databases
		}

		for _, db := range databasesToBackup {
			result, err := dumpPostgreSQLDatabase(config, db)
			if err != nil {
				return err
			}

			if err := uploadToDropboxPostgreSQL(config, result.BackupPath, result.Database, uploader, dropboxPath); err != nil {
				return err
			}

			if err := manageBackupRetentionPostgreSQL(config, uploader, dropboxPath, retention, db); err != nil {
				return err
			}
		}
	}
	return nil
}

func uploadToDropboxPostgreSQL(config PostgreSQLConfig, backupPath string, database string, uploader *DropboxUploader, dropboxPath string) error {
	defer func() {
		if err := os.Remove(backupPath); err != nil {
			logStep("⚠️ Warning: Failed to clean up local PostgreSQL backup file %s: %v", backupPath, err)
		} else {
			logStep("🗑️ Local PostgreSQL backup file deleted: %s", backupPath)
		}
	}()

	if uploader == nil {
		return nil
	}

	logHeader("📤 Uploading PostgreSQL backup to Dropbox...")
	timestamp := time.Now().Format("20060102.150405")
	backupFileName := timestamp + ".sql"
	backupID := getPostgreSQLBackupID(config)
	dropboxTargetPath := filepath.Join(dropboxPath, backupID, database, backupFileName)
	if !strings.HasPrefix(dropboxTargetPath, "/") {
		dropboxTargetPath = "/" + dropboxTargetPath
	}

	logStep("📁 Uploading PostgreSQL backup to Dropbox: %s", dropboxTargetPath)
	if err := uploader.Upload(backupPath, dropboxTargetPath); err != nil {
		return fmt.Errorf("dropbox upload failed: %v", err)
	}

	logStep("✅ PostgreSQL backup successfully uploaded to Dropbox")
	return nil
}

func manageBackupRetentionPostgreSQL(config PostgreSQLConfig, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy, database string) error {
	if uploader == nil {
		return nil
	}

	if retention.KeepDaily > 0 || retention.KeepWeekly > 0 ||
		retention.KeepMonthly > 0 || retention.KeepYearly > 0 {

		backupID := getPostgreSQLBackupID(config)
		retentionPath := filepath.Join(dropboxPath, backupID, database)

		if err := manageRetention(uploader, retentionPath, retention); err != nil {
			return fmt.Errorf("retention management failed for PostgreSQL database %s: %v", database, err)
		}
	}

	return nil
}

func getPostgreSQLBackupID(config PostgreSQLConfig) string {
	if config.BackupID != nil && *config.BackupID != "" {
		return *config.BackupID
	}
	return config.Container
}