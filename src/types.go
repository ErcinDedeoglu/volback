package main

// Volume represents the structure of a volume in the output
type Volume struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
	Type        string `json:"type"`
	Name        string `json:"name"`
}

// VolumeResult represents the structure of docker-volume command output
type VolumeResult struct {
	ContainerName string   `json:"containerName"`
	Status        string   `json:"status"`
	Error         string   `json:"error,omitempty"`
	Volumes       []Volume `json:"volumes"`
}

// ControlResult represents the structure of docker-control command output
type ControlResult struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ArchiverResult represents the structure of archiver command output
type ArchiverResult struct {
	Path        string `json:"path"`
	ArchivePath string `json:"archivePath"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type RetentionPolicy struct {
	KeepDaily   int
	KeepWeekly  int
	KeepMonthly int
	KeepYearly  int
}

type ContainerConfig struct {
	Container string   `json:"container"`
	BackupID  string   `json:"backup_id"`
	Stop      bool     `json:"stop"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type ContainerConfigs []ContainerConfig
