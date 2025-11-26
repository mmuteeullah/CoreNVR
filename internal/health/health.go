package health

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Status represents the health check status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CameraHealth represents health status of a single camera
type CameraHealth struct {
	Name           string    `json:"name"`
	Status         Status    `json:"status"`
	Recording      bool      `json:"recording"`
	LiveStream     bool      `json:"live_stream"`
	LastSegment    time.Time `json:"last_segment"`
	SegmentCount   int       `json:"segment_count"`
	ErrorCount     int       `json:"error_count"`
	LastError      string    `json:"last_error,omitempty"`
}

// SystemHealth represents system resource health
type SystemHealth struct {
	CPUUsage       float64 `json:"cpu_usage_percent"`
	MemoryUsed     uint64  `json:"memory_used_bytes"`
	MemoryTotal    uint64  `json:"memory_total_bytes"`
	MemoryPercent  float64 `json:"memory_percent"`
	LoadAverage    [3]float64 `json:"load_average"`
	Uptime         int64   `json:"uptime_seconds"`
	GoRoutines     int     `json:"goroutines"`
}

// StorageHealth represents storage health
type StorageHealth struct {
	Path           string  `json:"path"`
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	Mounted        bool    `json:"mounted"`
	Writable       bool    `json:"writable"`
}

// HealthResponse represents the complete health check response
type HealthResponse struct {
	Status      Status               `json:"status"`
	Timestamp   time.Time            `json:"timestamp"`
	Version     string               `json:"version"`
	Uptime      time.Duration        `json:"uptime"`
	System      SystemHealth         `json:"system"`
	Storage     StorageHealth        `json:"storage"`
	Cameras     []CameraHealth       `json:"cameras"`
	Checks      map[string]bool      `json:"checks"`
	Messages    []string             `json:"messages,omitempty"`
}

// Monitor handles health monitoring
type Monitor struct {
	mu              sync.RWMutex
	startTime       time.Time
	version         string
	storagePath     string
	cameras         map[string]*CameraHealth
	lastCheck       time.Time
	errorThreshold  int
}

// NewMonitor creates a new health monitor
func NewMonitor(version, storagePath string) *Monitor {
	return &Monitor{
		startTime:      time.Now(),
		version:        version,
		storagePath:    storagePath,
		cameras:        make(map[string]*CameraHealth),
		errorThreshold: 5,
	}
}

// RegisterCamera registers a camera for health monitoring
func (m *Monitor) RegisterCamera(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.cameras[name]; !exists {
		m.cameras[name] = &CameraHealth{
			Name:   name,
			Status: StatusHealthy,
		}
	}
}

// UpdateCameraStatus updates the status of a camera
func (m *Monitor) UpdateCameraStatus(name string, recording, liveStream bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if camera, exists := m.cameras[name]; exists {
		camera.Recording = recording
		camera.LiveStream = liveStream
		camera.LastSegment = time.Now()

		if recording && liveStream {
			camera.Status = StatusHealthy
			camera.ErrorCount = 0
		} else if recording || liveStream {
			camera.Status = StatusDegraded
		} else {
			camera.Status = StatusUnhealthy
		}
	}
}

// ReportCameraError reports an error for a camera
func (m *Monitor) ReportCameraError(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if camera, exists := m.cameras[name]; exists {
		camera.ErrorCount++
		camera.LastError = err.Error()

		if camera.ErrorCount >= m.errorThreshold {
			camera.Status = StatusUnhealthy
			camera.Recording = false
			camera.LiveStream = false
		}
	}
}

// GetSystemHealth returns current system health metrics
func (m *Monitor) GetSystemHealth() SystemHealth {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	health := SystemHealth{
		MemoryUsed:    memStats.Alloc,
		MemoryTotal:   memStats.Sys,
		MemoryPercent: float64(memStats.Alloc) * 100 / float64(memStats.Sys),
		GoRoutines:    runtime.NumGoroutine(),
		Uptime:        int64(time.Since(m.startTime).Seconds()),
	}

	// Get load average
	if loadavg, err := ioutil.ReadFile("/proc/loadavg"); err == nil {
		var l1, l5, l15 float64
		fmt.Sscanf(string(loadavg), "%f %f %f", &l1, &l5, &l15)
		health.LoadAverage = [3]float64{l1, l5, l15}
	}

	// Get CPU usage (simplified - just using load average for now)
	if runtime.NumCPU() > 0 {
		health.CPUUsage = health.LoadAverage[0] * 100 / float64(runtime.NumCPU())
	}

	return health
}

