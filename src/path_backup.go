package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func processPathBackups(configs PathConfigs, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	if err := pullLatestPackmateImage(); err != nil {
		return fmt.Errorf("failed to pull the latest Packmate image: %w", err)
	}

	var errorList []string
	for _, config := range configs {
		if err := processPathBackup(config, uploader, dropboxPath, retention); err != nil {
			logStep("❌ Error processing path %s: %v", config.Path, err)
			errorList = append(errorList, fmt.Sprintf("%s: %v", config.Path, err))
		}
	}

	if len(errorList) > 0 {
		return fmt.Errorf("errors occurred during path backup: %s", strings.Join(errorList, "; "))
	}
	return nil
}

func processPathBackup(config PathConfig, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	logHeader("📁 Processing path: %s", config.Path)

	if _, err := os.Stat(config.Path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", config.Path)
	}

	backupID := getPathBackupID(config)
	tempDir := filepath.Join("/tmp", "volback-path-"+backupID+"-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}

	defer func() {
		logStep("🧹 Cleaning up temporary directory: %s", tempDir)
		if err := os.RemoveAll(tempDir); err != nil {
			logSubStep("⚠️  Failed to remove temporary directory %s: %v", tempDir, err)
		}
	}()

	archivePath := filepath.Join(tempDir, backupID+".7z")
	if err := createPathArchive(config.Path, backupID, tempDir); err != nil {
		return fmt.Errorf("failed to create archive: %v", err)
	}

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return fmt.Errorf("archive was not created at %s", archivePath)
	}

	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("failed to stat archive: %v", err)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("archive is empty (0 bytes)")
	}

	if uploader != nil {
		if err := uploadPathBackup(archivePath, backupID, uploader, dropboxPath); err != nil {
			return err
		}

		if err := managePathRetention(backupID, uploader, dropboxPath, retention); err != nil {
			return err
		}
	}

	return nil
}

func createPathArchive(sourcePath, backupID, outputDir string) error {
	logStep("💾 Creating archive with Packmate...")
	args := []string{
		"run", "--rm",
		"-v", sourcePath + ":/source:ro",
		"-v", outputDir + ":/output",
		"dublok/packmate:latest",
		"--name", backupID,
		"--compression=0",
		"--method=copy",
		"--multithreading=true",
		"--extra=-ms=off",
	}
	logSubStep("⚙️  Executing: docker %v", args)
	_, err := executeCommand("docker", args...)
	return err
}

func uploadPathBackup(archivePath, backupID string, uploader *DropboxUploader, dropboxPath string) error {
	logHeader("📤 Uploading backup to Dropbox...")
	timestamp := time.Now().Format("20060102.150405")
	backupFileName := timestamp + ".7z"
	dropboxTargetPath := filepath.Join(dropboxPath, backupID, backupFileName)
	if !strings.HasPrefix(dropboxTargetPath, "/") {
		dropboxTargetPath = "/" + dropboxTargetPath
	}

	logStep("📁 Uploading to Dropbox: %s", dropboxTargetPath)
	if err := uploader.Upload(archivePath, dropboxTargetPath); err != nil {
		return fmt.Errorf("dropbox upload failed: %v", err)
	}

	logStep("✅ Backup successfully uploaded to Dropbox")
	return nil
}

func managePathRetention(backupID string, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	if retention.KeepDaily > 0 || retention.KeepWeekly > 0 ||
		retention.KeepMonthly > 0 || retention.KeepYearly > 0 {
		retentionPath := filepath.Join(dropboxPath, backupID)
		if err := manageRetention(uploader, retentionPath, retention); err != nil {
			return fmt.Errorf("retention management failed: %v", err)
		}
	}
	return nil
}

func getPathBackupID(config PathConfig) string {
	if config.BackupID != nil && *config.BackupID != "" {
		return *config.BackupID
	}
	return filepath.Base(config.Path)
}
