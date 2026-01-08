package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
		if processed[config.Container] {
			return nil
		}

		// Process dependencies
		if deps, ok := dependencies[config.Container]; ok {
			for _, dep := range deps {
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
		if shouldStop(config) {
			if err := stopDockerContainer(config.Container); err != nil {
				return err
			}
		}

		// Create temporary working directory
		tempDir := filepath.Join("/tmp", "volback-"+config.Container+"-"+time.Now().Format("20060102150405"))
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}

		// Move cleanup to after all operations are complete
		defer func() {
			logStep("🧹 Cleaning up temporary directory: %s", tempDir)
			if err := os.RemoveAll(tempDir); err != nil {
				logSubStep("⚠️  Failed to remove temporary directory %s: %v", tempDir, err)
			}
		}()

		// Get and process volumes
		volumeResult, err := getContainerVolumes(config.Container)
		if err != nil {
			return err
		}
		if volumeResult.Status == "Failed" {
			return fmt.Errorf("failed to get container volumes: %s", volumeResult.Error)
		}

		if err := processVolumes(config, volumeResult.Volumes, tempDir); err != nil {
			return err
		}

		// Start container if it was stopped
		if shouldStop(config) {
			if err := startDockerContainer(config.Container); err != nil {
				return err
			}
		}

		// Upload to Dropbox
		if uploader != nil {
			logHeader("📤 Uploading backup to Dropbox...")
			timestamp := time.Now().Format("20060102.150405")
			backupFileName := timestamp + ".7z"
			localBackupPath := filepath.Join(tempDir, config.Container+".7z")

			// Verify backup file exists and has content before uploading
			fileInfo, err := os.Stat(localBackupPath)
			if err != nil {
				return fmt.Errorf("backup file not found: %v", err)
			}
			if fileInfo.Size() == 0 {
				return fmt.Errorf("backup file is empty (0 bytes), skipping upload for %s", config.Container)
			}

			// Use the helper function to get the backup ID
			backupID := getBackupID(config)
			dropboxTargetPath := filepath.Join(dropboxPath, backupID, backupFileName)

			if !strings.HasPrefix(dropboxTargetPath, "/") {
				dropboxTargetPath = "/" + dropboxTargetPath
			}

			logStep("📁 Uploading to Dropbox: %s", dropboxTargetPath)
			if err := uploader.Upload(localBackupPath, dropboxTargetPath); err != nil {
				return fmt.Errorf("dropbox upload failed: %v", err)
			}
			logStep("✅ Backup successfully uploaded to Dropbox")

			// Apply retention policy
			if retentionPolicy.KeepDaily > 0 || retentionPolicy.KeepWeekly > 0 ||
				retentionPolicy.KeepMonthly > 0 || retentionPolicy.KeepYearly > 0 {
				// Use the helper function here as well
				retentionPath := filepath.Join(dropboxPath, backupID)
				if err := manageRetention(uploader, retentionPath, retentionPolicy); err != nil {
					return fmt.Errorf("retention management failed: %v", err)
				}
			}
		}

		processed[config.Container] = true
		return nil
	}

	// Process all containers
	var errorList []string
	for _, config := range configs {
		if err := processContainer(config); err != nil {
			logStep("❌ Error processing container %s: %v", config.Container, err)
			errorList = append(errorList, fmt.Sprintf("%s: %v", config.Container, err))
			// Continue to next container instead of returning
		}
	}

	if len(errorList) > 0 {
		return fmt.Errorf("errors occurred during backup: %s", strings.Join(errorList, "; "))
	}
	return nil
}

func getBackupID(config ContainerConfig) string {
	if config.BackupID != nil && *config.BackupID != "" {
		return *config.BackupID
	}
	return config.Container
}

func shouldStop(config ContainerConfig) bool {
	return config.Stop != nil && *config.Stop
}
