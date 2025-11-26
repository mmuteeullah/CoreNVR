package webui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mmuteeullah/CoreNVR/internal/auth"
	"github.com/mmuteeullah/CoreNVR/internal/config"
)

// Server represents the web UI server
type Server struct {
	config         *config.Config
	port           int
	logger         *log.Logger
	sessionManager *auth.SessionManager
	authEnabled    bool
}

// NewServer creates a new web UI server
func NewServer(cfg *config.Config, port int) *Server {
	var sessionManager *auth.SessionManager
	authEnabled := cfg.WebUI.Authentication.Enabled

	if authEnabled {
		sessionManager = auth.NewSessionManager(
			cfg.WebUI.Authentication.Username,
			cfg.WebUI.Authentication.PasswordHash,
			cfg.WebUI.Authentication.SecretKey,
			cfg.WebUI.Authentication.SessionTimeout,
		)
	}

	return &Server{
		config:         cfg,
		port:           port,
		logger:         log.New(os.Stdout, "[WebUI] ", log.LstdFlags),
		sessionManager: sessionManager,
		authEnabled:    authEnabled,
	}
}

// Start begins the web server
func (s *Server) Start() {
	if s.authEnabled {
		// Public routes (no authentication required)
		http.HandleFunc("/login", s.handleLogin)
		http.HandleFunc("/logout", s.handleLogout)
		http.HandleFunc("/health", s.handleHealth)

		// Protected routes (require authentication)
		http.HandleFunc("/api/status", s.requireAuth(s.handleAPIStatus))
		http.HandleFunc("/api/cameras", s.requireAuth(s.handleAPICameras))
		http.HandleFunc("/api/storage", s.requireAuth(s.handleAPIStorage))
		http.HandleFunc("/api/recordings/", s.requireAuth(s.handleRecordingsAPI))
		http.HandleFunc("/stream/", s.requireAuth(s.handleStream))
		http.HandleFunc("/segments/", s.requireAuth(s.handleSegments))
		http.HandleFunc("/recordings/", s.requireAuth(s.handleRecordingPlayback))
		http.HandleFunc("/", s.requireAuth(s.handleIndex))

		s.logger.Println("Authentication enabled")
	} else {
		// No authentication - all routes public
		http.HandleFunc("/api/status", s.handleAPIStatus)
		http.HandleFunc("/api/cameras", s.handleAPICameras)
		http.HandleFunc("/api/storage", s.handleAPIStorage)
		http.HandleFunc("/api/recordings/", s.handleRecordingsAPI)
		http.HandleFunc("/health", s.handleHealth)
		http.HandleFunc("/stream/", s.handleStream)
		http.HandleFunc("/segments/", s.handleSegments)
		http.HandleFunc("/recordings/", s.handleRecordingPlayback)
		http.HandleFunc("/", s.handleIndex)

		s.logger.Println("Authentication disabled - public access")
	}

	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Printf("Starting web UI on http://0.0.0.0%s", addr)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			s.logger.Printf("Web server error: %v", err)
		}
	}()
}

// requireAuth is middleware that requires authentication
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return s.sessionManager.AuthMiddleware(next)
}

// handleIndex serves the main HTML page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlTemplate))
}

