package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mmuteeullah/CoreNVR/internal/config"
	"github.com/mmuteeullah/CoreNVR/internal/recorder"
	"github.com/mmuteeullah/CoreNVR/internal/recovery"
	"github.com/mmuteeullah/CoreNVR/internal/storage"
	"github.com/mmuteeullah/CoreNVR/internal/webui"
)

var (
	version = "0.1.0"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "/etc/corenvr/config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	testPlug := flag.String("test-plug", "", "Test smart plug: on|off|status|cycle")
	flag.Parse()

	if *showVersion {
		fmt.Printf("CoreNVR v%s - Lightweight NVR for Raspberry Pi\n", version)
		os.Exit(0)
	}

	// Test plug command
	if *testPlug != "" {
		testSmartPlug(*configPath, *testPlug)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup logging
	setupLogging(cfg.System)

	log.Printf("Starting CoreNVR v%s", version)
	log.Printf("Base storage path: %s", cfg.Storage.BasePath)
	log.Printf("Segment duration: %d seconds", cfg.Storage.SegmentDuration)
	log.Printf("Retention: %d days", cfg.Storage.RetentionDays)

	// Create base storage directory
	if err := os.MkdirAll(cfg.Storage.BasePath, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start storage cleaner with monitoring
	slackWebhook := ""
	if cfg.Recovery.Enabled && cfg.Recovery.SlackWebhook != "" {
		slackWebhook = cfg.Recovery.SlackWebhook
	}
	cleaner := storage.NewCleaner(cfg.Storage, slackWebhook)
	cleaner.Start(10 * time.Minute) // Check disk usage every 10 minutes

	// Start recorders for each enabled camera
	var wg sync.WaitGroup
	recorders := make([]*recorder.Recorder, 0)

	for _, cam := range cfg.Cameras {
		if !cam.Enabled {
			log.Printf("Camera %s is disabled, skipping", cam.Name)
			continue
		}

		log.Printf("Starting recorder for camera: %s", cam.Name)
		rec := recorder.New(cam, cfg.Storage)
		recorders = append(recorders, rec)

		wg.Add(1)
		go func(r *recorder.Recorder) {
			defer wg.Done()
			r.Start(ctx)
		}(rec)
	}

	// Start health monitor if configured
	if cfg.System.HealthCheckInterval > 0 {
		go startHealthMonitor(ctx, cfg, recorders)
	}

	// Start recovery manager if enabled
	if cfg.Recovery.Enabled {
		log.Println("Initializing camera recovery system...")
		recoveryMgr, err := recovery.NewRecoveryManager(&cfg.Recovery, recorders, ctx)
		if err != nil {
			log.Printf("WARNING: Failed to initialize recovery manager: %v", err)
			log.Println("Continuing without automatic recovery...")
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				recoveryMgr.Start(ctx)
			}()
			log.Println("✅ Camera recovery system active")
		}
	}

	// Start web UI if enabled
	if cfg.WebUI.Enabled {
		webServer := webui.NewServer(cfg, cfg.WebUI.Port)
		webServer.Start()
		log.Printf("Web UI available at http://0.0.0.0:%d", cfg.WebUI.Port)
	}

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	// Cancel context to stop all recorders
	cancel()

	// Stop all recorders gracefully
	for _, rec := range recorders {
		rec.Stop()
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		log.Println("All recorders stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for recorders to stop")
	}

	log.Println("CoreNVR shutdown complete")
}

// setupLogging configures the logging system
func setupLogging(cfg config.SystemConfig) {
	// Set log level flags based on config
	logFlags := log.LstdFlags

	if cfg.LogLevel == "debug" {
		logFlags |= log.Lshortfile
	}

	log.SetFlags(logFlags)

	// If log file is specified, create/open it
	if cfg.LogFile != "" {
		logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("Failed to open log file %s: %v, using stdout", cfg.LogFile, err)
		} else {
			log.SetOutput(logFile)
		}
	}
}

// startHealthMonitor runs periodic health checks
func startHealthMonitor(ctx context.Context, cfg *config.Config, recorders []*recorder.Recorder) {
	ticker := time.NewTicker(time.Duration(cfg.System.HealthCheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			performHealthCheck(cfg, recorders)
		}
	}
}

