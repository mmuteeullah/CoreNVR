package recovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/mmuteeullah/CoreNVR/internal/config"
	"github.com/mmuteeullah/CoreNVR/internal/recorder"
)

// CameraRecoveryState tracks recovery state for a camera
type CameraRecoveryState struct {
	cameraName        string
	failureDetectedAt time.Time
	lastRecoveryCheck time.Time
	recoveryAttempts  []RecoveryAttempt
	mutex             sync.Mutex
}

// RecoveryAttempt records a recovery action
type RecoveryAttempt struct {
	Timestamp time.Time
	Action    string // "goroutine_restart", "service_restart", "power_cycle"
	Success   bool
}

// RecoveryManager handles camera recovery
type RecoveryManager struct {
	config     *config.RecoveryConfig
	recorders  map[string]*recorder.Recorder
	states     map[string]*CameraRecoveryState
	smartPlug  *SmartPlug
	logger     *log.Logger
	parentCtx  context.Context
	mutex      sync.RWMutex
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(cfg *config.RecoveryConfig, recorders []*recorder.Recorder, parentCtx context.Context) (*RecoveryManager, error) {
	logger := log.New(os.Stdout, "[Recovery] ", log.LstdFlags)

	// Initialize smart plug
	plug, err := NewSmartPlug(cfg.SmartPlug, logger)
	if err != nil {
		return nil, fmt.Errorf("initializing smart plug: %w", err)
	}

	// Create recorder map
	recMap := make(map[string]*recorder.Recorder)
	states := make(map[string]*CameraRecoveryState)
	for _, rec := range recorders {
		name := rec.GetCameraName()
		recMap[name] = rec
		states[name] = &CameraRecoveryState{
			cameraName: name,
		}
	}

	rm := &RecoveryManager{
		config:     cfg,
		recorders:  recMap,
		states:     states,
		smartPlug:  plug,
		logger:     logger,
		parentCtx:  parentCtx,
	}

	logger.Println("✅ Recovery manager initialized")
	return rm, nil
}

// Start begins the recovery monitoring loop
func (rm *RecoveryManager) Start(ctx context.Context) {
	rm.logger.Println("Starting camera recovery monitor...")

	ticker := time.NewTicker(time.Duration(rm.config.HealthCheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rm.logger.Println("Recovery manager shutting down")
			return
		case <-ticker.C:
			rm.checkAllCameras()
		}
	}
}

// checkAllCameras checks health of all cameras
func (rm *RecoveryManager) checkAllCameras() {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	for cameraName, rec := range rm.recorders {
		if err := rm.checkCamera(cameraName, rec); err != nil {
			rm.logger.Printf("Error checking camera %s: %v", cameraName, err)
		}
	}
}

// checkCamera checks a single camera's health
func (rm *RecoveryManager) checkCamera(cameraName string, rec *recorder.Recorder) error {
	state := rm.states[cameraName]
	state.mutex.Lock()
	defer state.mutex.Unlock()

	// Get last recording time
	lastRecording := rec.GetLastRecordingTime()

	// Check if recording is stale
	if lastRecording.IsZero() {
		// No recordings at all - camera just started or serious issue
		rm.logger.Printf("⚠️  Camera %s: No recordings found yet", cameraName)
		return nil
	}

	age := time.Since(lastRecording)
	staleThreshold := time.Duration(rm.config.StaleThreshold) * time.Second

	if age < staleThreshold {
		// Recording is fresh - reset failure state
		if !state.failureDetectedAt.IsZero() {
			rm.logger.Printf("✅ Camera %s recovered! Recording is fresh (%v old)", cameraName, age.Round(time.Second))
			rm.sendAlert(fmt.Sprintf("✅ *Camera Recovered*\nCamera: `%s`\nRecording resumed successfully", cameraName))
			state.failureDetectedAt = time.Time{}
			state.recoveryAttempts = nil
		}
		return nil
	}

	// Recording is stale
	if state.failureDetectedAt.IsZero() {
		// First detection of stale recording
		state.failureDetectedAt = time.Now()
		rm.logger.Printf("⚠️  Camera %s: Stale recording detected (%v old)", cameraName, age.Round(time.Second))
		return nil
	}

	// Check verification delay (don't act immediately)
	timeSinceDetection := time.Since(state.failureDetectedAt)
	verificationDelay := time.Duration(rm.config.VerificationDelay) * time.Second

	if timeSinceDetection < verificationDelay {
		rm.logger.Printf("⏳ Camera %s: Waiting to verify failure is persistent (%v/%v)",
			cameraName,
			timeSinceDetection.Round(time.Second),
			verificationDelay)
		return nil
	}

	// Failure verified - attempt recovery
	rm.logger.Printf("🚨 Camera %s: Recording stale for %v, starting recovery...", cameraName, age.Round(time.Second))
	return rm.recoverCamera(cameraName, rec, state)
}

// recoverCamera attempts to recover a camera
func (rm *RecoveryManager) recoverCamera(cameraName string, rec *recorder.Recorder, state *CameraRecoveryState) error {
	// Check if we've exceeded power cycle limits
	recentPowerCycles := rm.countRecentAttempts(state, "power_cycle", 30*time.Minute)
	if recentPowerCycles >= rm.config.MaxPowerCyclesPer30Min {
		rm.sendAlert(fmt.Sprintf("🚨 *CRITICAL: Max Recovery Attempts Reached*\nCamera: `%s`\nExceeded %d power cycles in 30 minutes\nManual intervention required",
			cameraName, rm.config.MaxPowerCyclesPer30Min))
		return fmt.Errorf("max power cycles reached")
	}

	// Level 1: Try restarting camera goroutine
	if !rm.hasAttempted(state, "goroutine_restart") {
		return rm.restartCameraGoroutine(cameraName, rec, state)
	}

	// Level 2: Try restarting entire service
	if !rm.hasAttempted(state, "service_restart") {
		return rm.restartService(cameraName, state)
	}

	// Level 3: Power cycle camera
	if !rm.hasAttempted(state, "power_cycle") {
		return rm.powerCycleCamera(cameraName, state)
	}

	// All attempts exhausted
	rm.sendAlert(fmt.Sprintf("💀 *CRITICAL: All Recovery Attempts Failed*\nCamera: `%s`\nAll recovery methods exhausted\nImmediate attention required",
		cameraName))
	return fmt.Errorf("all recovery attempts failed")
}

// restartCameraGoroutine restarts just the camera's recorder goroutine
func (rm *RecoveryManager) restartCameraGoroutine(cameraName string, rec *recorder.Recorder, state *CameraRecoveryState) error {
	rm.logger.Printf("🔄 Level 1: Restarting recorder goroutine for %s", cameraName)
	rm.sendAlert(fmt.Sprintf("🔄 *Recovery Started*\nCamera: `%s`\nAction: Restarting recorder goroutine", cameraName))

	state.recoveryAttempts = append(state.recoveryAttempts, RecoveryAttempt{
		Timestamp: time.Now(),
		Action:    "goroutine_restart",
		Success:   false,
	})

	if err := rec.Restart(rm.parentCtx); err != nil {
		rm.logger.Printf("❌ Failed to restart goroutine: %v", err)
		return err
	}

	// Wait and verify
	time.Sleep(time.Duration(rm.config.ServiceRestartTimeout) * time.Second)
	return nil
}

// restartService restarts the entire CoreNVR service
func (rm *RecoveryManager) restartService(cameraName string, state *CameraRecoveryState) error {
	rm.logger.Printf("🔄 Level 2: Restarting CoreNVR service for %s", cameraName)
	rm.sendAlert(fmt.Sprintf("🔄 *Escalating Recovery*\nCamera: `%s`\nAction: Restarting CoreNVR service", cameraName))

	state.recoveryAttempts = append(state.recoveryAttempts, RecoveryAttempt{
		Timestamp: time.Now(),
		Action:    "service_restart",
		Success:   false,
	})

	// Execute systemctl restart
	cmd := exec.Command("systemctl", "restart", "corenvr")
	if err := cmd.Run(); err != nil {
		rm.logger.Printf("❌ Failed to restart service: %v", err)
		return err
	}

	rm.logger.Println("Service restart initiated")
	return nil
}

// powerCycleCamera power cycles the camera via smart plug
func (rm *RecoveryManager) powerCycleCamera(cameraName string, state *CameraRecoveryState) error {
	rm.logger.Printf("🔌 Level 3: Power-cycling camera %s", cameraName)
	rm.sendAlert(fmt.Sprintf("🔌 *Power Cycle Initiated*\nCamera: `%s`\nAction: Cycling camera power via smart plug", cameraName))

	state.recoveryAttempts = append(state.recoveryAttempts, RecoveryAttempt{
		Timestamp: time.Now(),
		Action:    "power_cycle",
		Success:   false,
	})

	if err := rm.smartPlug.PowerCycle(); err != nil {
		rm.logger.Printf("❌ Failed to power cycle: %v", err)
		rm.sendAlert(fmt.Sprintf("❌ *Power Cycle Failed*\nCamera: `%s`\nError: %v", cameraName, err))
		return err
	}

	// Wait for camera to come back online
	rm.logger.Printf("⏳ Waiting %ds for camera to recover...", rm.config.PowerCycleRecoveryTimeout)
	time.Sleep(time.Duration(rm.config.PowerCycleRecoveryTimeout) * time.Second)

	return nil
}

// hasAttempted checks if a recovery action was already attempted
func (rm *RecoveryManager) hasAttempted(state *CameraRecoveryState, action string) bool {
	for _, attempt := range state.recoveryAttempts {
		if attempt.Action == action {
			return true
		}
	}
	return false
}

// countRecentAttempts counts attempts of a specific type within a time window
func (rm *RecoveryManager) countRecentAttempts(state *CameraRecoveryState, action string, window time.Duration) int {
	cutoff := time.Now().Add(-window)
	count := 0
	for _, attempt := range state.recoveryAttempts {
		if attempt.Action == action && attempt.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

// SlackMessage represents a Slack webhook payload
type SlackMessage struct {
	Text string `json:"text"`
}

// sendAlert sends a Slack notification
func (rm *RecoveryManager) sendAlert(message string) {
	if rm.config.SlackWebhook == "" {
		rm.logger.Printf("ALERT (no webhook): %s", message)
		return
	}

	// Log the alert
	rm.logger.Printf("ALERT: %s", message)

	// Send to Slack in background (don't block recovery)
	go rm.sendSlackMessage(message)
}

// sendSlackMessage sends a message to Slack via webhook
func (rm *RecoveryManager) sendSlackMessage(message string) {
	// Prepare payload
	payload := SlackMessage{
		Text: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		rm.logger.Printf("Error marshaling Slack message: %v", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", rm.config.SlackWebhook, bytes.NewBuffer(jsonData))
	if err != nil {
		rm.logger.Printf("Error creating Slack request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		rm.logger.Printf("Error sending Slack notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rm.logger.Printf("Slack webhook returned non-OK status: %d", resp.StatusCode)
		return
	}

	rm.logger.Printf("✅ Slack notification sent successfully")
}
