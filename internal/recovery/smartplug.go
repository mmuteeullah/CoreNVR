package recovery

import (
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/mmuteeullah/CoreNVR/internal/config"
)

// SmartPlug controls a Tuya smart plug for camera power cycling
type SmartPlug struct {
	config config.SmartPlugConfig
	mutex  sync.Mutex
	logger *log.Logger
}

// NewSmartPlug creates a new SmartPlug instance
func NewSmartPlug(cfg config.SmartPlugConfig, logger *log.Logger) (*SmartPlug, error) {
	// Set default Python script path if not specified
	if cfg.PythonScript == "" {
		cfg.PythonScript = "/opt/BashNVR/plug.py"
	}

	sp := &SmartPlug{
		config: cfg,
		logger: logger,
	}

	// Test connection by checking if Python script exists
	if _, err := exec.LookPath("python3"); err != nil {
		return nil, fmt.Errorf("python3 not found: %w", err)
	}

	logger.Printf("Smart plug configured at %s (using Python tinytuya)", cfg.IP)

	return sp, nil
}

// PowerCycle turns the plug off, waits, then turns it back on
func (sp *SmartPlug) PowerCycle() error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.logger.Println("🔌 Power-cycling camera smart plug...")

	// Create inline Python script for power cycling
	pythonCode := fmt.Sprintf(`
import tinytuya
import time

plug = tinytuya.OutletDevice(
    dev_id="%s",
    address="%s",
    local_key="%s",
    version=%s
)

print("Turning OFF camera...")
plug.turn_off()
time.sleep(%d)
print("Turning ON camera...")
plug.turn_on()
print("Power cycle complete")
`, sp.config.DeviceID, sp.config.IP, sp.config.LocalKey, sp.config.Version, sp.config.PowerOffDelay)

	// Execute Python script
	cmd := exec.Command("python3", "-c", pythonCode)
	output, err := cmd.CombinedOutput()

	sp.logger.Printf("Smart plug output: %s", output)

	if err != nil {
		return fmt.Errorf("power cycling failed: %w - %s", err, output)
	}

	sp.logger.Println("✅ Camera plug power-cycled successfully")
	return nil
}

// TurnOn turns the plug on
func (sp *SmartPlug) TurnOn() error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.logger.Println("Turning ON smart plug...")

	pythonCode := fmt.Sprintf(`
import tinytuya
plug = tinytuya.OutletDevice("%s", "%s", "%s", version=%s)
plug.turn_on()
`, sp.config.DeviceID, sp.config.IP, sp.config.LocalKey, sp.config.Version)

	cmd := exec.Command("python3", "-c", pythonCode)
	return cmd.Run()
}

// TurnOff turns the plug off
func (sp *SmartPlug) TurnOff() error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.logger.Println("Turning OFF smart plug...")

	pythonCode := fmt.Sprintf(`
import tinytuya
plug = tinytuya.OutletDevice("%s", "%s", "%s", version=%s)
plug.turn_off()
`, sp.config.DeviceID, sp.config.IP, sp.config.LocalKey, sp.config.Version)

	cmd := exec.Command("python3", "-c", pythonCode)
	return cmd.Run()
}

// GetStatus returns the current on/off status
func (sp *SmartPlug) GetStatus() (bool, error) {
	pythonCode := fmt.Sprintf(`
import tinytuya
import json
plug = tinytuya.OutletDevice("%s", "%s", "%s", version=%s)
status = plug.status()
print(json.dumps(status))
`, sp.config.DeviceID, sp.config.IP, sp.config.LocalKey, sp.config.Version)

	cmd := exec.Command("python3", "-c", pythonCode)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("getting status: %w", err)
	}

	// For now, just return true if no error
	// Could parse JSON output for detailed status
	return len(output) > 0, nil
}
