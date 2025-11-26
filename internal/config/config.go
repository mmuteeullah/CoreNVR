package config

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration
type Config struct {
	Storage       StorageConfig        `yaml:"storage"`
	Cameras       []CameraConfig       `yaml:"cameras"`
	System        SystemConfig         `yaml:"system"`
	WebUI         WebUIConfig          `yaml:"webui"`
	Notifications NotificationsConfig  `yaml:"notifications"`
	Recovery      RecoveryConfig       `yaml:"recovery"`
}

// StorageConfig defines storage settings
type StorageConfig struct {
	BasePath         string `yaml:"base_path"`
	SegmentDuration  int    `yaml:"segment_duration"`  // seconds
	RetentionDays    int    `yaml:"retention_days"`
}

// CameraConfig defines camera settings
type CameraConfig struct {
	Name       string `yaml:"name"`
	URL        string `yaml:"url"`
	Enabled    bool   `yaml:"enabled"`
	RetryDelay int    `yaml:"retry_delay"`  // seconds
	MaxRetries int    `yaml:"max_retries"`  // -1 for infinite
}

// SystemConfig defines system settings
type SystemConfig struct {
	LogLevel            string `yaml:"log_level"`
	LogFile             string `yaml:"log_file"`
	HealthCheckInterval int    `yaml:"health_check_interval"`
}

// WebUIConfig defines web interface settings
type WebUIConfig struct {
	Enabled        bool       `yaml:"enabled"`
	Port           int        `yaml:"port"`
	Authentication AuthConfig `yaml:"authentication"`
}

// AuthConfig defines authentication settings
type AuthConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Username        string `yaml:"username"`
	PasswordHash    string `yaml:"password_hash"`    // bcrypt hash
	SessionTimeout  int    `yaml:"session_timeout"`  // minutes (default: 60)
	SecretKey       string `yaml:"secret_key"`       // for signing sessions
}

// NotificationsConfig defines notification settings
type NotificationsConfig struct {
	Enabled          bool   `yaml:"enabled"`
	Type             string `yaml:"type"`
	TelegramBotToken string `yaml:"telegram_bot_token"`
	TelegramChatID   string `yaml:"telegram_chat_id"`
	GotifyURL        string `yaml:"gotify_url"`
	GotifyToken      string `yaml:"gotify_token"`
}

// RecoveryConfig defines camera recovery settings
type RecoveryConfig struct {
	Enabled                   bool            `yaml:"enabled"`
	StaleThreshold            int             `yaml:"stale_threshold"`            // seconds (default: 600)
	VerificationDelay         int             `yaml:"verification_delay"`         // seconds (default: 120)
	HealthCheckInterval       int             `yaml:"health_check_interval"`      // seconds (default: 60)
	MaxPowerCyclesPer30Min    int             `yaml:"max_power_cycles_per_30min"` // default: 2
	ServiceRestartTimeout     int             `yaml:"service_restart_timeout"`    // seconds (default: 30)
	PowerCycleRecoveryTimeout int             `yaml:"power_cycle_recovery_timeout"` // seconds (default: 60)
	SmartPlug                 SmartPlugConfig `yaml:"smartplug"`
	SlackWebhook              string          `yaml:"slack_webhook"`
}

// SmartPlugConfig defines Tuya smart plug settings
type SmartPlugConfig struct {
	DeviceID       string `yaml:"device_id"`
	IP             string `yaml:"ip"`
	LocalKey       string `yaml:"local_key"`
	Version        string `yaml:"version"`
	PowerOffDelay  int    `yaml:"power_off_duration"` // seconds (default: 10)
	PythonScript   string `yaml:"python_script"`       // path to plug.py (optional)
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if c.Storage.BasePath == "" {
		return fmt.Errorf("storage.base_path is required")
	}

	if c.Storage.SegmentDuration < 60 {
		return fmt.Errorf("segment_duration must be at least 60 seconds")
	}

	enabledCameras := 0
	for _, cam := range c.Cameras {
		if cam.Enabled {
			enabledCameras++
			if cam.URL == "" {
				return fmt.Errorf("camera %s: URL is required", cam.Name)
			}
		}
	}

	if enabledCameras == 0 {
		return fmt.Errorf("at least one camera must be enabled")
	}

	return nil
}