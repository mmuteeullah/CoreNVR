package recorder

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mmuteeullah/CoreNVR/internal/config"
)

// Recorder handles recording for a single camera
type Recorder struct {
	camera  config.CameraConfig
	storage config.StorageConfig
	logger  *log.Logger
	recordCmd     *exec.Cmd  // For long-term recording (30-min segments)
	liveStreamCmd *exec.Cmd  // For live streaming (2-sec segments)
	enableLive    bool       // Whether to enable live streaming
	ctx           context.Context
	cancel        context.CancelFunc
	lastSegmentTime time.Time  // Track last recording time
}

// New creates a new Recorder instance
func New(camera config.CameraConfig, storage config.StorageConfig) *Recorder {
	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", camera.Name), log.LstdFlags)

	return &Recorder{
		camera:     camera,
		storage:    storage,
		logger:     logger,
		enableLive: true,  // Enable live streaming by default
	}
}

// Start begins recording from the camera
func (r *Recorder) Start(ctx context.Context) {
	// Create internal context for this recorder instance
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Start recording stream (30-minute segments for storage)
	go r.startRecording(r.ctx)

	// Start live stream (2-second segments for web UI) if enabled
	if r.enableLive {
		go r.startLiveStream(r.ctx)
	}

	// Wait for context cancellation
	<-r.ctx.Done()
	r.logger.Println("Shutting down recorder")
}

// startRecording handles long-term recording with large segments
func (r *Recorder) startRecording(ctx context.Context) {
	retryCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check max retries
			if r.camera.MaxRetries >= 0 && retryCount >= r.camera.MaxRetries {
				r.logger.Printf("Recording: Max retries (%d) reached", r.camera.MaxRetries)
				return
			}

			// Start recording
			r.logger.Printf("Starting recording (attempt %d)", retryCount+1)
			err := r.record(ctx)

			if err != nil {
				r.logger.Printf("Recording failed: %v", err)
				retryCount++

				// Wait before retry
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(r.camera.RetryDelay) * time.Second):
					continue
				}
			}
		}
	}
}

// startLiveStream handles low-latency streaming for web UI
func (r *Recorder) startLiveStream(ctx context.Context) {
	retryCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if r.camera.MaxRetries >= 0 && retryCount >= r.camera.MaxRetries {
				r.logger.Printf("Live stream: Max retries (%d) reached", r.camera.MaxRetries)
				return
			}

			r.logger.Printf("Starting live stream (attempt %d)", retryCount+1)
			err := r.liveStream(ctx)

			if err != nil {
				r.logger.Printf("Live stream failed: %v", err)
				retryCount++

				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(r.camera.RetryDelay) * time.Second):
					continue
				}
			}
		}
	}
}

// record handles the actual FFmpeg recording
func (r *Recorder) record(ctx context.Context) error {
	// Create base recordings directory
	baseDir := filepath.Join(r.storage.BasePath, r.camera.Name, "recordings")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("creating base recordings directory: %w", err)
	}

	// Create today's date folder
	// FFmpeg's strftime won't create the directory, so we must do it
	dateStr := time.Now().Format("2006-01-02")
	dateDir := filepath.Join(baseDir, dateStr)
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return fmt.Errorf("creating date directory: %w", err)
	}

	// Start a goroutine to create tomorrow's folder at midnight
	go r.createNextDayFolder(ctx, baseDir)

	// Build FFmpeg command with strftime for FULL path (including date folder)
	// This ensures recordings go into the correct date folder even after midnight
	outputPattern := filepath.Join(baseDir, "%Y-%m-%d", "%H-%M-%S.ts")

	// FFmpeg arguments for RECORDING (30-minute segments)
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-rtsp_transport", "tcp",
		"-i", r.camera.URL,
		"-c:v", "copy",             // No video transcoding
		"-c:a", "copy",             // No audio transcoding
		"-f", "segment",
		"-segment_time", "1800",    // 30-minute segments for storage
		"-segment_format", "mpegts",
		"-segment_atclocktime", "1",
		"-reset_timestamps", "1",
		"-strftime", "1",           // Enable strftime for date-based folders
		outputPattern,
	}

	// Create command with context for proper cancellation
	r.recordCmd = exec.CommandContext(ctx, "ffmpeg", args...)

	// Log stderr for debugging
	r.recordCmd.Stderr = &logWriter{logger: r.logger, prefix: "REC"}

	// Start FFmpeg
	if err := r.recordCmd.Start(); err != nil {
		return fmt.Errorf("starting recording ffmpeg: %w", err)
	}

	r.logger.Println("📹 Recording started (30-min segments)")

	// Wait for completion or context cancellation
	return r.recordCmd.Wait()
}

// liveStream handles the live HLS streaming
func (r *Recorder) liveStream(ctx context.Context) error {
	// Create output directory for live stream
	liveDir := filepath.Join(r.storage.BasePath, r.camera.Name, "live")
	if err := os.MkdirAll(liveDir, 0755); err != nil {
		return fmt.Errorf("creating live directory: %w", err)
	}

	// HLS playlist and segments
	playlistPath := filepath.Join(liveDir, "stream.m3u8")
	segmentPattern := filepath.Join(liveDir, "segment%03d.ts")

	// FFmpeg arguments for LIVE STREAMING (2-second segments)
	args := []string{
		"-hide_banner",
		"-loglevel", "error",

		// Input options for low latency
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-rtsp_transport", "tcp",
		"-i", r.camera.URL,

		// Copy streams (no transcoding for efficiency)
		"-c:v", "copy",
		"-c:a", "copy",

		// HLS output with LOW LATENCY settings
		"-f", "hls",
		"-hls_time", "2",            // 2-second segments for low latency
		"-hls_list_size", "5",       // Keep only 5 segments (10 seconds)
		"-hls_flags", "delete_segments+append_list",
		"-hls_segment_type", "mpegts",
		"-hls_segment_filename", segmentPattern,
		"-hls_allow_cache", "0",

		playlistPath,
	}

	// Create command with context
	r.liveStreamCmd = exec.CommandContext(ctx, "ffmpeg", args...)

	// Log stderr for debugging
	r.liveStreamCmd.Stderr = &logWriter{logger: r.logger, prefix: "LIVE"}

	// Start FFmpeg
	if err := r.liveStreamCmd.Start(); err != nil {
		return fmt.Errorf("starting live stream ffmpeg: %w", err)
	}

	r.logger.Println("🔴 Live stream started (2-sec segments)")

	// Wait for completion or context cancellation
	return r.liveStreamCmd.Wait()
}

