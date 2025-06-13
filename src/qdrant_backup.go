package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type QdrantCollectionsResponse struct {
	Result struct {
		Collections []struct {
			Name string `json:"name"`
		} `json:"collections"`
	} `json:"result"`
}

type QdrantSnapshotResponse struct {
	Result struct {
		Name string `json:"name"`
	} `json:"result"`
}

func getQdrantCollections(config QdrantConfig) ([]string, error) {
	url := fmt.Sprintf("http://%s:%d/collections", config.Host, config.Port)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	if config.ApiKey != nil && *config.ApiKey != "" {
		req.Header.Set("api-key", *config.ApiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get collections from Qdrant: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Qdrant API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response QdrantCollectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode collections response: %v", err)
	}

	collections := make([]string, len(response.Result.Collections))
	for i, collection := range response.Result.Collections {
		collections[i] = collection.Name
	}

	logStep("📋 Found Qdrant collections: %s", strings.Join(collections, ", "))
	return collections, nil
}

func dumpQdrantCollection(config QdrantConfig, collection string) (*QdrantBackupResult, error) {
	logStep("📦 Backing up Qdrant collection: %s", collection)
	
	// Create snapshot
	snapshotURL := fmt.Sprintf("http://%s:%d/collections/%s/snapshots", config.Host, config.Port, collection)
	req, err := http.NewRequest("POST", snapshotURL, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.ApiKey != nil && *config.ApiKey != "" {
		req.Header.Set("api-key", *config.ApiKey)
	}

	client := &http.Client{Timeout: 300 * time.Second} // 5 minutes for snapshot creation
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot for collection %s: %v", collection, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("snapshot creation failed for collection %s, status %d: %s", collection, resp.StatusCode, string(body))
	}

	var snapshotResponse QdrantSnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapshotResponse); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot response: %v", err)
	}

	snapshotName := snapshotResponse.Result.Name
	logStep("📸 Snapshot created: %s", snapshotName)

	// Download snapshot file
	downloadURL := fmt.Sprintf("http://%s:%d/collections/%s/snapshots/%s", config.Host, config.Port, collection, snapshotName)
	downloadReq, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %v", err)
	}

	if config.ApiKey != nil && *config.ApiKey != "" {
		downloadReq.Header.Set("api-key", *config.ApiKey)
	}

	downloadResp, err := client.Do(downloadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to download snapshot for collection %s: %v", collection, err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(downloadResp.Body)
		return nil, fmt.Errorf("snapshot download failed for collection %s, status %d: %s", collection, downloadResp.StatusCode, string(body))
	}

	// Save to local file
	timestamp := time.Now().Format("20060102.150405")
	outputFile := fmt.Sprintf("%s_%s.snapshot", collection, timestamp)
	fullPath := filepath.Join("/tmp", outputFile)

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create local snapshot file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, downloadResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save snapshot file: %v", err)
	}

	logStep("✅ Successfully backed up Qdrant collection: %s", collection)
	return &QdrantBackupResult{
		Collection: collection,
		BackupPath: fullPath,
		FileName:   outputFile,
		Status:     "completed",
	}, nil
}

func processQdrantBackups(configs QdrantConfigs, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy) error {
	for _, config := range configs {
		logStep("🗄️ Processing Qdrant backup for host: %s", config.Host)
		var collectionsToBackup []string
		var err error
		
		if len(config.Collections) == 0 {
			logStep("📑 No specific collections configured - backing up all Qdrant collections")
			collectionsToBackup, err = getQdrantCollections(config)
			if err != nil {
				return err
			}
		} else {
			logStep("📑 Backing up specific Qdrant collections: %s", strings.Join(config.Collections, ", "))
			collectionsToBackup = config.Collections
		}

		for _, collection := range collectionsToBackup {
			result, err := dumpQdrantCollection(config, collection)
			if err != nil {
				return err
			}

			if err := uploadToDropboxQdrant(config, result.BackupPath, result.Collection, uploader, dropboxPath); err != nil {
				return err
			}

			if err := manageBackupRetentionQdrant(config, uploader, dropboxPath, retention, collection); err != nil {
				return err
			}
		}
	}
	return nil
}

func uploadToDropboxQdrant(config QdrantConfig, backupPath string, collection string, uploader *DropboxUploader, dropboxPath string) error {
	defer func() {
		if err := os.Remove(backupPath); err != nil {
			logStep("⚠️ Warning: Failed to clean up local Qdrant backup file %s: %v", backupPath, err)
		} else {
			logStep("🗑️ Local Qdrant backup file deleted: %s", backupPath)
		}
	}()

	if uploader == nil {
		return nil
	}

	logHeader("📤 Uploading Qdrant backup to Dropbox...")
	timestamp := time.Now().Format("20060102.150405")
	backupFileName := timestamp + ".snapshot"
	backupID := getQdrantBackupID(config)
	dropboxTargetPath := filepath.Join(dropboxPath, backupID, collection, backupFileName)
	if !strings.HasPrefix(dropboxTargetPath, "/") {
		dropboxTargetPath = "/" + dropboxTargetPath
	}

	logStep("📁 Uploading Qdrant backup to Dropbox: %s", dropboxTargetPath)
	if err := uploader.Upload(backupPath, dropboxTargetPath); err != nil {
		return fmt.Errorf("dropbox upload failed: %v", err)
	}

	logStep("✅ Qdrant backup successfully uploaded to Dropbox")
	return nil
}

func manageBackupRetentionQdrant(config QdrantConfig, uploader *DropboxUploader, dropboxPath string, retention RetentionPolicy, collection string) error {
	if uploader == nil {
		return nil
	}

	if retention.KeepDaily > 0 || retention.KeepWeekly > 0 ||
		retention.KeepMonthly > 0 || retention.KeepYearly > 0 {

		backupID := getQdrantBackupID(config)
		retentionPath := filepath.Join(dropboxPath, backupID, collection)

		if err := manageRetention(uploader, retentionPath, retention); err != nil {
			return fmt.Errorf("retention management failed for Qdrant collection %s: %v", collection, err)
		}
	}

	return nil
}

func getQdrantBackupID(config QdrantConfig) string {
	if config.BackupID != nil && *config.BackupID != "" {
		return *config.BackupID
	}
	return config.Host
}