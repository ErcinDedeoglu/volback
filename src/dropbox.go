package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	// Validate inputs
	if sourcePath == "" || targetPath == "" {
		return fmt.Errorf("source and target paths are required")
	}

	if err := d.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %v", err)
	}

	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer file.Close()

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

	req, err := http.NewRequest("POST", dropboxUploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+d.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Dropbox-API-Arg", string(apiArgJSON))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
