package webui

const loginTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>CoreNVR - Login</title>
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
            --accent-red: #ef4444;
            --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.4);
            --shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.5);
            --radius-md: 8px;
            --radius-lg: 12px;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #0f0f0f 0%, #1a1a1a 100%);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        .login-container {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-lg);
            padding: 40px;
            width: 100%;
            max-width: 400px;
            box-shadow: var(--shadow-lg);
        }

        .logo {
            text-align: center;
            margin-bottom: 32px;
        }

        .logo h1 {
            font-size: 2em;
            color: var(--accent-green);
            font-weight: 700;
            letter-spacing: -0.5px;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 12px;
        }

        .logo .status-dot {
            width: 12px;
            height: 12px;
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

        .logo p {
            color: var(--text-secondary);
            font-size: 0.9em;
            margin-top: 8px;
        }

        .form-group {
            margin-bottom: 24px;
        }

        label {
            display: block;
            color: var(--text-secondary);
            font-size: 0.9em;
            margin-bottom: 8px;
            font-weight: 500;
        }

        input[type="text"],
        input[type="password"] {
            width: 100%;
            padding: 12px 16px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: var(--radius-md);
            color: var(--text-primary);
            font-size: 1em;
            transition: all 0.2s ease;
        }

        input[type="text"]:focus,
        input[type="password"]:focus {
            outline: none;
            border-color: var(--accent-green);
            box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
        }

        .checkbox-group {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 24px;
        }

        input[type="checkbox"] {
            width: 18px;
            height: 18px;
            cursor: pointer;
        }

        .checkbox-group label {
            margin: 0;
            cursor: pointer;
            user-select: none;
        }

        .btn-login {
            width: 100%;
            padding: 12px;
            background: var(--accent-green);
            border: none;
            border-radius: var(--radius-md);
            color: white;
            font-size: 1em;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .btn-login:hover {
            background: var(--accent-green-dark);
            transform: translateY(-1px);
            box-shadow: var(--shadow-md);
        }

        .btn-login:active {
            transform: translateY(0);
        }

        .error-message {
            background: rgba(239, 68, 68, 0.1);
            border: 1px solid var(--accent-red);
            border-radius: var(--radius-md);
            padding: 12px 16px;
            color: var(--accent-red);
            margin-bottom: 20px;
            font-size: 0.9em;
            display: none;
        }

        .error-message.show {
            display: block;
        }

        .footer {
            text-align: center;
            margin-top: 24px;
            color: var(--text-secondary);
            font-size: 0.85em;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="logo">
            <h1>
                <span class="status-dot"></span>
                CoreNVR
            </h1>
            <p>Network Video Recorder</p>
        </div>

        <div id="error-message" class="error-message"></div>

        <form id="login-form" method="POST" action="/login">
            <div class="form-group">
                <label for="username">Username</label>
                <input
                    type="text"
                    id="username"
                    name="username"
                    required
                    autofocus
                    autocomplete="username"
                >
            </div>

            <div class="form-group">
                <label for="password">Password</label>
                <input
                    type="password"
                    id="password"
                    name="password"
                    required
                    autocomplete="current-password"
                >
            </div>

            <div class="checkbox-group">
                <input
                    type="checkbox"
                    id="remember"
                    name="remember"
                >
                <label for="remember">Remember me (30 days)</label>
            </div>

            <button type="submit" class="btn-login">Login</button>
        </form>

        <div class="footer">
            Lightweight NVR for Raspberry Pi
        </div>
    </div>

    <script>
        // Handle form submission
        const form = document.getElementById('login-form');
        const errorDiv = document.getElementById('error-message');

        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            errorDiv.classList.remove('show');

            const formData = new FormData(form);

            // Convert FormData to URLSearchParams for proper form encoding
            const params = new URLSearchParams();
            for (const [key, value] of formData) {
                params.append(key, value);
            }

            try {
                const response = await fetch('/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: params.toString()
                });

                if (response.ok) {
                    // Login successful, redirect
                    window.location.href = '/';
                } else {
                    // Login failed
                    try {
                        const data = await response.json();
                        errorDiv.textContent = data.error || 'Invalid username or password';
                    } catch {
                        errorDiv.textContent = 'Invalid username or password';
                    }
                    errorDiv.classList.add('show');
                }
            } catch (error) {
                console.error('Login error:', error);
                errorDiv.textContent = 'An error occurred. Please try again.';
                errorDiv.classList.add('show');
            }
        });

        // Check for error message in URL
        const urlParams = new URLSearchParams(window.location.search);
        const error = urlParams.get('error');
        if (error) {
            errorDiv.textContent = decodeURIComponent(error);
            errorDiv.classList.add('show');
        }
    </script>
</body>
</html>
`
