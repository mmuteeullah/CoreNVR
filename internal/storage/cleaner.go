package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mmuteeullah/CoreNVR/internal/config"
)

// DiskAlert levels
const (
	DiskAlertNone     = 0
	DiskAlertWarning  = 1 // 80% full
	DiskAlertCritical = 2 // 90% full
	DiskAlertEmergency = 3 // 95% full
)

// Cleaner handles deletion of old recordings
type Cleaner struct {
	config config.StorageConfig
	logger *log.Logger
	lastAlertLevel int
	lastAlertTime  time.Time
	slackWebhook   string
}

// NewCleaner creates a new storage cleaner
func NewCleaner(cfg config.StorageConfig, slackWebhook string) *Cleaner {
	return &Cleaner{
		config: cfg,
		logger: log.New(os.Stdout, "[Storage] ", log.LstdFlags),
		slackWebhook: slackWebhook,
		lastAlertLevel: DiskAlertNone,
	}
}

// Start begins the cleanup and monitoring routine
func (c *Cleaner) Start(interval time.Duration) {
	c.logger.Printf("Starting storage manager (retention: %d days, monitoring interval: %v)",
		c.config.RetentionDays, interval)

	// Run initial disk usage check
	c.MonitorDiskUsage()

	// Run cleanup if retention is enabled
	if c.config.RetentionDays > 0 {
		c.cleanup()
	}

	// Monitor disk usage every interval
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			// Check disk usage first
			c.MonitorDiskUsage()

			// Then run regular cleanup if enabled
			if c.config.RetentionDays > 0 {
				c.cleanup()
			}
		}
	}()
}

// cleanup removes old recordings
func (c *Cleaner) cleanup() {
	c.logger.Println("Running cleanup...")

	cutoffTime := time.Now().AddDate(0, 0, -c.config.RetentionDays)
	deletedDirs := 0
	freedBytes := int64(0)

	// Walk through the base directory
	err := filepath.Walk(c.config.BasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue with other files
		}

		// Skip the base directory itself
		if path == c.config.BasePath {
			return nil
		}

		// Check if this is a date directory (YYYY-MM-DD format)
		if info.IsDir() {
			dirName := filepath.Base(path)
			if len(dirName) == 10 && dirName[4] == '-' && dirName[7] == '-' {
				// Parse the date
				dirDate, err := time.Parse("2006-01-02", dirName)
				if err == nil && dirDate.Before(cutoffTime) {
					// Calculate size before deletion
					size := c.getDirSize(path)

					// Delete the entire directory
					if err := os.RemoveAll(path); err != nil {
						c.logger.Printf("Failed to delete %s: %v", path, err)
					} else {
						deletedDirs++
						freedBytes += size
						c.logger.Printf("Deleted old directory: %s", path)
					}

					// Skip walking into this directory
					return filepath.SkipDir
				}
			}
		}

		return nil
	})

	if err != nil {
		c.logger.Printf("Cleanup error: %v", err)
	}

	if deletedDirs > 0 {
		c.logger.Printf("Cleanup complete: deleted %d directories, freed %.2f GB",
			deletedDirs, float64(freedBytes)/(1024*1024*1024))
	} else {
		c.logger.Println("Cleanup complete: nothing to delete")
	}
}

// getDirSize calculates the total size of a directory
func (c *Cleaner) getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

// GetDiskUsage returns current disk usage statistics
func GetDiskUsage(path string) (used, available uint64, percentUsed float64, err error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get disk stats: %w", err)
	}

	// Calculate disk usage
	total := stat.Blocks * uint64(stat.Bsize)
	available = stat.Bavail * uint64(stat.Bsize)
	used = total - available
	percentUsed = 100.0 * float64(used) / float64(total)

	return used, available, percentUsed, nil
}

// MonitorDiskUsage checks disk usage and triggers alerts/cleanup if needed
func (c *Cleaner) MonitorDiskUsage() {
	used, available, percentUsed, err := GetDiskUsage(c.config.BasePath)
	if err != nil {
		c.logger.Printf("Error checking disk usage: %v", err)
		return
	}

	usedGB := float64(used) / (1024 * 1024 * 1024)
	availableGB := float64(available) / (1024 * 1024 * 1024)

	c.logger.Printf("Disk usage: %.1f%% (%.2f GB used, %.2f GB available)",
		percentUsed, usedGB, availableGB)

	// Determine alert level
	currentAlertLevel := DiskAlertNone
	if percentUsed >= 95 {
		currentAlertLevel = DiskAlertEmergency
	} else if percentUsed >= 90 {
		currentAlertLevel = DiskAlertCritical
	} else if percentUsed >= 80 {
		currentAlertLevel = DiskAlertWarning
	}

	// Send alert if level increased or it's been > 1 hour since last alert
	timeSinceLastAlert := time.Since(c.lastAlertTime)
	if currentAlertLevel > c.lastAlertLevel ||
	   (currentAlertLevel > DiskAlertNone && timeSinceLastAlert > 1*time.Hour) {

		c.sendDiskAlert(currentAlertLevel, percentUsed, availableGB)
		c.lastAlertLevel = currentAlertLevel
		c.lastAlertTime = time.Now()
	}

	// Emergency cleanup if disk is critically low
	if currentAlertLevel == DiskAlertEmergency {
		c.logger.Println("⚠️  EMERGENCY: Disk usage at 95%+, triggering emergency cleanup")
		c.emergencyCleanup(availableGB)
	} else if currentAlertLevel == DiskAlertCritical {
		c.logger.Println("⚠️  WARNING: Disk usage at 90%+, running cleanup")
		c.cleanup()
	}
}

