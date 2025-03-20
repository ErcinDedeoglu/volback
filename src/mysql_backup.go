package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func getDatabases(config MySQLConfig) ([]string, error) {
	args := []string{
		"-h", config.Container,
		"-P", fmt.Sprintf("%d", config.Port),
		"-u", config.User,
		"-N", "-e", "SHOW DATABASES",
	}
	cmd := exec.Command("mysql", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", config.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get database list: %v, output: %s", err, string(output))
	}

	databases := []string{}
	for _, db := range strings.Split(string(output), "\n") {
		db = strings.TrimSpace(db)
		if db != "" &&
			db != "information_schema" &&
			db != "performance_schema" &&
			db != "mysql" &&
			db != "sys" {
			databases = append(databases, db)
		}
	}
	logStep("📋 Found databases: %s", strings.Join(databases, ", "))
	return databases, nil
}

func dumpDatabase(config MySQLConfig, database string) (*MySQLBackupResult, error) {
	logStep("📦 Backing up database: %s", database)

	// Create output file path in /tmp directory
	outputFile := fmt.Sprintf("%s.sql", database)
	fullPath := filepath.Join("/tmp", outputFile)

	args := []string{
		"-h", config.Container,
		"-P", fmt.Sprintf("%d", config.Port),
		"-u", config.User,
		"--result-file=" + fullPath,
		database,
	}

	cmd := exec.Command("mysqldump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", config.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mysqldump failed for database %s: %v, output: %s",
			database, err, strings.TrimSpace(string(output)))
	}

	logStep("✅ Successfully backed up database: %s", database)
	return &MySQLBackupResult{
		Database:   database,
		BackupPath: fullPath,
		FileName:   outputFile,
		Status:     "completed",
	}, nil
}

func processMySQLBackups(configs MySQLConfigs, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	for _, config := range configs {
		logStep("🗄️ Processing MySQL backup for host: %s", config.Container)
		var databasesToBackup []string
		var err error
		if len(config.Databases) == 0 {
			logStep("📑 No specific databases configured - backing up all databases")
			databasesToBackup, err = getDatabases(config)
			if err != nil {
				return err
			}
		} else {
			logStep("📑 Backing up specific databases: %s", strings.Join(config.Databases, ", "))
			databasesToBackup = config.Databases
		}

		for _, db := range databasesToBackup {
			result, err := dumpDatabase(config, db)
			if err != nil {
				return err
			}

			if err := uploadToDropbox(config, result.BackupPath, result.Database, uploader, dropboxPath); err != nil {
				return err
			}

			if err := manageBackupRetention(config, uploader, dropboxPath, retention); err != nil {
				return err
			}
		}
	}
	return nil
}

func uploadToDropbox(config MySQLConfig, backupPath string, database string, uploader *DropboxUploader, dropboxPath string) error {
	defer func() {
		if err := os.Remove(backupPath); err != nil {
			logStep("⚠️ Warning: Failed to clean up local backup file %s: %v", backupPath, err)
		} else {
			logStep("🗑️ Local backup file deleted: %s", backupPath)
		}
	}()

	if uploader == nil {
		return nil
	}

	logHeader("📤 Uploading backup to Dropbox...")
	timestamp := time.Now().Format("20060102.150405")
	backupFileName := timestamp + ".sql"
	backupID := getMySQLBackupID(config)
	dropboxTargetPath := filepath.Join(dropboxPath, backupID, database, backupFileName)
	if !strings.HasPrefix(dropboxTargetPath, "/") {
		dropboxTargetPath = "/" + dropboxTargetPath
	}

	logStep("📁 Uploading to Dropbox: %s", dropboxTargetPath)
	if err := uploader.Upload(backupPath, dropboxTargetPath); err != nil {
		return fmt.Errorf("dropbox upload failed: %v", err)
	}

	logStep("✅ Backup successfully uploaded to Dropbox")
	return nil
}

func manageBackupRetention(config MySQLConfig, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	if uploader == nil {
		return nil
	}

	if retention.KeepDaily > 0 || retention.KeepWeekly > 0 ||
		retention.KeepMonthly > 0 || retention.KeepYearly > 0 {

		backupID := getMySQLBackupID(config)
		retentionPath := filepath.Join(dropboxPath, backupID)

		if err := manageRetention(uploader, retentionPath, retention); err != nil {
			return fmt.Errorf("retention management failed: %v", err)
		}
	}

	return nil
}

func getMySQLBackupID(config MySQLConfig) string {
	if config.BackupID != nil && *config.BackupID != "" {
		return *config.BackupID
	}
	return config.Container
}