// Stop gracefully stops both recording and live streaming
func (r *Recorder) Stop() {
	// Cancel context first
	if r.cancel != nil {
		r.cancel()
	}

	// Stop recording
	if r.recordCmd != nil && r.recordCmd.Process != nil {
		r.logger.Println("Stopping recording...")
		r.recordCmd.Process.Signal(os.Interrupt)

		done := make(chan struct{})
		go func() {
			r.recordCmd.Wait()
			close(done)
		}()

		select {
		case <-done:
			r.logger.Println("Recording stopped gracefully")
		case <-time.After(5 * time.Second):
			r.logger.Println("Force killing recording")
			r.recordCmd.Process.Kill()
		}
	}

	// Stop live stream
	if r.liveStreamCmd != nil && r.liveStreamCmd.Process != nil {
		r.logger.Println("Stopping live stream...")
		r.liveStreamCmd.Process.Signal(os.Interrupt)

		done := make(chan struct{})
		go func() {
			r.liveStreamCmd.Wait()
			close(done)
		}()

		select {
		case <-done:
			r.logger.Println("Live stream stopped gracefully")
		case <-time.After(5 * time.Second):
			r.logger.Println("Force killing live stream")
			r.liveStreamCmd.Process.Kill()
		}
	}
}

// Restart stops and restarts the recorder
func (r *Recorder) Restart(parentCtx context.Context) error {
	r.logger.Println("🔄 Restarting recorder...")

	// Stop current recording
	r.Stop()

	// Brief pause to ensure clean shutdown
	time.Sleep(2 * time.Second)

	// Restart in new goroutine
	go r.Start(parentCtx)

	r.logger.Println("✅ Recorder restarted")
	return nil
}

// GetLastRecordingTime returns the time of the last recording segment
func (r *Recorder) GetLastRecordingTime() time.Time {
	// Check the most recent file in today's recording directory
	dateStr := time.Now().Format("2006-01-02")
	recordDir := filepath.Join(r.storage.BasePath, r.camera.Name, "recordings", dateStr)

	// Find the most recent .ts file
	entries, err := os.ReadDir(recordDir)
	if err != nil {
		// If today's folder doesn't exist, try yesterday (for overnight transitions)
		yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
		recordDir = filepath.Join(r.storage.BasePath, r.camera.Name, "recordings", yesterday)
		entries, err = os.ReadDir(recordDir)
		if err != nil {
			return time.Time{} // Return zero time if neither directory exists
		}
	}

	var latestTime time.Time
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ts" {
			info, err := entry.Info()
			if err == nil && info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
			}
		}
	}

	return latestTime
}

// GetCameraName returns the camera name
func (r *Recorder) GetCameraName() string {
	return r.camera.Name
}

// createNextDayFolder creates tomorrow's recording folder just before midnight
func (r *Recorder) createNextDayFolder(ctx context.Context, baseDir string) {
	for {
		// Calculate time until next midnight (from today, not 24 hours from now)
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

		// Wake up 1 minute before midnight to create tomorrow's folder
		wakeTime := nextMidnight.Add(-1 * time.Minute)
		sleepDuration := wakeTime.Sub(now)

		// If we're already past 23:59 (edge case during startup), wait until next cycle
		if sleepDuration < 0 {
			// We're past midnight, calculate for tomorrow's midnight
			nextMidnight = time.Date(now.Year(), now.Month(), now.Day()+2, 0, 0, 0, 0, now.Location())
			wakeTime = nextMidnight.Add(-1 * time.Minute)
			sleepDuration = wakeTime.Sub(now)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepDuration):
			// Create tomorrow's folder
			tomorrow := time.Now().Add(24 * time.Hour)
			tomorrowStr := tomorrow.Format("2006-01-02")
			tomorrowDir := filepath.Join(baseDir, tomorrowStr)
			if err := os.MkdirAll(tomorrowDir, 0755); err != nil {
				r.logger.Printf("Warning: Failed to create tomorrow's folder %s: %v", tomorrowStr, err)
			} else {
				r.logger.Printf("Created next day folder: %s", tomorrowStr)
			}

			// Sleep briefly after creation to avoid duplicate attempts
			time.Sleep(2 * time.Minute)
		}
	}
}

// getOutputDir returns the output directory for today's recordings
func (r *Recorder) getOutputDir() string {
	dateStr := time.Now().Format("2006-01-02")
	return filepath.Join(r.storage.BasePath, r.camera.Name, dateStr)
}

// logWriter wraps a logger for stderr output
type logWriter struct {
	logger *log.Logger
	prefix string
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	if lw.prefix != "" {
		lw.logger.Printf("[%s] FFmpeg: %s", lw.prefix, p)
	} else {
		lw.logger.Printf("FFmpeg: %s", p)
	}
	return len(p), nil
}