// emergencyCleanup deletes oldest recordings until we have at least 10% free space
func (c *Cleaner) emergencyCleanup(currentAvailableGB float64) {
	c.logger.Println("Starting emergency cleanup...")

	// Get target: at least 10% free space
	var stat syscall.Statfs_t
	if err := syscall.Statfs(c.config.BasePath, &stat); err != nil {
		c.logger.Printf("Failed to get disk stats: %v", err)
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	targetFree := float64(total) * 0.10 // 10% free

	// Find all date directories
	type DirInfo struct {
		path string
		date time.Time
		size int64
	}

	var dirs []DirInfo
	filepath.Walk(c.config.BasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == c.config.BasePath || !info.IsDir() {
			return nil
		}

		dirName := filepath.Base(path)
		if len(dirName) == 10 && dirName[4] == '-' && dirName[7] == '-' {
			if dirDate, err := time.Parse("2006-01-02", dirName); err == nil {
				size := c.getDirSize(path)
				dirs = append(dirs, DirInfo{path: path, date: dirDate, size: size})
				return filepath.SkipDir
			}
		}
		return nil
	})

	// Sort by date (oldest first)
	for i := 0; i < len(dirs)-1; i++ {
		for j := i + 1; j < len(dirs); j++ {
			if dirs[i].date.After(dirs[j].date) {
				dirs[i], dirs[j] = dirs[j], dirs[i]
			}
		}
	}

	// Delete oldest directories until we have enough space
	freedBytes := int64(0)
	deletedCount := 0

	for _, dir := range dirs {
		// Check current free space
		syscall.Statfs(c.config.BasePath, &stat)
		currentFree := float64(stat.Bavail * uint64(stat.Bsize))

		if currentFree >= targetFree {
			break
		}

		// Don't delete today's or yesterday's recordings
		if dir.date.After(time.Now().AddDate(0, 0, -2)) {
			c.logger.Printf("Skipping recent directory: %s", dir.path)
			continue
		}

		c.logger.Printf("Emergency deleting: %s (%.2f GB)", dir.path, float64(dir.size)/(1024*1024*1024))
		if err := os.RemoveAll(dir.path); err != nil {
			c.logger.Printf("Failed to delete %s: %v", dir.path, err)
		} else {
			freedBytes += dir.size
			deletedCount++
		}
	}

	c.logger.Printf("Emergency cleanup complete: deleted %d directories, freed %.2f GB",
		deletedCount, float64(freedBytes)/(1024*1024*1024))

	c.sendSlackMessage(fmt.Sprintf("🚨 *Emergency Cleanup Completed*\n" +
		"Deleted %d directories\n" +
		"Freed %.2f GB of space", deletedCount, float64(freedBytes)/(1024*1024*1024)))
}

// sendDiskAlert sends a Slack notification about disk usage
func (c *Cleaner) sendDiskAlert(level int, percentUsed, availableGB float64) {
	var emoji, levelName string

	switch level {
	case DiskAlertWarning:
		emoji = "⚠️"
		levelName = "WARNING"
	case DiskAlertCritical:
		emoji = "🔴"
		levelName = "CRITICAL"
	case DiskAlertEmergency:
		emoji = "🚨"
		levelName = "EMERGENCY"
	default:
		return
	}

	message := fmt.Sprintf("%s *Disk Usage %s*\n" +
		"Usage: %.1f%%\n" +
		"Available: %.2f GB\n" +
		"Path: %s",
		emoji, levelName, percentUsed, availableGB, c.config.BasePath)

	c.logger.Println(message)
	c.sendSlackMessage(message)
}

// sendSlackMessage sends a message to Slack webhook
func (c *Cleaner) sendSlackMessage(message string) {
	if c.slackWebhook == "" {
		return
	}

	// Send in background to avoid blocking
	go func() {
		payload := map[string]string{"text": message}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			c.logger.Printf("Failed to marshal Slack message: %v", err)
			return
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post(c.slackWebhook, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			c.logger.Printf("Failed to send Slack alert: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.logger.Printf("Slack webhook returned status: %d", resp.StatusCode)
		}
	}()
}