// handleAPIStatus returns system status as JSON
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	var stat syscall.Statfs_t
	diskUsage := 0.0

	if err := syscall.Statfs(s.config.Storage.BasePath, &stat); err == nil {
		available := stat.Bavail * uint64(stat.Bsize)
		total := stat.Blocks * uint64(stat.Bsize)
		diskUsage = 100.0 * float64(total-available) / float64(total)
	}

	status := map[string]interface{}{
		"status":           "running",
		"storage_path":     s.config.Storage.BasePath,
		"disk_usage":       fmt.Sprintf("%.1f", diskUsage),
		"retention_days":   s.config.Storage.RetentionDays,
		"segment_duration": s.config.Storage.SegmentDuration,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleAPICameras returns camera status as JSON
func (s *Server) handleAPICameras(w http.ResponseWriter, r *http.Request) {
	cameras := []map[string]interface{}{}

	for _, cam := range s.config.Cameras {
		// Check if camera is recording by looking at today's recording folder
		recordingsPath := filepath.Join(s.config.Storage.BasePath, cam.Name, "recordings")
		todayPath := filepath.Join(recordingsPath, time.Now().Format("2006-01-02"))

		isRecording := false
		lastFile := ""
		var lastModTime time.Time

		// Check if today's folder exists and has recent files
		if _, err := os.Stat(todayPath); err == nil {
			// Get all .ts files in today's folder
			files, _ := filepath.Glob(filepath.Join(todayPath, "*.ts"))
			if len(files) > 0 {
				lastFile = filepath.Base(files[len(files)-1])

				// Check if the last file was modified recently (within 5 minutes)
				if info, err := os.Stat(files[len(files)-1]); err == nil {
					lastModTime = info.ModTime()
					if time.Since(lastModTime) < 5*time.Minute {
						isRecording = true
					}
				}
			}
		}

		cameras = append(cameras, map[string]interface{}{
			"name":          cam.Name,
			"enabled":       cam.Enabled,
			"recording":     isRecording,
			"last_file":     lastFile,
			"last_modified": lastModTime.Format("15:04:05"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cameras)
}

// handleHealth simple health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleAPIStorage returns detailed storage statistics
func (s *Server) handleAPIStorage(w http.ResponseWriter, r *http.Request) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(s.config.Storage.BasePath, &stat); err != nil {
		http.Error(w, "Failed to get disk stats", http.StatusInternalServerError)
		return
	}

	// Calculate disk usage
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - available
	percentUsed := 100.0 * float64(used) / float64(total)

	// Determine alert level
	alertLevel := "normal"
	if percentUsed >= 95 {
		alertLevel = "emergency"
	} else if percentUsed >= 90 {
		alertLevel = "critical"
	} else if percentUsed >= 80 {
		alertLevel = "warning"
	}

	// Get per-camera storage info
	cameras := []map[string]interface{}{}
	for _, cam := range s.config.Cameras {
		if !cam.Enabled {
			continue
		}

		cameraBasePath := filepath.Join(s.config.Storage.BasePath, cam.Name, "recordings")
		cameraSize := s.getDirSize(cameraBasePath)

		// Count recording days
		days := 0
		if entries, err := os.ReadDir(cameraBasePath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && len(entry.Name()) == 10 {
					days++
				}
			}
		}

		cameras = append(cameras, map[string]interface{}{
			"name":         cam.Name,
			"size_bytes":   cameraSize,
			"size_gb":      fmt.Sprintf("%.2f", float64(cameraSize)/(1024*1024*1024)),
			"days_stored":  days,
		})
	}

	response := map[string]interface{}{
		"total_bytes":      total,
		"used_bytes":       used,
		"available_bytes":  available,
		"total_gb":         fmt.Sprintf("%.2f", float64(total)/(1024*1024*1024)),
		"used_gb":          fmt.Sprintf("%.2f", float64(used)/(1024*1024*1024)),
		"available_gb":     fmt.Sprintf("%.2f", float64(available)/(1024*1024*1024)),
		"percent_used":     fmt.Sprintf("%.1f", percentUsed),
		"alert_level":      alertLevel,
		"retention_days":   s.config.Storage.RetentionDays,
		"cameras":          cameras,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getDirSize calculates total size of a directory
func (s *Server) getDirSize(path string) int64 {
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

// handleRecordingsAPI routes /api/recordings/* requests to appropriate handlers
func (s *Server) handleRecordingsAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	s.logger.Printf("handleRecordingsAPI called with path: %s", path)

	if strings.HasPrefix(path, "/api/recordings/playlist/") {
		s.logger.Println("Routing to handleRecordingPlaylist")
		s.handleRecordingPlaylist(w, r)
	} else if strings.HasPrefix(path, "/api/recordings/timeline") {
		s.logger.Println("Routing to handleRecordingsTimeline")
		s.handleRecordingsTimeline(w, r)
	} else if strings.HasPrefix(path, "/api/recordings/list") {
		s.logger.Println("Routing to handleRecordingsList")
		s.handleRecordingsList(w, r)
	} else if strings.HasPrefix(path, "/api/recordings/dates") {
		s.logger.Println("Routing to handleRecordingDates")
		s.handleRecordingDates(w, r)
	} else {
		s.logger.Printf("No route matched for: %s", path)
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleRecordingDates returns list of dates with recordings
func (s *Server) handleRecordingDates(w http.ResponseWriter, r *http.Request) {
	camera := r.URL.Query().Get("camera")
	if camera == "" {
		http.Error(w, "camera parameter required", http.StatusBadRequest)
		return
	}

	recordingsPath := filepath.Join(s.config.Storage.BasePath, camera, "recordings")
	entries, err := os.ReadDir(recordingsPath)
	if err != nil {
		http.Error(w, "Failed to read recordings", http.StatusInternalServerError)
		return
	}

	var dates []string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 10 {
			// Verify it's a valid date format YYYY-MM-DD
			if _, err := time.Parse("2006-01-02", entry.Name()); err == nil {
				dates = append(dates, entry.Name())
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"camera": camera,
		"dates":  dates,
	})
}

// handleRecordingsList returns list of recordings for a specific date
func (s *Server) handleRecordingsList(w http.ResponseWriter, r *http.Request) {
	camera := r.URL.Query().Get("camera")
	date := r.URL.Query().Get("date")

	if camera == "" || date == "" {
		http.Error(w, "camera and date parameters required", http.StatusBadRequest)
		return
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", date); err != nil {
		http.Error(w, "invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	datePath := filepath.Join(s.config.Storage.BasePath, camera, "recordings", date)
	files, err := filepath.Glob(filepath.Join(datePath, "*.ts"))
	if err != nil {
		http.Error(w, "Failed to list recordings", http.StatusInternalServerError)
		return
	}

	type Recording struct {
		Filename    string `json:"filename"`
		StartTime   string `json:"start_time"`
		Size        int64  `json:"size"`
		SizeMB      string `json:"size_mb"`
		Duration    int    `json:"duration_seconds"`
		URL         string `json:"url"`
		PlaylistURL string `json:"playlist_url"`
	}

	recordings := []Recording{}
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		filename := filepath.Base(file)
		// Parse time from filename (HH-MM-SS.ts)
		timeStr := filename[:8] // HH-MM-SS
		startTime := date + " " + timeStr[:2] + ":" + timeStr[3:5] + ":" + timeStr[6:8]

		recordings = append(recordings, Recording{
			Filename:    filename,
			StartTime:   startTime,
			Size:        info.Size(),
			SizeMB:      fmt.Sprintf("%.2f", float64(info.Size())/(1024*1024)),
			Duration:    1800, // 30 minutes in seconds
			URL:         fmt.Sprintf("/recordings/%s/%s/%s", camera, date, filename),
			PlaylistURL: fmt.Sprintf("/api/recordings/playlist/%s/%s/%s", camera, date, filename),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"camera":     camera,
		"date":       date,
		"count":      len(recordings),
		"recordings": recordings,
	})
}

// handleRecordingsTimeline returns timeline with gap detection
func (s *Server) handleRecordingsTimeline(w http.ResponseWriter, r *http.Request) {
	camera := r.URL.Query().Get("camera")
	date := r.URL.Query().Get("date")

	if camera == "" || date == "" {
		http.Error(w, "camera and date parameters required", http.StatusBadRequest)
		return
	}

	// Validate date format
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		http.Error(w, "invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	datePath := filepath.Join(s.config.Storage.BasePath, camera, "recordings", date)
	files, err := filepath.Glob(filepath.Join(datePath, "*.ts"))
	if err != nil {
		http.Error(w, "Failed to list recordings", http.StatusInternalServerError)
		return
	}

	type Segment struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Filename  string `json:"filename"`
		SizeMB    string `json:"size_mb"`
	}

	type Gap struct {
		StartTime    string `json:"start_time"`
		EndTime      string `json:"end_time"`
		DurationMins int    `json:"duration_mins"`
	}

	segments := []Segment{}
	gaps := []Gap{}

	// Parse all segments
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		filename := filepath.Base(file)
		timeStr := filename[:8]

		startTime, _ := time.Parse("2006-01-02 15:04:05",
			fmt.Sprintf("%s %s:%s:%s", date, timeStr[:2], timeStr[3:5], timeStr[6:8]))
		endTime := startTime.Add(30 * time.Minute)

		// Cap end time to 23:59:59 if it wraps to next day
		endTimeStr := endTime.Format("15:04:05")
		if endTime.Day() != startTime.Day() {
			endTimeStr = "23:59:59"
		}

		segments = append(segments, Segment{
			StartTime: startTime.Format("15:04:05"),
			EndTime:   endTimeStr,
			Filename:  filename,
			SizeMB:    fmt.Sprintf("%.2f", float64(info.Size())/(1024*1024)),
		})
	}

	// Sort segments by start time
	for i := 0; i < len(segments)-1; i++ {
		for j := i + 1; j < len(segments); j++ {
			if segments[i].StartTime > segments[j].StartTime {
				segments[i], segments[j] = segments[j], segments[i]
			}
		}
	}

	// Detect gaps
	if len(segments) > 0 {
		// Check for gap at start of day (00:00 to first segment)
		dayStart := time.Date(dateObj.Year(), dateObj.Month(), dateObj.Day(), 0, 0, 0, 0, time.UTC)
		firstStart, _ := time.Parse("15:04:05", segments[0].StartTime)
		firstStart = time.Date(dateObj.Year(), dateObj.Month(), dateObj.Day(),
			firstStart.Hour(), firstStart.Minute(), firstStart.Second(), 0, time.UTC)

		startGapDuration := firstStart.Sub(dayStart)
		if startGapDuration > 2*time.Minute {
			gaps = append(gaps, Gap{
				StartTime:    "00:00:00",
				EndTime:      segments[0].StartTime,
				DurationMins: int(startGapDuration.Minutes()),
			})
		}

		// Check for gaps between consecutive segments
		for i := 0; i < len(segments)-1; i++ {
			currentEnd, _ := time.Parse("15:04:05", segments[i].EndTime)
			nextStart, _ := time.Parse("15:04:05", segments[i+1].StartTime)

			// Add date context for comparison
			currentEnd = time.Date(dateObj.Year(), dateObj.Month(), dateObj.Day(),
				currentEnd.Hour(), currentEnd.Minute(), currentEnd.Second(), 0, time.UTC)
			nextStart = time.Date(dateObj.Year(), dateObj.Month(), dateObj.Day(),
				nextStart.Hour(), nextStart.Minute(), nextStart.Second(), 0, time.UTC)

			gapDuration := nextStart.Sub(currentEnd)
			if gapDuration > 2*time.Minute {
				gaps = append(gaps, Gap{
					StartTime:    segments[i].EndTime,
					EndTime:      segments[i+1].StartTime,
					DurationMins: int(gapDuration.Minutes()),
				})
			}
		}

		// Check for gap at end of day (last segment to 23:59)
		dayEnd := time.Date(dateObj.Year(), dateObj.Month(), dateObj.Day(), 23, 59, 59, 0, time.UTC)
		lastEnd, _ := time.Parse("15:04:05", segments[len(segments)-1].EndTime)
		lastEnd = time.Date(dateObj.Year(), dateObj.Month(), dateObj.Day(),
			lastEnd.Hour(), lastEnd.Minute(), lastEnd.Second(), 0, time.UTC)

		endGapDuration := dayEnd.Sub(lastEnd)
		if endGapDuration > 2*time.Minute {
			gaps = append(gaps, Gap{
				StartTime:    segments[len(segments)-1].EndTime,
				EndTime:      "23:59:59",
				DurationMins: int(endGapDuration.Minutes()),
			})
		}
	} else {
		// No recordings at all = entire day is a gap
		gaps = append(gaps, Gap{
			StartTime:    "00:00:00",
			EndTime:      "23:59:59",
			DurationMins: 1440,
		})
	}

	// Calculate coverage based on actual recorded time (not just segment count)
	totalMinutes := 24 * 60
	recordedMinutes := 0

	if len(segments) > 0 {
		// Calculate total gap time
		totalGapMinutes := 0
		for _, gap := range gaps {
			totalGapMinutes += gap.DurationMins
		}

		// Recorded time = Total time - Gap time
		recordedMinutes = totalMinutes - totalGapMinutes
	}

	coveragePercent := float64(recordedMinutes) / float64(totalMinutes) * 100

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"camera":           camera,
		"date":             date,
		"segments":         segments,
		"gaps":             gaps,
		"total_segments":   len(segments),
		"total_gaps":       len(gaps),
		"coverage_percent": fmt.Sprintf("%.1f", coveragePercent),
		"recorded_hours":   fmt.Sprintf("%.1f", float64(recordedMinutes)/60),
	})
}

// handleRecordingPlayback serves recorded video files
func (s *Server) handleRecordingPlayback(w http.ResponseWriter, r *http.Request) {
	// URL format: /recordings/{camera}/{date}/{filename}
	// Expected: /recordings/imou_cruiser/2025-11-22/15-30-00.ts

	trimmed := r.URL.Path[len("/recordings/"):]

	// Parse camera name (until first /)
	var camera, date, filename string
	idx1 := -1
	for i, c := range trimmed {
		if c == '/' {
			camera = trimmed[:i]
			idx1 = i + 1
			break
		}
	}

	if idx1 == -1 || camera == "" {
		http.Error(w, "Invalid path: camera not found", http.StatusBadRequest)
		return
	}

	// Parse date (10 chars: YYYY-MM-DD)
	if len(trimmed) > idx1+10 && trimmed[idx1+10] == '/' {
		date = trimmed[idx1 : idx1+10]
		filename = trimmed[idx1+11:]
	} else {
		http.Error(w, "Invalid path: date not found", http.StatusBadRequest)
		return
	}

	if date == "" || filename == "" {
		http.Error(w, "Invalid recording path", http.StatusBadRequest)
		return
	}

	// Validate date
	if _, err := time.Parse("2006-01-02", date); err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Validate filename (should end with .ts)
	if filepath.Ext(filename) != ".ts" {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.config.Storage.BasePath, camera, "recordings", date, filename)

	// Security: ensure path doesn't escape storage directory
	baseDir := filepath.Join(s.config.Storage.BasePath, camera, "recordings")
	absBase, _ := filepath.Abs(baseDir)
	absPath, err := filepath.Abs(filePath)
	if err != nil || !filepath.HasPrefix(absPath, absBase) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Set appropriate content type
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Accept-Ranges", "bytes")

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// handleRecordingPlaylist generates an HLS playlist for a single recording file
// URL format: /api/recordings/playlist/{camera}/{date}/{filename}
func (s *Server) handleRecordingPlaylist(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Playlist request: %s", r.URL.Path)

	// Parse URL path
	trimmed := r.URL.Path[len("/api/recordings/playlist/"):]

	// Extract camera, date, and filename
	var camera, date, filename string
	parts := []string{}
	current := ""
	for i, ch := range trimmed {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
		if i == len(trimmed)-1 && current != "" {
			parts = append(parts, current)
		}
	}

	if len(parts) != 3 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	camera = parts[0]
	date = parts[1]
	filename = parts[2]

	// Verify file exists
	filePath := filepath.Join(s.config.Storage.BasePath, camera, "recordings", date, filename)

	// Security check
	baseDir := filepath.Join(s.config.Storage.BasePath, camera, "recordings")
	absBase, _ := filepath.Abs(baseDir)
	absPath, err := filepath.Abs(filePath)
	if err != nil || !filepath.HasPrefix(absPath, absBase) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Recording not found", http.StatusNotFound)
		return
	}

	// Generate HLS playlist for this single file
	recordingURL := fmt.Sprintf("/recordings/%s/%s/%s", camera, date, filename)

	playlist := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:1800
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:1800.0,
%s
#EXT-X-ENDLIST
`, recordingURL)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(playlist))
}

// handleLogin handles the login page and authentication
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Show login page
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(loginTemplate))
		return
	}

	if r.Method == http.MethodPost {
		// Process login
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid form data"})
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		remember := r.FormValue("remember") == "on"

		// Authenticate
		if !s.sessionManager.Authenticate(username, password) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username or password"})
			return
		}

		// Create session
		sessionID, err := s.sessionManager.CreateSession(username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
			return
		}

		// Set cookie
		maxAge := 0 // Session cookie (expires when browser closes)
		if remember {
			maxAge = 30 * 24 * 60 * 60 // 30 days
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			MaxAge:   maxAge,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		s.logger.Printf("User '%s' logged in successfully", username)

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// handleLogout handles user logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Destroy session
		s.sessionManager.DestroySession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	s.logger.Println("User logged out")

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}