// GetStorageHealth returns storage health metrics
func (m *Monitor) GetStorageHealth() StorageHealth {
	health := StorageHealth{
		Path:     m.storagePath,
		Mounted:  false,
		Writable: false,
	}

	// Check if path exists
	if _, err := os.Stat(m.storagePath); err == nil {
		health.Mounted = true

		// Check if writable
		testFile := filepath.Join(m.storagePath, ".health_check")
		if file, err := os.Create(testFile); err == nil {
			file.Close()
			os.Remove(testFile)
			health.Writable = true
		}

		// Get disk usage
		var stat syscall.Statfs_t
		if err := syscall.Statfs(m.storagePath, &stat); err == nil {
			health.TotalBytes = stat.Blocks * uint64(stat.Bsize)
			health.AvailableBytes = stat.Bavail * uint64(stat.Bsize)
			health.UsedBytes = health.TotalBytes - health.AvailableBytes
			health.UsagePercent = float64(health.UsedBytes) * 100 / float64(health.TotalBytes)
		}
	}

	return health
}

// Check performs a complete health check
func (m *Monitor) Check() HealthResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	response := HealthResponse{
		Timestamp: time.Now(),
		Version:   m.version,
		Uptime:    time.Since(m.startTime),
		System:    m.GetSystemHealth(),
		Storage:   m.GetStorageHealth(),
		Cameras:   make([]CameraHealth, 0),
		Checks:    make(map[string]bool),
		Messages:  make([]string, 0),
	}

	// Add camera health
	for _, camera := range m.cameras {
		response.Cameras = append(response.Cameras, *camera)
	}

	// Perform checks
	response.Checks["storage_mounted"] = response.Storage.Mounted
	response.Checks["storage_writable"] = response.Storage.Writable
	response.Checks["storage_space"] = response.Storage.UsagePercent < 90

	allCamerasHealthy := true
	anyCameraRecording := false

	for _, camera := range response.Cameras {
		if camera.Status != StatusHealthy {
			allCamerasHealthy = false
		}
		if camera.Recording {
			anyCameraRecording = true
		}
	}

	response.Checks["cameras_healthy"] = allCamerasHealthy
	response.Checks["cameras_recording"] = anyCameraRecording

	// Determine overall status
	if !response.Storage.Mounted || !response.Storage.Writable {
		response.Status = StatusUnhealthy
		response.Messages = append(response.Messages, "Storage not accessible")
	} else if response.Storage.UsagePercent >= 95 {
		response.Status = StatusUnhealthy
		response.Messages = append(response.Messages, "Storage critically full")
	} else if response.Storage.UsagePercent >= 90 {
		response.Status = StatusDegraded
		response.Messages = append(response.Messages, "Storage nearly full")
	} else if !allCamerasHealthy {
		response.Status = StatusDegraded
		response.Messages = append(response.Messages, "Some cameras are not healthy")
	} else {
		response.Status = StatusHealthy
	}

	m.lastCheck = time.Now()
	return response
}

// HTTPHandler returns an HTTP handler for health checks
func (m *Monitor) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := m.Check()

		// Set appropriate status code
		statusCode := http.StatusOK
		switch health.Status {
		case StatusDegraded:
			statusCode = http.StatusOK // Still return 200 for degraded
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		}

		// Support both simple and detailed responses
		if r.URL.Query().Get("detail") == "true" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(health)
		} else {
			// Simple response for basic health checks
			w.WriteHeader(statusCode)
			if health.Status == StatusHealthy {
				w.Write([]byte("ok"))
			} else {
				w.Write([]byte(string(health.Status)))
			}
		}
	}
}

// StartHealthServer starts a dedicated health check HTTP server
func (m *Monitor) StartHealthServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", m.HTTPHandler())
	mux.HandleFunc("/", m.HTTPHandler()) // Default route also serves health

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash
			fmt.Printf("Health server error: %v\n", err)
		}
	}()
}

// BackgroundMonitor runs periodic health checks in the background
func (m *Monitor) BackgroundMonitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			health := m.Check()

			// Log warnings if unhealthy
			if health.Status != StatusHealthy {
				fmt.Printf("Health check warning: %s - %v\n", health.Status, health.Messages)
			}

		case <-ctx.Done():
			return
		}
	}
}