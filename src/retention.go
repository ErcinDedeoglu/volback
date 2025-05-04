// retention.go
package main

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Backup struct {
	Path     string
	DateTime time.Time
}

var validFilePattern = regexp.MustCompile(`^[0-9]{8}\.[0-9]{6}`)

func manageRetention(uploader *DropboxUploader, backupPath string, policy RetentionPolicy) error {
	logHeader("🧹 Managing backup retention...")

	files, err := uploader.ListFiles(backupPath)
	if err != nil {
		return err
	}

	var backups []Backup
	for _, file := range files {
		filename := filepath.Base(file)
		filename = strings.TrimSuffix(filename, ".7z")
		filename = strings.TrimSuffix(filename, ".sql")
		filename = strings.TrimSuffix(filename, ".bak")

		if !validFilePattern.MatchString(filename) {
			logSubStep("⚠️  Skipping invalid filename: %s", filename)
			continue
		}

		t, err := time.Parse("20060102.150405", filename)
		if err != nil {
			logSubStep("⚠️  Skipping unparseable file: %s (%v)", filename, err)
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

	toKeep := make(map[string]bool)
	backupCategories := make(map[string]string) // Track which category each backup belongs to

	// Always keep most recent backup
	mostRecent := backups[0]
	toKeep[mostRecent.Path] = true
	backupCategories[mostRecent.Path] = "most_recent"
	logSubStep("📌 Keeping most recent backup: %s", filepath.Base(mostRecent.Path))

	// Helper function to check if backup belongs to a different period
	isDifferentPeriod := func(t1, t2 time.Time, format string) bool {
		return t1.Format(format) != t2.Format(format)
	}

	// Process remaining backups
	remainingBackups := backups[1:]

	// Daily retention (different day than most recent)
	for _, b := range remainingBackups {
		if !toKeep[b.Path] && isDifferentPeriod(b.DateTime, mostRecent.DateTime, "2006-01-02") {
			toKeep[b.Path] = true
			backupCategories[b.Path] = "daily"
			logSubStep("📌 Keeping daily backup: %s (different day)", filepath.Base(b.Path))
			break
		}
	}

	// Weekly retention (different week than most recent)
	for _, b := range remainingBackups {
		if !toKeep[b.Path] && isDifferentPeriod(b.DateTime, mostRecent.DateTime, "2006-W02") {
			toKeep[b.Path] = true
			backupCategories[b.Path] = "weekly"
			logSubStep("📌 Keeping weekly backup: %s (different week)", filepath.Base(b.Path))
			break
		}
	}

	// Monthly retention (different month than most recent)
	for _, b := range remainingBackups {
		if !toKeep[b.Path] && isDifferentPeriod(b.DateTime, mostRecent.DateTime, "2006-01") {
			toKeep[b.Path] = true
			backupCategories[b.Path] = "monthly"
			logSubStep("📌 Keeping monthly backup: %s (different month)", filepath.Base(b.Path))
			break
		}
	}

	// Yearly retention (different year than most recent)
	for _, b := range remainingBackups {
		if !toKeep[b.Path] && isDifferentPeriod(b.DateTime, mostRecent.DateTime, "2006") {
			toKeep[b.Path] = true
			backupCategories[b.Path] = "yearly"
			logSubStep("📌 Keeping yearly backup: %s (different year)", filepath.Base(b.Path))
			break
		}
	}

	// Delete unneeded backups
	deletedCount := 0
	for _, backup := range backups {
		if !toKeep[backup.Path] {
			logSubStep("🗑️  Deleting backup: %s (from same period as existing backup)",
				filepath.Base(backup.Path))
			if err := uploader.DeleteFile(backup.Path); err != nil {
				logSubStep("⚠️  Failed to delete backup %s: %v", filepath.Base(backup.Path), err)
			} else {
				deletedCount++
			}
		}
	}

	// Count backups by their assigned categories
	counts := map[string]int{
		"daily":   0,
		"weekly":  0,
		"monthly": 0,
		"yearly":  0,
	}

	for _, category := range backupCategories {
		if category != "most_recent" {
			counts[category]++
		}
	}

	// Log detailed summary
	logStep("📊 Retention Summary:")
	logSubStep("Most Recent: 1")
	logSubStep("Daily: %d/%d (from different days)", counts["daily"], policy.KeepDaily)
	logSubStep("Weekly: %d/%d (from different weeks)", counts["weekly"], policy.KeepWeekly)
	logSubStep("Monthly: %d/%d (from different months)", counts["monthly"], policy.KeepMonthly)
	logSubStep("Yearly: %d/%d (from different years)", counts["yearly"], policy.KeepYearly)
	logStep("✅ Retention completed. Kept %d backups, deleted %d backups", len(toKeep), deletedCount)

	return nil
}
