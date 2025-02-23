// retention.go
package main

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Backup struct {
	Path     string
	DateTime time.Time
}

func parseBackupDateTime(filename string) (time.Time, error) {
	// Remove .7z extension if present
	filename = strings.TrimSuffix(filename, ".7z")

	// Try to parse the timestamp
	return time.Parse("20060102.150405", filename)
}

func manageRetention(uploader *DropboxUploader, backupPath string, policy RetentionPolicy) error {
	logHeader("🧹 Managing backup retention...")

	// List all backups in the directory
	files, err := uploader.ListFiles(backupPath)
	if err != nil {
		return err
	}

	// Parse files into backup objects
	var backups []Backup
	for _, file := range files {
		// Get just the filename from the full path
		filename := filepath.Base(file)

		// Parse timestamp from filename
		t, err := parseBackupDateTime(filename)
		if err != nil {
			logSubStep("⚠️  Skipping file with invalid format: %s", filename)
			continue
		}
		backups = append(backups, Backup{Path: file, DateTime: t})
	}

	if len(backups) == 0 {
		logStep("ℹ️  No backups found to process")
		return nil
	}

	// Sort backups by date (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].DateTime.After(backups[j].DateTime)
	})

	// Determine which backups to keep
	toKeep := make(map[string]bool)

	// Keep daily backups
	dailyMap := make(map[string]bool)
	dailyCount := 0
	for _, b := range backups {
		dateKey := b.DateTime.Format("2006-01-02")
		if !dailyMap[dateKey] && dailyCount < policy.KeepDaily {
			dailyMap[dateKey] = true
			toKeep[b.Path] = true
			dailyCount++
			logSubStep("📌 Keeping daily backup: %s", filepath.Base(b.Path))
		}
	}

	// Keep weekly backups
	weeklyMap := make(map[string]bool)
	weeklyCount := 0
	for _, b := range backups {
		weekKey := b.DateTime.Format("2006-W%V")
		if !weeklyMap[weekKey] && weeklyCount < policy.KeepWeekly && !toKeep[b.Path] {
			weeklyMap[weekKey] = true
			toKeep[b.Path] = true
			weeklyCount++
			logSubStep("📌 Keeping weekly backup: %s", filepath.Base(b.Path))
		}
	}

	// Keep monthly backups
	monthlyMap := make(map[string]bool)
	monthlyCount := 0
	for _, b := range backups {
		monthKey := b.DateTime.Format("2006-01")
		if !monthlyMap[monthKey] && monthlyCount < policy.KeepMonthly && !toKeep[b.Path] {
			monthlyMap[monthKey] = true
			toKeep[b.Path] = true
			monthlyCount++
			logSubStep("📌 Keeping monthly backup: %s", filepath.Base(b.Path))
		}
	}

	// Keep yearly backups
	yearlyMap := make(map[string]bool)
	yearlyCount := 0
	for _, b := range backups {
		yearKey := b.DateTime.Format("2006")
		if !yearlyMap[yearKey] && yearlyCount < policy.KeepYearly && !toKeep[b.Path] {
			yearlyMap[yearKey] = true
			toKeep[b.Path] = true
			yearlyCount++
			logSubStep("📌 Keeping yearly backup: %s", filepath.Base(b.Path))
		}
	}

	// Delete backups that are not in the toKeep map
	deletedCount := 0
	for _, backup := range backups {
		if !toKeep[backup.Path] {
			logSubStep("🗑️  Deleting old backup: %s", filepath.Base(backup.Path))
			if err := uploader.DeleteFile(backup.Path); err != nil {
				logSubStep("⚠️  Failed to delete backup %s: %v", filepath.Base(backup.Path), err)
			} else {
				deletedCount++
			}
		}
	}

	logStep("✅ Retention management completed. Kept %d backups, deleted %d backups", len(toKeep), deletedCount)
	return nil
}