// performHealthCheck checks system health
func performHealthCheck(cfg *config.Config, recorders []*recorder.Recorder) {
	// Check disk space
	_, err := os.Stat(cfg.Storage.BasePath)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}

	// Get disk usage (simplified - you'd want proper implementation)
	var stat syscall.Statfs_t
	err = syscall.Statfs(cfg.Storage.BasePath, &stat)
	if err == nil {
		available := stat.Bavail * uint64(stat.Bsize)
		total := stat.Blocks * uint64(stat.Bsize)
		usedPercent := 100.0 * float64(total-available) / float64(total)

		if usedPercent > 90 {
			log.Printf("WARNING: Disk usage critical: %.1f%% used", usedPercent)
			// Send notification if configured
			sendNotification(cfg, fmt.Sprintf("Disk usage critical: %.1f%% used", usedPercent))
		} else if usedPercent > 80 {
			log.Printf("WARNING: Disk usage high: %.1f%% used", usedPercent)
		}
	}

	// Check if recordings are being created
	for _, cam := range cfg.Cameras {
		if !cam.Enabled {
			continue
		}

		cameraPath := fmt.Sprintf("%s/%s/%s", cfg.Storage.BasePath, cam.Name, time.Now().Format("2006-01-02"))
		if info, err := os.Stat(cameraPath); err != nil || time.Since(info.ModTime()) > 10*time.Minute {
			log.Printf("WARNING: No recent recordings for camera %s", cam.Name)
			sendNotification(cfg, fmt.Sprintf("No recent recordings for camera %s", cam.Name))
		}
	}
}

// sendNotification sends alerts if configured
func sendNotification(cfg *config.Config, message string) {
	if !cfg.Notifications.Enabled {
		return
	}

	// Implement lightweight notification
	// For now, just log it
	log.Printf("NOTIFICATION: %s", message)

	// TODO: Implement Telegram or Gotify notifications
	// These are much lighter than Slack for Raspberry Pi
}

// testSmartPlug tests the smart plug configuration
func testSmartPlug(configPath, command string) {
	fmt.Println("🔌 Testing Smart Plug Integration...")
	fmt.Println("=====================================")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	if !cfg.Recovery.Enabled {
		log.Fatal("❌ Recovery system is not enabled in config")
	}

	// Create smart plug instance
	logger := log.New(os.Stdout, "[SmartPlug Test] ", log.LstdFlags)
	plug, err := recovery.NewSmartPlug(cfg.Recovery.SmartPlug, logger)
	if err != nil {
		log.Fatalf("❌ Failed to initialize smart plug: %v", err)
	}

	fmt.Printf("\n✅ Smart plug initialized: %s\n\n", cfg.Recovery.SmartPlug.IP)

	// Execute command
	switch command {
	case "on":
		fmt.Println("📍 Command: Turn ON")
		if err := plug.TurnOn(); err != nil {
			log.Fatalf("❌ Failed to turn on: %v", err)
		}
		fmt.Println("✅ Plug turned ON successfully")

	case "off":
		fmt.Println("📍 Command: Turn OFF")
		if err := plug.TurnOff(); err != nil {
			log.Fatalf("❌ Failed to turn off: %v", err)
		}
		fmt.Println("✅ Plug turned OFF successfully")

	case "status":
		fmt.Println("📍 Command: Get Status")
		online, err := plug.GetStatus()
		if err != nil {
			log.Fatalf("❌ Failed to get status: %v", err)
		}
		if online {
			fmt.Println("✅ Plug is ONLINE and responsive")
		} else {
			fmt.Println("⚠️  Plug is OFFLINE or not responding")
		}

	case "cycle":
		fmt.Println("📍 Command: Power Cycle")
		fmt.Printf("⏳ This will turn OFF the plug for %d seconds\n", cfg.Recovery.SmartPlug.PowerOffDelay)
		fmt.Println("⚠️  WARNING: Camera will lose power!")
		fmt.Print("\nPress Enter to continue or Ctrl+C to cancel...")
		fmt.Scanln()

		if err := plug.PowerCycle(); err != nil {
			log.Fatalf("❌ Failed to power cycle: %v", err)
		}
		fmt.Println("\n✅ Power cycle completed successfully")

	default:
		log.Fatalf("❌ Unknown command: %s\nValid commands: on, off, status, cycle", command)
	}

	fmt.Println("\n=====================================")
	fmt.Println("✅ Smart plug test completed")
}