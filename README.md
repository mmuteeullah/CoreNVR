# CoreNVR

A lightweight, reliable Network Video Recorder designed for Raspberry Pi and low-resource systems.

## Features

- **Dual-Stream Architecture** - Separate streams for recording (30-min segments) and live viewing (2-sec segments)
- **Low Latency Live View** - 2-5 second delay for real-time monitoring
- **Memory Efficient** - ~15-20MB per camera
- **Web Interface** - Full-featured UI with authentication
- **Automatic Cleanup** - Configurable retention policies
- **Recovery System** - Optional automatic camera recovery via smart plug
- **Single Binary** - Easy deployment with no runtime dependencies except FFmpeg

## Requirements

- Go 1.21+ (for building)
- FFmpeg (runtime dependency)
- Raspberry Pi 3B+ or newer (or any Linux system)
- USB HDD recommended for storage

## Quick Start

### 1. Build

```bash
# Clone the repository
git clone https://github.com/mmuteeullah/CoreNVR.git
cd CoreNVR

# Build for your platform
make build

# Or build for Raspberry Pi
make build-pi
```

### 2. Configure

```bash
# Copy example configuration
sudo mkdir -p /etc/corenvr
sudo cp configs/config.example.yaml /etc/corenvr/config.yaml

# Edit configuration
sudo nano /etc/corenvr/config.yaml
```

Update the following in your config:
- Camera RTSP URL
- Storage path
- Authentication credentials (generate hash with `make hashpass`)

### 3. Run

```bash
# Run directly
./corenvr -config /etc/corenvr/config.yaml

# Or install as service (see below)
```

### 4. Access Web UI

Open `http://your-ip:8080` in your browser.

## Installation as Service

```bash
# Create service file
sudo tee /etc/systemd/system/corenvr.service > /dev/null << 'EOF'
[Unit]
Description=CoreNVR Service
After=network.target

[Service]
Type=simple
ExecStart=/opt/corenvr/corenvr -config /etc/corenvr/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Install binary
sudo mkdir -p /opt/corenvr
sudo cp corenvr /opt/corenvr/

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable corenvr
sudo systemctl start corenvr
```

## Configuration

See `configs/config.example.yaml` for a complete example with all options documented.

### Key Settings

| Setting | Description |
|---------|-------------|
| `storage.base_path` | Where to store recordings |
| `storage.retention_days` | Auto-delete recordings older than this |
| `cameras[].url` | RTSP URL of your camera |
| `webui.port` | Web interface port (default: 8080) |
| `webui.authentication` | Enable/configure authentication |

### Authentication

Generate a password hash:

```bash
make hashpass
# Or: go run tools/hashpass.go YourPassword
```

Add the hash to your config file under `webui.authentication.password_hash`.

## Build Targets

```bash
make build        # Build for current platform
make build-pi     # Build for Raspberry Pi (arm64)
make build-pi32   # Build for Raspberry Pi (32-bit)
make build-all    # Build for all platforms
make clean        # Remove build artifacts
make deps         # Download dependencies
make hashpass     # Generate password hash
make help         # Show all targets
```

## Project Structure

```
CoreNVR/
├── cmd/corenvr/      # Application entry point
├── internal/
│   ├── auth/         # Authentication
│   ├── config/       # Configuration loading
│   ├── health/       # Health monitoring
│   ├── recorder/     # Recording & live streaming
│   ├── recovery/     # Camera recovery (optional)
│   ├── storage/      # Storage management
│   └── webui/        # Web interface
├── configs/          # Example configurations
├── tools/            # Utility tools
├── Makefile
└── README.md
```

## Recovery System (Optional)

CoreNVR can automatically recover cameras using a Tuya-compatible smart plug:

1. Detect stale recordings
2. Restart camera goroutine
3. Restart CoreNVR service
4. Power cycle camera via smart plug

Requires `python3` and `tinytuya` package:
```bash
pip3 install tinytuya
```

Enable in config with `recovery.enabled: true`.

## Logs

```bash
# View service logs
sudo journalctl -u corenvr -f

# Or check log file (if configured)
tail -f /var/log/corenvr/corenvr.log
```

## License

MIT License
