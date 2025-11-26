package webui

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>CoreNVR - Live View</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        :root {
            --bg-primary: #0f0f0f;
            --bg-secondary: #1a1a1a;
            --bg-tertiary: #242424;
            --border-color: #2a2a2a;
            --text-primary: #e8e8e8;
            --text-secondary: #a0a0a0;
            --accent-green: #10b981;
            --accent-green-dark: #059669;
            --accent-orange: #f59e0b;
            --accent-red: #ef4444;
            --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.3);
            --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.4);
            --shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.5);
            --radius-sm: 6px;
            --radius-md: 8px;
            --radius-lg: 12px;
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            overflow-x: hidden;
            line-height: 1.6;
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }

        .header {
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border-color);
            padding: 16px 24px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            box-shadow: var(--shadow-sm);
            position: sticky;
            top: 0;
            z-index: 100;
            backdrop-filter: blur(10px);
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
            font-size: 1.5em;
            font-weight: 700;
            color: var(--accent-green);
            letter-spacing: -0.5px;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: var(--accent-green);
            box-shadow: 0 0 10px var(--accent-green);
            animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
            0%, 100% {
                opacity: 1;
                transform: scale(1);
            }
            50% {
                opacity: 0.6;
                transform: scale(0.95);
            }
        }
        .controls {
            display: flex;
            gap: 8px;
        }

        .btn {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            padding: 10px 18px;
            border-radius: var(--radius-sm);
            cursor: pointer;
            font-size: 0.9em;
            font-weight: 500;
            transition: all 0.2s ease;
            position: relative;
            overflow: hidden;
        }

        .btn::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(90deg, transparent, rgba(255,255,255,0.1), transparent);
            transition: left 0.5s ease;
        }

        .btn:hover {
            background: var(--bg-tertiary);
            border-color: var(--accent-green);
            transform: translateY(-1px);
            box-shadow: var(--shadow-md);
        }

        .btn:hover::before {
            left: 100%;
        }

        .btn:active {
            transform: translateY(0);
        }

        .btn.active {
            background: var(--accent-green);
            border-color: var(--accent-green);
            color: white;
            box-shadow: 0 0 20px rgba(16, 185, 129, 0.3);
        }
        .container {
            padding: 24px;
            max-width: 1600px;
            margin: 0 auto;
        }

        .view-mode {
            display: none;
        }

        .view-mode.active {
            display: block;
        }

        .video-grid {
            display: grid;
            gap: 20px;
            margin-top: 20px;
        }

        .grid-1 { grid-template-columns: 1fr; }
        .grid-2 { grid-template-columns: repeat(2, 1fr); }
        .grid-4 { grid-template-columns: repeat(2, 1fr); }

        @media (max-width: 768px) {
            .grid-2, .grid-4 { grid-template-columns: 1fr; }
            .container { padding: 16px; }
            .header { padding: 12px 16px; }
        }

        .camera-container {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-lg);
            overflow: hidden;
            transition: all 0.3s ease;
            box-shadow: var(--shadow-sm);
        }

        .camera-container:hover {
            transform: translateY(-2px);
            box-shadow: var(--shadow-lg);
            border-color: var(--accent-green);
        }

        .camera-header {
            background: var(--bg-tertiary);
            padding: 14px 16px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid var(--border-color);
        }

        .camera-name {
            font-weight: 600;
            color: var(--accent-green);
            font-size: 1.05em;
            letter-spacing: -0.2px;
        }

        .camera-status {
            font-size: 0.875em;
            color: var(--text-secondary);
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 4px 10px;
            background: var(--bg-secondary);
            border-radius: 12px;
        }
        .video-wrapper {
            position: relative;
            padding-bottom: 56.25%; /* 16:9 */
            background: #000;
        }
        video {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: #000;
        }
        .video-overlay {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            text-align: center;
            color: #666;
        }
        .spinner {
            width: 40px;
            height: 40px;
            border: 3px solid #333;
            border-top-color: #4CAF50;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin: 0 auto 10px;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        .stats {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-lg);
            padding: 24px;
            margin-top: 24px;
            box-shadow: var(--shadow-md);
        }

        .stats h3 {
            color: var(--accent-green);
            margin-bottom: 16px;
            font-size: 1.25em;
            font-weight: 600;
            letter-spacing: -0.3px;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 16px;
            margin-top: 16px;
        }

        .stat-item {
            display: flex;
            flex-direction: column;
            gap: 8px;
            padding: 16px;
            background: var(--bg-tertiary);
            border-radius: var(--radius-md);
            border: 1px solid var(--border-color);
            transition: all 0.2s ease;
        }

        .stat-item:hover {
            background: var(--bg-secondary);
            border-color: var(--accent-green);
            transform: translateY(-2px);
            box-shadow: var(--shadow-md);
        }

        .stat-label {
            color: var(--text-secondary);
            font-size: 0.875em;
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .stat-value {
            font-size: 1.5em;
            font-weight: 700;
            color: var(--text-primary);
            letter-spacing: -0.5px;
        }
        .error-message {
            background: #2a1a1a;
            border: 1px solid #633;
            color: #f66;
            padding: 15px;
            border-radius: 6px;
            margin: 20px 0;
        }
        .no-camera {
            text-align: center;
            padding: 60px 20px;
            color: #666;
        }
        .no-camera h2 {
            color: #999;
            margin-bottom: 10px;
        }
        .alert-banner {
            padding: 16px 20px;
            border-radius: var(--radius-md);
            border-left: 4px solid;
            display: flex;
            align-items: center;
            gap: 12px;
            box-shadow: var(--shadow-md);
            backdrop-filter: blur(10px);
        }

        .alert-banner strong {
            font-size: 1.2em;
        }

        .alert-warning {
            background: rgba(245, 158, 11, 0.1);
            border-color: var(--accent-orange);
            color: var(--accent-orange);
        }

        .alert-critical {
            background: rgba(239, 68, 68, 0.1);
            border-color: var(--accent-red);
            color: var(--accent-red);
        }

        .alert-emergency {
            background: rgba(239, 68, 68, 0.15);
            border-color: var(--accent-red);
            color: var(--accent-red);
            animation: alertPulse 2s ease-in-out infinite;
        }

        @keyframes alertPulse {
            0%, 100% {
                opacity: 1;
                box-shadow: 0 4px 6px rgba(0, 0, 0, 0.4), 0 0 20px rgba(239, 68, 68, 0.3);
            }
            50% {
                opacity: 0.85;
                box-shadow: 0 4px 6px rgba(0, 0, 0, 0.4), 0 0 30px rgba(239, 68, 68, 0.5);
            }
        }
        .storage-card {
            background: var(--bg-tertiary);
            padding: 18px;
            border-radius: var(--radius-md);
            border: 1px solid var(--border-color);
            transition: all 0.3s ease;
        }

        .storage-card:hover {
            background: var(--bg-secondary);
            border-color: var(--accent-green);
            transform: translateY(-2px);
            box-shadow: var(--shadow-md);
        }

        .storage-camera-name {
            font-weight: 600;
            color: var(--accent-green);
            margin-bottom: 12px;
            font-size: 1.1em;
            letter-spacing: -0.2px;
        }

        .storage-detail {
            display: flex;
            justify-content: space-between;
            color: var(--text-secondary);
            font-size: 0.9em;
            margin: 8px 0;
            padding: 6px 0;
            border-bottom: 1px solid var(--border-color);
        }

        .storage-detail:last-child {
            border-bottom: none;
        }

        .storage-detail-value {
            color: var(--text-primary);
            font-weight: 600;
        }
        .progress-bar {
            height: 8px;
            background: #333;
            border-radius: 4px;
            overflow: hidden;
            margin: 10px 0;
        }
        .progress-fill {
            height: 100%;
            background: #4CAF50;
            transition: width 0.3s ease;
        }
        .progress-fill.warning { background: #fa0; }
        .progress-fill.critical { background: #f44; }
        .progress-fill.emergency { background: #f00; }

        /* Playback Interface Styles */
        .playback-select {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            padding: 10px 16px;
            border-radius: var(--radius-sm);
            font-size: 0.95em;
            min-width: 200px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .playback-select:hover {
            border-color: var(--accent-green);
            background: var(--bg-secondary);
        }

        .playback-select:focus {
            outline: none;
            border-color: var(--accent-green);
            box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
        }

        .recording-card {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-md);
            padding: 16px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .recording-card:hover {
            background: var(--bg-secondary);
            border-color: var(--accent-green);
            transform: translateY(-2px);
            box-shadow: var(--shadow-md);
        }

        .recording-card.active {
            border-color: var(--accent-green);
            background: var(--bg-secondary);
            box-shadow: 0 0 20px rgba(16, 185, 129, 0.2);
        }

        .recording-time {
            font-size: 1.1em;
            font-weight: 600;
            color: var(--accent-green);
            margin-bottom: 8px;
        }

        .recording-meta {
            display: flex;
            justify-content: space-between;
            font-size: 0.875em;
            color: var(--text-secondary);
            margin-top: 8px;
        }

        .timeline-bar {
            height: 60px;
            background: var(--bg-tertiary);
            border-radius: var(--radius-md);
            position: relative;
            overflow: visible;
            border: 1px solid var(--border-color);
            margin-bottom: 40px;
        }

        .timeline-segment {
            position: absolute;
            height: 100%;
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            cursor: pointer;
            transition: all 0.2s;
            border-right: 1px solid rgba(255,255,255,0.1);
        }

        .timeline-segment:hover {
            transform: scaleY(1.15);
            z-index: 10;
            box-shadow: 0 4px 16px rgba(16, 185, 129, 0.6);
            filter: brightness(1.1);
        }

        .timeline-segment:active {
            transform: scaleY(1.05);
        }

        .timeline-gap {
            position: absolute;
            height: 100%;
            background: repeating-linear-gradient(
                45deg,
                rgba(239, 68, 68, 0.2),
                rgba(239, 68, 68, 0.2) 10px,
                rgba(239, 68, 68, 0.4) 10px,
                rgba(239, 68, 68, 0.4) 20px
            );
            cursor: help;
            transition: all 0.2s;
            border: 1px solid var(--accent-red);
            border-radius: 2px;
        }

        .timeline-gap:hover {
            background: rgba(239, 68, 68, 0.6);
            transform: scaleY(1.1);
            z-index: 10;
            box-shadow: 0 4px 12px rgba(239, 68, 68, 0.5);
        }

        .timeline-future {
            position: absolute;
            height: 100%;
            background: var(--bg-tertiary);
            border: 1px dashed var(--border-color);
            border-radius: 2px;
            opacity: 0.5;
            cursor: default;
        }

        .timeline-current-time {
            position: absolute;
            top: -10px;
            bottom: -10px;
            width: 2px;
            background: #3b82f6;
            z-index: 100;
            box-shadow: 0 0 10px rgba(59, 130, 246, 0.6);
        }

        .timeline-current-time::before {
            content: 'NOW';
            position: absolute;
            top: -20px;
            left: -15px;
            background: #3b82f6;
            color: white;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.7em;
            font-weight: 700;
            white-space: nowrap;
        }

        .timeline-labels {
            position: absolute;
            top: 100%;
            left: 0;
            right: 0;
            display: flex;
            justify-content: space-between;
            margin-top: 8px;
            font-size: 0.75em;
            color: var(--text-secondary);
            font-weight: 500;
        }

        .timeline-hour-marker {
            position: absolute;
            top: 0;
            bottom: 0;
            width: 1px;
            background: var(--border-color);
            opacity: 0.3;
        }

        .timeline-hour-marker.major {
            opacity: 0.6;
            background: var(--text-secondary);
        }

        .timeline-legend {
            display: flex;
            gap: 24px;
            margin-bottom: 16px;
            padding: 12px 16px;
            background: var(--bg-tertiary);
            border-radius: var(--radius-sm);
            font-size: 0.875em;
        }

        .timeline-legend-item {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .timeline-legend-color {
            width: 24px;
            height: 16px;
            border-radius: 3px;
        }

        .timeline-legend-color.recorded {
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        }

        .timeline-legend-color.gap {
            background: repeating-linear-gradient(
                45deg,
                rgba(239, 68, 68, 0.3),
                rgba(239, 68, 68, 0.3) 4px,
                rgba(239, 68, 68, 0.6) 4px,
                rgba(239, 68, 68, 0.6) 8px
            );
            border: 1px solid var(--accent-red);
        }

        .timeline-legend-color.no-data {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
        }

        .timeline-legend-color.future {
            background: var(--bg-tertiary);
            border: 1px dashed var(--border-color);
            opacity: 0.6;
        }

        .gap-item {
            background: rgba(239, 68, 68, 0.1);
            border-left: 3px solid var(--accent-red);
            padding: 12px 16px;
            border-radius: var(--radius-sm);
            margin: 8px 0;
            cursor: pointer;
            transition: all 0.2s;
        }

        .gap-item:hover {
            background: rgba(239, 68, 68, 0.2);
            transform: translateX(4px);
        }

        .gap-time {
            font-weight: 600;
            color: var(--accent-red);
            font-size: 0.95em;
        }

        .gap-duration {
            margin-top: 4px;
            color: var(--text-secondary);
            font-size: 0.875em;
        }

        .timeline-tooltip {
            position: absolute;
            background: rgba(0, 0, 0, 0.95);
            color: white;
            padding: 8px 12px;
            border-radius: 6px;
            font-size: 0.875em;
            pointer-events: none;
            z-index: 1000;
            white-space: nowrap;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
        }

        .gap-duration {
            color: var(--text-secondary);
            font-size: 0.9em;
            margin-top: 4px;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="logo">
            <span class="status-dot"></span>
            CoreNVR
        </div>
        <div class="controls">
            <button class="btn active" id="btn-live" onclick="showLiveView()">Live</button>
            <button class="btn" id="btn-playback" onclick="showPlaybackView()">Playback</button>
            <button class="btn" id="btn-layout-single" onclick="setLayout(1)">Single</button>
            <button class="btn" id="btn-layout-2x2" onclick="setLayout(2)">2x2</button>
            <button class="btn" id="btn-layout-grid" onclick="setLayout(4)">Grid</button>
            <button class="btn" onclick="toggleStats()">Stats</button>
            <button class="btn" onclick="logout()" style="margin-left: auto;">Logout</button>
        </div>
    </div>

    <div class="container">
        <!-- Live View Container -->
        <div id="live-view-container">
            <div id="video-container" class="video-grid grid-1">
                <!-- Cameras will be loaded here -->
            </div>
        </div>

        <!-- Playback View Container -->
        <div id="playback-container" style="display: none;">
            <div style="display: flex; gap: 20px; margin-bottom: 20px; align-items: center;">
                <div>
                    <label style="color: var(--text-secondary); font-size: 0.9em; display: block; margin-bottom: 4px;">Camera</label>
                    <select id="playback-camera" class="playback-select">
                        <option value="">Loading...</option>
                    </select>
                </div>
                <div>
                    <label style="color: var(--text-secondary); font-size: 0.9em; display: block; margin-bottom: 4px;">Date</label>
                    <select id="playback-date" class="playback-select">
                        <option value="">Select date...</option>
                    </select>
                </div>
            </div>

            <!-- Timeline Visualization -->
            <div id="timeline-container" class="stats" style="display: none;">
                <h3>Recording Timeline for <span id="timeline-date"></span></h3>

                <!-- Timeline Stats -->
                <div id="timeline-info" style="display: flex; gap: 20px; margin: 16px 0; flex-wrap: wrap;">
                    <div class="stat-item" style="flex: 0 0 auto; min-width: 150px;">
                        <span class="stat-label">Coverage</span>
                        <span class="stat-value" id="timeline-coverage">--</span>
                    </div>
                    <div class="stat-item" style="flex: 0 0 auto; min-width: 150px;">
                        <span class="stat-label">Recorded Hours</span>
                        <span class="stat-value" id="timeline-hours">--</span>
                    </div>
                    <div class="stat-item" style="flex: 0 0 auto; min-width: 150px;">
                        <span class="stat-label">Segments</span>
                        <span class="stat-value" id="timeline-segments">--</span>
                    </div>
                    <div class="stat-item" style="flex: 0 0 auto; min-width: 150px;">
                        <span class="stat-label">Gaps</span>
                        <span class="stat-value" id="timeline-gaps">--</span>
                    </div>
                </div>

                <!-- Timeline Legend -->
                <div class="timeline-legend">
                    <div class="timeline-legend-item">
                        <div class="timeline-legend-color recorded"></div>
                        <span>Recorded</span>
                    </div>
                    <div class="timeline-legend-item">
                        <div class="timeline-legend-color gap"></div>
                        <span>Missing (Past)</span>
                    </div>
                    <div class="timeline-legend-item">
                        <div class="timeline-legend-color future"></div>
                        <span>Future</span>
                    </div>
                    <div class="timeline-legend-item">
                        <div style="width: 24px; height: 16px; background: #3b82f6; border-radius: 2px;"></div>
                        <span>Current Time</span>
                    </div>
                    <div style="margin-left: auto; color: var(--text-secondary); font-size: 0.875em;">
                        💡 Hover over timeline for details
                    </div>
                </div>

                <!-- Visual Timeline -->
                <div id="timeline-visual" style="position: relative;"></div>

                <!-- Gaps List (only shown if gaps exist) -->
                <div id="timeline-gaps-list" style="margin-top: 24px;"></div>
            </div>

            <!-- Video Player -->
            <div id="player-container" class="camera-container" style="display: none; margin-top: 20px;">
                <div class="camera-header">
                    <span class="camera-name" id="player-title">Select a recording</span>
                    <span class="camera-status" id="player-info"></span>
                </div>
                <div class="video-wrapper">
                    <video id="playback-video" controls style="position: absolute; top: 0; left: 0; width: 100%; height: 100%;">
                        Your browser does not support video playback.
                    </video>
                </div>
            </div>

            <!-- Recordings List -->
            <div id="recordings-list" class="stats" style="margin-top: 20px; display: none;">
                <h3>Available Recordings</h3>
                <div id="recordings-grid" style="margin-top: 16px;"></div>
            </div>
        </div>

        <div id="stats" class="stats" style="display: none;">
            <h3 style="color: #4CAF50; margin-bottom: 10px;">System Statistics</h3>

            <!-- Disk Usage Alert Banner -->
            <div id="disk-alert" style="display: none; margin-bottom: 15px;"></div>

            <div class="stats-grid">
                <div class="stat-item">
                    <span class="stat-label">Total Storage</span>
                    <span class="stat-value" id="total-storage">Loading...</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Used Storage</span>
                    <span class="stat-value" id="used-storage">Loading...</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Available Storage</span>
                    <span class="stat-value" id="available-storage">Loading...</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Disk Usage</span>
                    <span class="stat-value" id="disk-usage">Loading...</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Active Cameras</span>
                    <span class="stat-value" id="active-cameras">Loading...</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Retention Policy</span>
                    <span class="stat-value" id="retention">Loading...</span>
                </div>
            </div>

            <!-- Per-Camera Storage -->
            <h3 style="color: #4CAF50; margin: 20px 0 10px 0;">Camera Storage</h3>
            <div id="camera-storage" class="stats-grid">
                <div style="text-align: center; color: #666; padding: 20px;">Loading camera storage data...</div>
            </div>
        </div>
    </div>

    <!-- Include HLS.js for HLS playback support -->
    <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
    <script>
        let cameras = [];
        let currentLayout = 1;
        let statsVisible = false;
        let players = [];

        async function loadCameras() {
            try {
                const response = await fetch('/api/cameras');
                cameras = await response.json();
                renderCameras();
                updateStats();
            } catch (err) {
                console.error('Failed to load cameras:', err);
                document.getElementById('video-container').innerHTML =
                    '<div class="error-message">Failed to load cameras. Please check the connection.</div>';
            }
        }

        function renderCameras() {
            const container = document.getElementById('video-container');

            if (!cameras || cameras.length === 0) {
                container.innerHTML =
                    '<div class="no-camera">' +
                    '<h2>No Cameras Configured</h2>' +
                    '<p>Please configure cameras in /etc/corenvr/config.yaml</p>' +
                    '</div>';
                return;
            }

            // Clear existing players
            players.forEach(p => {
                if (p.hls) {
                    p.hls.destroy();
                }
            });
            players = [];

            const enabledCameras = cameras.filter(c => c.enabled);
            const camerasToShow = enabledCameras.slice(0, currentLayout === 1 ? 1 : currentLayout);

            container.innerHTML = camerasToShow.map((cam, index) =>
                '<div class="camera-container">' +
                    '<div class="camera-header">' +
                        '<span class="camera-name">' + cam.name + '</span>' +
                        '<span class="camera-status" id="status-' + index + '">' +
                            (cam.recording ? '🔴 Recording' : '⚫ Not Recording') +
                        '</span>' +
                    '</div>' +
                    '<div class="video-wrapper">' +
                        '<video id="video-' + index + '" controls muted autoplay></video>' +
                        '<div class="video-overlay" id="overlay-' + index + '">' +
                            '<div class="spinner"></div>' +
                            '<div>Loading stream...</div>' +
                        '</div>' +
                    '</div>' +
                '</div>'
            ).join('');

            // Initialize video players
            camerasToShow.forEach((cam, index) => {
                setupVideoPlayer(cam, index);
            });
        }

        function setupVideoPlayer(camera, index) {
            const video = document.getElementById('video-' + index);
            const overlay = document.getElementById('overlay-' + index);
            const streamUrl = '/stream/' + camera.name + '/playlist.m3u8';

            if (Hls.isSupported()) {
                const hls = new Hls({
                    enableWorker: true,
                    lowLatencyMode: true,
                    backBufferLength: 90
                });

                hls.loadSource(streamUrl);
                hls.attachMedia(video);

                hls.on(Hls.Events.MANIFEST_PARSED, function() {
                    overlay.style.display = 'none';
                    video.play().catch(e => {
                        console.log('Auto-play prevented:', e);
                    });
                });

                hls.on(Hls.Events.ERROR, function(event, data) {
                    console.error('HLS error:', data);
                    if (data.fatal) {
                        overlay.innerHTML =
                            '<div style="color: #f66;">⚠️ Stream unavailable</div>' +
                            '<div style="font-size: 0.9em; margin-top: 5px;">Check if camera is recording</div>';
                        overlay.style.display = 'block';
                    }
                });

                players.push({ video, hls, camera });

            } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
                // Native HLS support (Safari)
                video.src = streamUrl;
                video.addEventListener('loadedmetadata', function() {
                    overlay.style.display = 'none';
                    video.play().catch(e => {
                        console.log('Auto-play prevented:', e);
                    });
                });
            } else {
                overlay.innerHTML =
                    '<div style="color: #f66;">Browser not supported</div>' +
                    '<div style="font-size: 0.9em;">Please use a modern browser</div>';
            }
        }

        function setLayout(count) {
            currentLayout = count;
            const container = document.getElementById('video-container');

            // Remove all grid classes
            container.classList.remove('grid-1', 'grid-2', 'grid-4');

            // Add appropriate grid class
            if (count === 1) {
                container.classList.add('grid-1');
            } else if (count === 2) {
                container.classList.add('grid-2');
            } else {
                container.classList.add('grid-4');
            }

            // Update button states
            document.querySelectorAll('.controls .btn').forEach(btn => {
                btn.classList.remove('active');
            });
            event.target.classList.add('active');

            renderCameras();
        }

        function toggleStats() {
            statsVisible = !statsVisible;
            document.getElementById('stats').style.display = statsVisible ? 'block' : 'none';
            event.target.classList.toggle('active');
        }

        async function updateStats() {
            try {
                // Fetch storage statistics
                const storageResponse = await fetch('/api/storage');
                const storage = await storageResponse.json();

                // Update disk usage stats
                document.getElementById('total-storage').textContent = storage.total_gb + ' GB';
                document.getElementById('used-storage').textContent = storage.used_gb + ' GB';
                document.getElementById('available-storage').textContent = storage.available_gb + ' GB';
                document.getElementById('retention').textContent = storage.retention_days + ' days';

                const diskValue = parseFloat(storage.percent_used);
                const diskElement = document.getElementById('disk-usage');
                diskElement.textContent = storage.percent_used + '%';

                // Color code disk usage
                if (diskValue >= 95) {
                    diskElement.style.color = '#f00';
                } else if (diskValue >= 90) {
                    diskElement.style.color = '#f66';
                } else if (diskValue >= 80) {
                    diskElement.style.color = '#fa0';
                } else {
                    diskElement.style.color = '#4CAF50';
                }

                // Show alert banner if needed
                const alertDiv = document.getElementById('disk-alert');
                if (storage.alert_level !== 'normal') {
                    let alertClass = 'alert-warning';
                    let emoji = '⚠️';
                    let message = 'Disk usage is high';

                    if (storage.alert_level === 'critical') {
                        alertClass = 'alert-critical';
                        emoji = '🔴';
                        message = 'Disk usage is critical! Cleanup will run automatically.';
                    } else if (storage.alert_level === 'emergency') {
                        alertClass = 'alert-emergency';
                        emoji = '🚨';
                        message = 'EMERGENCY: Disk almost full! Emergency cleanup in progress.';
                    }

                    alertDiv.className = 'alert-banner ' + alertClass;
                    alertDiv.innerHTML = '<strong>' + emoji + '</strong> <span>' + message +
                        ' (' + storage.percent_used + '% used, ' + storage.available_gb + ' GB available)</span>';
                    alertDiv.style.display = 'flex';
                } else {
                    alertDiv.style.display = 'none';
                }

                // Update active cameras count
                const activeCameras = cameras.filter(c => c.enabled && c.recording).length;
                const totalCameras = cameras.filter(c => c.enabled).length;
                document.getElementById('active-cameras').textContent = activeCameras + ' / ' + totalCameras;

                // Update per-camera storage
                updateCameraStorage(storage.cameras, diskValue);

            } catch (err) {
                console.error('Failed to update stats:', err);
            }
        }

        function updateCameraStorage(cameraStorage, diskUsage) {
            const container = document.getElementById('camera-storage');

            if (!cameraStorage || cameraStorage.length === 0) {
                container.innerHTML = '<div style="text-align: center; color: #666; padding: 20px;">No camera storage data available</div>';
                return;
            }

            let progressClass = '';
            if (diskUsage >= 95) progressClass = 'emergency';
            else if (diskUsage >= 90) progressClass = 'critical';
            else if (diskUsage >= 80) progressClass = 'warning';

            container.innerHTML = cameraStorage.map(cam =>
                '<div class="storage-card">' +
                    '<div class="storage-camera-name">' + cam.name + '</div>' +
                    '<div class="storage-detail">' +
                        '<span>Storage Used:</span>' +
                        '<span class="storage-detail-value">' + cam.size_gb + ' GB</span>' +
                    '</div>' +
                    '<div class="storage-detail">' +
                        '<span>Days Stored:</span>' +
                        '<span class="storage-detail-value">' + cam.days_stored + ' days</span>' +
                    '</div>' +
                '</div>'
            ).join('');
        }

        // Playback functionality
        let currentView = 'live';
        let selectedCamera = '';
        let selectedDate = '';
        let timelineData = null;
        let recordingsList = [];

        function showLiveView() {
            currentView = 'live';
            document.getElementById('live-view-container').style.display = 'block';
            document.getElementById('playback-container').style.display = 'none';
            document.getElementById('btn-live').classList.add('active');
            document.getElementById('btn-playback').classList.remove('active');
            document.getElementById('btn-layout-single').style.display = 'inline-block';
            document.getElementById('btn-layout-2x2').style.display = 'inline-block';
            document.getElementById('btn-layout-grid').style.display = 'inline-block';
        }

        function showPlaybackView() {
            currentView = 'playback';
            document.getElementById('live-view-container').style.display = 'none';
            document.getElementById('playback-container').style.display = 'block';
            document.getElementById('btn-live').classList.remove('active');
            document.getElementById('btn-playback').classList.add('active');
            document.getElementById('btn-layout-single').style.display = 'none';
            document.getElementById('btn-layout-2x2').style.display = 'none';
            document.getElementById('btn-layout-grid').style.display = 'none';
            loadPlaybackCameras();
        }

        async function loadPlaybackCameras() {
            const select = document.getElementById('playback-camera');
            select.innerHTML = '<option value="">Loading...</option>';

            try {
                const response = await fetch('/api/cameras');
                const cams = await response.json();

                select.innerHTML = '<option value="">Select camera...</option>';
                cams.forEach(cam => {
                    const option = document.createElement('option');
                    option.value = cam.name;
                    option.textContent = cam.name;
                    select.appendChild(option);
                });

                select.onchange = () => loadPlaybackDates(select.value);
            } catch (err) {
                console.error('Failed to load cameras:', err);
                select.innerHTML = '<option value="">Error loading cameras</option>';
            }
        }

        async function loadPlaybackDates(camera) {
            if (!camera) return;

            selectedCamera = camera;
            const dateSelect = document.getElementById('playback-date');
            dateSelect.innerHTML = '<option value="">Loading...</option>';

            try {
                const response = await fetch('/api/recordings/dates?camera=' + camera);
                const data = await response.json();

                dateSelect.innerHTML = '<option value="">Select date...</option>';
                data.dates.reverse().forEach(date => {
                    const option = document.createElement('option');
                    option.value = date;
                    option.textContent = date;
                    dateSelect.appendChild(option);
                });

                dateSelect.onchange = () => loadDateRecordings(camera, dateSelect.value);
            } catch (err) {
                console.error('Failed to load dates:', err);
                dateSelect.innerHTML = '<option value="">Error loading dates</option>';
            }
        }

        async function loadDateRecordings(camera, date) {
            if (!date) return;

            selectedDate = date;

            // Load timeline
            loadTimeline(camera, date);

            // Load recordings list
            try {
                const response = await fetch('/api/recordings/list?camera=' + camera + '&date=' + date);
                const data = await response.json();
                recordingsList = data.recordings;

                displayRecordingsList(data.recordings);
            } catch (err) {
                console.error('Failed to load recordings:', err);
            }
        }

        async function loadTimeline(camera, date) {
            try {
                const response = await fetch('/api/recordings/timeline?camera=' + camera + '&date=' + date);
                timelineData = await response.json();

                document.getElementById('timeline-container').style.display = 'block';
                document.getElementById('timeline-coverage').textContent = timelineData.coverage_percent + '%';
                document.getElementById('timeline-hours').textContent = timelineData.recorded_hours + ' hrs';
                document.getElementById('timeline-segments').textContent = timelineData.total_segments;
                document.getElementById('timeline-gaps').textContent = timelineData.total_gaps;

                // Set color for coverage
                const coverageEl = document.getElementById('timeline-coverage');
                const coverage = parseFloat(timelineData.coverage_percent);
                if (coverage >= 95) {
                    coverageEl.style.color = 'var(--accent-green)';
                } else if (coverage >= 80) {
                    coverageEl.style.color = 'var(--accent-orange)';
                } else {
                    coverageEl.style.color = 'var(--accent-red)';
                }

                // Render visual timeline
                renderTimeline(timelineData);

                // Show gaps if any
                if (timelineData.gaps && timelineData.gaps.length > 0) {
                    renderGaps(timelineData.gaps);
                }
            } catch (err) {
                console.error('Failed to load timeline:', err);
            }
        }

        function renderTimeline(data) {
            const visual = document.getElementById('timeline-visual');
            const segments = data.segments || [];
            const gaps = data.gaps || [];

            // Get current time for time-aware rendering
            const now = new Date();
            const isToday = selectedDate === now.toISOString().split('T')[0];
            const currentMinutes = isToday ? (now.getHours() * 60 + now.getMinutes()) : 1440;

            let html = '<div class="timeline-bar">';

            // Add hour markers (every 3 hours, with major markers at 6-hour intervals)
            for (let hour = 0; hour <= 24; hour++) {
                const left = (hour / 24) * 100;
                const isMajor = hour % 6 === 0;
                html += '<div class="timeline-hour-marker' + (isMajor ? ' major' : '') + '" style="left: ' + left + '%;"></div>';
            }

            // Render recording segments (green bars) - clickable to play
            segments.forEach(seg => {
                const start = timeToMinutes(seg.start_time);
                const end = timeToMinutes(seg.end_time);
                const left = (start / 1440) * 100;
                const width = ((end - start) / 1440) * 100;

                const tooltip = seg.start_time + ' - ' + seg.end_time + '\\n' + seg.size_mb + ' MB';
                html += '<div class="timeline-segment" style="left: ' + left + '%; width: ' + width + '%;"' +
                    ' title="' + tooltip + ' - Click to play"' +
                    ' onclick="playFromTimeline(\'' + seg.start_time + '\')"' +
                    ' onmouseenter="showTooltip(event, \'' + seg.start_time + '\', \'' + seg.end_time + '\', \'' + seg.size_mb + ' MB - Click to play\')"' +
                    ' onmouseleave="hideTooltip()"></div>';
            });

            // Render gaps - only for time in the PAST (before current time)
            gaps.forEach(gap => {
                const start = timeToMinutes(gap.start_time);
                const end = timeToMinutes(gap.end_time);

                // Only show as gap (red) if it's in the past
                if (start < currentMinutes) {
                    const actualEnd = Math.min(end, currentMinutes); // Cap at current time
                    const left = (start / 1440) * 100;
                    const width = ((actualEnd - start) / 1440) * 100;

                    const tooltip = 'GAP: ' + gap.start_time + ' - ' + gap.end_time + '\\n' + gap.duration_mins + ' minutes missing';
                    html += '<div class="timeline-gap" style="left: ' + left + '%; width: ' + width + '%;"' +
                        ' title="' + tooltip + '"' +
                        ' onmouseenter="showTooltip(event, \'' + gap.start_time + '\', \'' + gap.end_time + '\', \'' + gap.duration_mins + ' min gap\', true)"' +
                        ' onmouseleave="hideTooltip()"></div>';
                }
            });

            // Render future time (gray dashed area) - only for today
            if (isToday && currentMinutes < 1440) {
                const futureLeft = (currentMinutes / 1440) * 100;
                const futureWidth = 100 - futureLeft;
                const futureStartTime = formatMinutesToTime(currentMinutes);

                html += '<div class="timeline-future" style="left: ' + futureLeft + '%; width: ' + futureWidth + '%;"' +
                    ' title="Future time (not yet recorded)"' +
                    ' onmouseenter="showTooltip(event, \'' + futureStartTime + '\', \'23:59:59\', \'Future - not yet recorded\')"' +
                    ' onmouseleave="hideTooltip()"></div>';
            }

            // Add current time marker (blue line) - only for today
            if (isToday) {
                const currentLeft = (currentMinutes / 1440) * 100;
                html += '<div class="timeline-current-time" style="left: ' + currentLeft + '%;"></div>';
            }

            html += '</div>';

            // Enhanced time labels (every 3 hours)
            html += '<div class="timeline-labels">';
            for (let hour = 0; hour <= 24; hour += 3) {
                const timeStr = (hour < 10 ? '0' : '') + hour + ':00';
                html += '<span>' + timeStr + '</span>';
            }
            html += '</div>';

            visual.innerHTML = html;

            // Update the date display
            document.getElementById('timeline-date').textContent = selectedDate + (isToday ? ' (Today)' : '');
        }

        // Helper function to format minutes back to time string
        function formatMinutesToTime(minutes) {
            const hours = Math.floor(minutes / 60);
            const mins = Math.floor(minutes % 60);
            const secs = 0;
            return (hours < 10 ? '0' : '') + hours + ':' + (mins < 10 ? '0' : '') + mins + ':' + (secs < 10 ? '0' : '') + secs;
        }

        // Tooltip functions for better interactivity
        let tooltipEl = null;

        function showTooltip(event, startTime, endTime, extra, isGap) {
            hideTooltip();

            tooltipEl = document.createElement('div');
            tooltipEl.className = 'timeline-tooltip';
            tooltipEl.style.position = 'fixed';
            tooltipEl.style.left = event.clientX + 10 + 'px';
            tooltipEl.style.top = event.clientY - 10 + 'px';

            const icon = isGap ? '⚠️ ' : '✅ ';
            const label = isGap ? 'Recording Gap' : 'Recording';

            tooltipEl.innerHTML = '<strong>' + icon + label + '</strong><br>' +
                startTime + ' → ' + endTime + '<br>' +
                '<small>' + extra + '</small>';

            document.body.appendChild(tooltipEl);
        }

        function hideTooltip() {
            if (tooltipEl) {
                tooltipEl.remove();
                tooltipEl = null;
            }
        }

        function renderGaps(gaps) {
            const container = document.getElementById('timeline-gaps-list');

            if (!gaps || gaps.length === 0) {
                container.innerHTML = '';
                return;
            }

            // Get current time to filter out future "gaps"
            const now = new Date();
            const isToday = selectedDate === now.toISOString().split('T')[0];
            const currentMinutes = isToday ? (now.getHours() * 60 + now.getMinutes()) : 1440;

            // Filter gaps to only show those in the PAST
            const actualGaps = gaps.filter(gap => {
                const gapStart = timeToMinutes(gap.start_time);
                return gapStart < currentMinutes;
            });

            if (actualGaps.length === 0) {
                container.innerHTML = '';
                return;
            }

            let html = '<div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;">';
            html += '<h4 style="color: var(--accent-red); margin: 0;">⚠️ ' + actualGaps.length + ' Recording Gap' + (actualGaps.length > 1 ? 's' : '') + ' Detected</h4>';
            html += '<button class="btn" onclick="toggleGapsList()" id="toggle-gaps-btn" style="font-size: 0.875em; padding: 6px 12px;">Show Details</button>';
            html += '</div>';

            html += '<div id="gaps-details" style="display: none;">';
            actualGaps.forEach((gap, idx) => {
                const gapStart = timeToMinutes(gap.start_time);
                const gapEnd = timeToMinutes(gap.end_time);

                // Calculate actual gap duration (capped at current time for today)
                const actualEnd = Math.min(gapEnd, currentMinutes);
                const actualDuration = actualEnd - gapStart;

                const durationHours = (actualDuration / 60).toFixed(1);
                const durationText = actualDuration >= 60
                    ? durationHours + ' hours (' + actualDuration + ' min)'
                    : actualDuration + ' minutes';

                html += '<div class="gap-item">' +
                    '<div style="display: flex; justify-content: space-between; align-items: center;">' +
                        '<div class="gap-time">Gap #' + (idx + 1) + ': ' + gap.start_time + ' → ' + (gapEnd > currentMinutes && isToday ? 'NOW' : gap.end_time) + '</div>' +
                        '<div style="color: var(--accent-red); font-weight: 600; font-size: 0.9em;">' + durationText + '</div>' +
                    '</div>' +
                '</div>';
            });
            html += '</div>';

            container.innerHTML = html;
        }

        function toggleGapsList() {
            const details = document.getElementById('gaps-details');
            const btn = document.getElementById('toggle-gaps-btn');
            if (details.style.display === 'none') {
                details.style.display = 'block';
                btn.textContent = 'Hide Details';
            } else {
                details.style.display = 'none';
                btn.textContent = 'Show Details';
            }
        }

        // Store recordings data for timeline clicks
        let currentRecordings = [];

        // Play recording from timeline click
        function playFromTimeline(clickedTime) {
            console.log('Timeline clicked at:', clickedTime);

            // Find the recording that contains this time
            // Recordings are 30-minute segments, so calculate end_time
            const recording = currentRecordings.find(rec => {
                // Extract just the time portion (HH:MM:SS) from start_time
                const startTime = rec.start_time.includes(' ') ? rec.start_time.split(' ')[1] : rec.start_time;

                // Calculate end time (start + 30 minutes)
                const startMinutes = timeToMinutes(startTime);
                const endMinutes = startMinutes + 30; // 30-minute segments
                const clickMinutes = timeToMinutes(clickedTime);

                console.log('Checking recording:', startTime, 'startMin:', startMinutes, 'endMin:', endMinutes, 'clickMin:', clickMinutes);

                return clickMinutes >= startMinutes && clickMinutes < endMinutes;
            });

            if (recording) {
                console.log('Found recording:', recording);

                // Find the index in the recordings array
                const index = currentRecordings.indexOf(recording);

                // Play the recording using the existing playRecording function
                playRecording(recording.playlist_url, recording.start_time, recording.size_mb, index);

                // Scroll to the player
                document.getElementById('player-container').scrollIntoView({ behavior: 'smooth', block: 'start' });
            } else {
                console.error('No recording found for time:', clickedTime);
                console.log('Available recordings:', currentRecordings);
                alert('Recording not found for the selected time. This may be during a gap.');
            }
        }

        function displayRecordingsList(recordings) {
            const container = document.getElementById('recordings-grid');
            const listDiv = document.getElementById('recordings-list');
            listDiv.style.display = 'block';

            // Store recordings for timeline clicks
            currentRecordings = recordings || [];

            if (!recordings || recordings.length === 0) {
                container.innerHTML = '<div style="text-align: center; padding: 20px; color: var(--text-secondary);">No recordings found for this date.</div>';
                return;
            }

            container.style.display = 'grid';
            container.style.gridTemplateColumns = 'repeat(auto-fill, minmax(250px, 1fr))';
            container.style.gap = '16px';

            container.innerHTML = recordings.map((rec, idx) => {
                const time = rec.start_time.split(' ')[1];
                return '<div class="recording-card" onclick="playRecording(\'' + rec.playlist_url + '\', \'' + rec.start_time + '\', \'' + rec.size_mb + '\', ' + idx + ')">' +
                    '<div class="recording-time">' + time + '</div>' +
                    '<div class="recording-meta">' +
                        '<span>' + rec.size_mb + ' MB</span>' +
                        '<span>30 min</span>' +
                    '</div>' +
                    '<div style="margin-top: 12px;">' +
                        '<a href="' + rec.url + '" download style="color: var(--accent-green); text-decoration: none; font-size: 0.875em;"' +
                           ' onclick="event.stopPropagation();">⬇ Download</a>' +
                    '</div>' +
                '</div>';
            }).join('');
        }

        let playbackHls = null;

        function playRecording(playlistUrl, time, size, index) {
            console.log('Playing recording via HLS:', playlistUrl);

            const player = document.getElementById('playback-video');
            const container = document.getElementById('player-container');
            const title = document.getElementById('player-title');
            const info = document.getElementById('player-info');

            if (!player || !container) {
                console.error('Player elements not found');
                return;
            }

            container.style.display = 'block';
            title.textContent = selectedCamera + ' - ' + selectedDate;
            info.textContent = time + ' • ' + size + ' MB';

            // Clean up previous HLS instance
            if (playbackHls) {
                playbackHls.destroy();
                playbackHls = null;
            }

            // Stop any current playback
            player.pause();
            player.currentTime = 0;
            player.src = '';

            if (Hls.isSupported()) {
                // Use HLS.js to play the playlist
                playbackHls = new Hls({
                    debug: true,
                    enableWorker: true,
                    lowLatencyMode: false
                });

                playbackHls.loadSource(playlistUrl);
                playbackHls.attachMedia(player);

                playbackHls.on(Hls.Events.MANIFEST_PARSED, function() {
                    console.log('HLS manifest parsed, starting playback');
                    player.play().catch((error) => {
                        console.warn('Autoplay blocked:', error);
                        console.log('User needs to click play button');
                    });
                });

                playbackHls.on(Hls.Events.ERROR, function(event, data) {
                    console.error('HLS error:', data.type, data.details);
                    if (data.fatal) {
                        console.error('Fatal HLS error:', data);
                        alert('Failed to load video: ' + data.details);
                    }
                });
            } else if (player.canPlayType('application/vnd.apple.mpegurl')) {
                // Native HLS support (Safari)
                player.src = playlistUrl;
                player.load();
                player.play().catch((error) => {
                    console.warn('Autoplay blocked:', error);
                });
            } else {
                alert('Your browser does not support HLS playback. Please try a modern browser.');
            }

            // Highlight selected recording
            document.querySelectorAll('.recording-card').forEach((card, idx) => {
                if (idx === index) {
                    card.classList.add('active');
                } else {
                    card.classList.remove('active');
                }
            });

            // Scroll to player
            container.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }

        function timeToMinutes(timeStr) {
            const parts = timeStr.split(':');
            return parseInt(parts[0]) * 60 + parseInt(parts[1]);
        }

        // Initialize
        loadCameras();

        // Setup video player error handling
        const playbackVideo = document.getElementById('playback-video');
        if (playbackVideo) {
            playbackVideo.addEventListener('error', (e) => {
                console.error('Video error:', e);
                console.error('Video error code:', playbackVideo.error ? playbackVideo.error.code : 'unknown');
                console.error('Video error message:', playbackVideo.error ? playbackVideo.error.message : 'unknown');
            });

            playbackVideo.addEventListener('loadedmetadata', () => {
                console.log('Video metadata loaded');
            });

            playbackVideo.addEventListener('canplay', () => {
                console.log('Video can play');
            });
        }

        // Refresh stats every 30 seconds
        setInterval(updateStats, 30000);

        // Refresh camera status every 60 seconds
        setInterval(loadCameras, 60000);

        // Logout function
        function logout() {
            if (confirm('Are you sure you want to logout?')) {
                window.location.href = '/logout';
            }
        }
    </script>
</body>
</html>`