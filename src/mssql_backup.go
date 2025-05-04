package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func getMSSQLDatabases(config MSSQLConfig) ([]string, error) {
	args := []string{
		"-S", fmt.Sprintf("%s,%d", config.Host, config.Port),
		"-U", config.User,
		"-P", config.Password,
		"-C",
		"-Q", "SELECT name FROM sys.databases WHERE name NOT IN ('master', 'tempdb', 'model', 'msdb')",
		"-h", "1",
		"-W",
		"-s", ",",
	}

	cmd := exec.Command("sqlcmd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get MSSQL database list: %v, output: %s", err, string(output))
	}

	databases := []string{}
	for _, db := range strings.Split(string(output), "\n") {
		db = strings.TrimSpace(db)
		if db == "" || db == "name" || strings.Contains(db, "rows affected") || strings.Contains(db, "----") {
			continue
		}
		databases = append(databases, db)
	}

	logStep("📋 Found MSSQL databases: %s", strings.Join(databases, ", "))
	return databases, nil
}

func dumpMSSQLDatabase(config MSSQLConfig, database string) (*MSSQLBackupResult, error) {
	logStep("📦 Backing up MSSQL database: %s", database)
	timestamp := time.Now().Format("20060102.150405")
	outputFile := fmt.Sprintf("%s_%s.bak", database, timestamp)
	dbBackupPath := filepath.Join("/var/opt/mssql/backup", outputFile)
	localPath := filepath.Join("/tmp", outputFile)

	backupSQL := fmt.Sprintf("BACKUP DATABASE [%s] TO DISK = N'%s' WITH INIT, FORMAT;", database, dbBackupPath)
	args := []string{
		"-S", fmt.Sprintf("%s,%d", config.Host, config.Port),
		"-U", config.User,
		"-P", config.Password,
		"-C",
		"-Q", backupSQL,
	}

	cmd := exec.Command("sqlcmd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlcmd backup failed for database %s: %v, output: %s", database, err, strings.TrimSpace(string(output)))
	}

	logStep("✅ Successfully backed up MSSQL database: %s", database)
	return &MSSQLBackupResult{
		Database:   database,
		BackupPath: localPath,
		FileName:   outputFile,
		Status:     "completed",
	}, nil
}

func processMSSQLBackups(configs MSSQLConfigs, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	for _, config := range configs {
		logStep("🗄️ Processing MSSQL backup for host: %s", config.Host)
		var databasesToBackup []string
		var err error
		if len(config.Databases) == 0 {
			logStep("📑 No specific databases configured - backing up all MSSQL databases")
			databasesToBackup, err = getMSSQLDatabases(config)
			if err != nil {
				return err
			}
		} else {
			logStep("📑 Backing up specific MSSQL databases: %s", strings.Join(config.Databases, ", "))
			databasesToBackup = config.Databases
		}

		for _, db := range databasesToBackup {
			result, err := dumpMSSQLDatabase(config, db)
			if err != nil {
				return err
			}

			if err := uploadToDropboxMSSQL(config, result.BackupPath, result.Database, uploader, dropboxPath); err != nil {
				return err
			}

			if err := manageBackupRetentionMSSQL(config, uploader, dropboxPath, retention, db); err != nil {
				return err
			}
		}
	}
	return nil
}

func uploadToDropboxMSSQL(config MSSQLConfig, backupPath string, database string, uploader *DropboxUploader, dropboxPath string) error {
	defer func() {
		if err := os.Remove(backupPath); err != nil {
			logStep("⚠️ Warning: Failed to clean up local MSSQL backup file %s: %v", backupPath, err)
		} else {
			logStep("🗑️ Local MSSQL backup file deleted: %s", backupPath)
		}
	}()

	if uploader == nil {
		return nil
	}

	logHeader("📤 Uploading MSSQL backup to Dropbox...")
	timestamp := time.Now().Format("20060102.150405")
	backupFileName := timestamp + ".bak"
	backupID := getMSSQLBackupID(config)
	dropboxTargetPath := filepath.Join(dropboxPath, backupID, database, backupFileName)
	if !strings.HasPrefix(dropboxTargetPath, "/") {
		dropboxTargetPath = "/" + dropboxTargetPath
	}

	logStep("📁 Uploading MSSQL backup to Dropbox: %s", dropboxTargetPath)
	if err := uploader.Upload(backupPath, dropboxTargetPath); err != nil {
		return fmt.Errorf("dropbox upload failed: %v", err)
	}

	logStep("✅ MSSQL backup successfully uploaded to Dropbox")
	return nil
}

func manageBackupRetentionMSSQL(config MSSQLConfig, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy, database string) error {
	if uploader == nil {
		return nil
	}

	if retention.KeepDaily > 0 || retention.KeepWeekly > 0 ||
		retention.KeepMonthly > 0 || retention.KeepYearly > 0 {

		backupID := getMSSQLBackupID(config)
		retentionPath := filepath.Join(dropboxPath, backupID, database)

		if err := manageRetention(uploader, retentionPath, retention); err != nil {
			return fmt.Errorf("retention management failed for MSSQL database %s: %v", database, err)
		}
	}

	return nil
}

func getMSSQLBackupID(config MSSQLConfig) string {
	if config.BackupID != nil && *config.BackupID != "" {
		return *config.BackupID
	}
	return config.Host
}
