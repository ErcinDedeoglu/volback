package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	dropboxUploadURL     = "https://content.dropboxapi.com/2/files/upload"
	dropboxTokenURL      = "https://api.dropbox.com/oauth2/token"
	dropboxListFolderURL = "https://api.dropboxapi.com/2/files/list_folder"
	dropboxDeleteFileURL = "https://api.dropboxapi.com/2/files/delete_v2"
)

type DropboxUploader struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
	accessToken  string
	tokenExpiry  time.Time
}

type DropboxAPIArg struct {
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	AutoRename     bool   `json:"autorename"`
	Mute           bool   `json:"mute"`
	StrictConflict bool   `json:"strict_conflict"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// Dropbox API response structures
type DropboxListFolderResponse struct {
	Entries []struct {
		Name string `json:".tag"`
		Path string `json:"path_display"`
	} `json:"entries"`
	HasMore bool   `json:"has_more"`
	Cursor  string `json:"cursor"`
}

type DropboxDeleteFileResponse struct {
	Metadata struct {
		Path string `json:"path_display"`
	} `json:"metadata"`
}

// ListFiles returns a list of files in the specified Dropbox path
func (d *DropboxUploader) ListFiles(path string) ([]string, error) {
	// Ensure path starts with "/"
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"path":      path,
		"recursive": false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", dropboxListFolderURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var listResponse DropboxListFolderResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	// Extract file paths
	var files []string
	for _, entry := range listResponse.Entries {
		if strings.HasSuffix(entry.Path, ".7z") {
			files = append(files, entry.Path)
		}
	}

	return files, nil
}

// DeleteFile deletes a file from Dropbox
func (d *DropboxUploader) DeleteFile(path string) error {
	// Ensure path starts with "/"
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"path": path,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", dropboxDeleteFileURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var deleteResponse DropboxDeleteFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&deleteResponse); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
	}

	logSubStep("Deleted file: %s", deleteResponse.Metadata.Path)
	return nil
}

func NewDropboxUploader(refreshToken, clientID, clientSecret string) *DropboxUploader {
	return &DropboxUploader{
		RefreshToken: refreshToken,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

func (d *DropboxUploader) refreshAccessToken() error {
	// Create form data
	formData := url.Values{}
	formData.Set("grant_type", "refresh_token")
	formData.Set("refresh_token", d.RefreshToken)
	formData.Set("client_id", d.ClientID)
	formData.Set("client_secret", d.ClientSecret)

	// Create request
	req, err := http.NewRequest("POST", dropboxTokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %v", err)
	}

	// Set correct content type for form data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute token request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %v", err)
	}

	d.accessToken = tokenResp.AccessToken
	d.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

func (d *DropboxUploader) ensureValidToken() error {
	if d.accessToken == "" || time.Now().After(d.tokenExpiry) {
		return d.refreshAccessToken()
	}
	return nil
}

// Upload uploads a local file to Dropbox
// sourcePath: local file path
// targetPath: destination path in Dropbox (should start with "/")
func (d *DropboxUploader) Upload(sourcePath, targetPath string) error {
	const CHUNK_SIZE = 150 * 1024 * 1024 // 150MB chunks

	// Open and stat the source file
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}
	fileSize := fileInfo.Size()

	if err := d.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %v", err)
	}

	logStep("📁 Starting upload for: %s", filepath.Base(sourcePath))
	logSubStep("Target path: %s", targetPath)
	logSubStep("File size: %.2f MB", float64(fileSize)/1024/1024)

	// For files larger than CHUNK_SIZE, use upload session
	if fileSize > CHUNK_SIZE {
		logStep("📦 Large file detected - using chunked upload")
		return d.uploadLargeFile(file, targetPath, fileSize, CHUNK_SIZE)
	}

	// For smaller files, use simple upload
	logStep("📦 Small file detected - using simple upload")
	return d.uploadSmallFile(file, targetPath)
}

func (d *DropboxUploader) uploadLargeFile(file *os.File, targetPath string, fileSize int64, chunkSize int64) error {
	logStep("📦 Starting chunked upload process...")
	logSubStep("Total file size: %.2f MB", float64(fileSize)/1024/1024)
	logSubStep("Chunk size: %.2f MB", float64(chunkSize)/1024/1024)
	totalChunks := (fileSize + chunkSize - 1) / chunkSize
	logSubStep("Total chunks: %d", totalChunks)

	// Start upload session
	logStep("📤 Uploading first chunk...")
	sessionID, err := d.startUploadSession(file, chunkSize)
	if err != nil {
		return fmt.Errorf("failed to start upload session: %v", err)
	}
	logSubStep("✅ First chunk uploaded successfully")
	logSubStep("Session ID: %s", sessionID)

	// Upload chunks
	offset := chunkSize
	currentChunk := 2 // Starting from 2 as we've already uploaded the first chunk
	for offset < fileSize {
		remaining := fileSize - offset
		currentChunkSize := chunkSize
		if remaining < chunkSize {
			currentChunkSize = remaining
		}

		logStep("📤 Uploading chunk %d/%d...", currentChunk, totalChunks)
		logSubStep("Offset: %.2f MB", float64(offset)/1024/1024)
		logSubStep("Chunk size: %.2f MB", float64(currentChunkSize)/1024/1024)

		if err := d.appendToUploadSession(file, sessionID, offset, currentChunkSize); err != nil {
			return fmt.Errorf("failed to append chunk: %v", err)
		}
		logSubStep("✅ Chunk uploaded successfully")

		offset += currentChunkSize
		currentChunk++
	}

	// Finish upload session
	logStep("📤 Finalizing upload...")
	if err := d.finishUploadSession(sessionID, targetPath, offset); err != nil {
		return err
	}
	logStep("✅ Upload completed successfully")
	return nil
}

func (d *DropboxUploader) startUploadSession(file *os.File, chunkSize int64) (string, error) {
	const uploadSessionStartURL = "https://content.dropboxapi.com/2/files/upload_session/start"

	logSubStep("Starting upload session...")
	buffer := make([]byte, chunkSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	logSubStep("Read %.2f MB from file", float64(n)/1024/1024)

	req, err := http.NewRequest("POST", uploadSessionStartURL, bytes.NewReader(buffer[:n]))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	logSubStep("Sending request to Dropbox...")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload session start failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.SessionID, nil
}

func (d *DropboxUploader) appendToUploadSession(file *os.File, sessionID string, offset, chunkSize int64) error {
	const uploadSessionAppendURL = "https://content.dropboxapi.com/2/files/upload_session/append_v2"

	buffer := make([]byte, chunkSize)
	n, err := file.ReadAt(buffer, offset)
	if err != nil && err != io.EOF {
		return err
	}

	// The correct API argument structure for append_v2
	cursor := struct {
		Cursor struct {
			SessionID string `json:"session_id"`
			Offset    int64  `json:"offset"`
		} `json:"cursor"`
		Close bool `json:"close"`
	}{
		Cursor: struct {
			SessionID string `json:"session_id"`
			Offset    int64  `json:"offset"`
		}{
			SessionID: sessionID,
			Offset:    offset,
		},
		Close: false,
	}

	cursorJSON, err := json.Marshal(cursor)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", uploadSessionAppendURL, bytes.NewReader(buffer[:n]))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Dropbox-API-Arg", string(cursorJSON))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload session append failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (d *DropboxUploader) finishUploadSession(sessionID string, targetPath string, offset int64) error {
	const uploadSessionFinishURL = "https://content.dropboxapi.com/2/files/upload_session/finish"

	finishArg := struct {
		Cursor struct {
			SessionID string `json:"session_id"`
			Offset    int64  `json:"offset"`
		} `json:"cursor"`
		Commit struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		} `json:"commit"`
	}{
		Cursor: struct {
			SessionID string `json:"session_id"`
			Offset    int64  `json:"offset"`
		}{
			SessionID: sessionID,
			Offset:    offset,
		},
		Commit: struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
		}{
			Path: targetPath,
			Mode: "add",
		},
	}

	finishArgJSON, err := json.Marshal(finishArg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", uploadSessionFinishURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Dropbox-API-Arg", string(finishArgJSON))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload session finish failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (d *DropboxUploader) uploadSmallFile(file *os.File, targetPath string) error {
	// Create API argument
	apiArg := DropboxAPIArg{
		Path:           targetPath,
		Mode:           "add",
		AutoRename:     true,
		Mute:           false,
		StrictConflict: false,
	}

	apiArgJSON, err := json.Marshal(apiArg)
	if err != nil {
		return fmt.Errorf("failed to marshal API argument: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", dropboxUploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Dropbox-API-Arg", string(apiArgJSON))

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
