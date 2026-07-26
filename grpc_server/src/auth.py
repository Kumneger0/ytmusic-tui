import json
import os
import re
import sys
import tempfile
from pathlib import Path

from ytmusicapi import YTMusic, setup


def get_config_dir() -> Path:
    """
    Returns the platform-specific config directory for the application.
    Linux: $XDG_CONFIG_HOME/ytmusic-tui or ~/.config/ytmusic-tui
    macOS: ~/Library/Application Support/ytmusic-tui
    Windows: %APPDATA%/ytmusic-tui
    """
    if sys.platform == "win32":
        appdata = os.getenv("APPDATA")
        if appdata:
            return Path(appdata) / "ytmusic-tui"
        return Path.home() / "AppData" / "Roaming" / "ytmusic-tui"
    elif sys.platform == "darwin":
        return Path.home() / "Library" / "Application Support" / "ytmusic-tui"
    else:
        xdg_config = os.getenv("XDG_CONFIG_HOME")
        if xdg_config:
            return Path(xdg_config) / "ytmusic-tui"
        return Path.home() / ".config" / "ytmusic-tui"


def get_browser_json_path() -> Path:
    """
    Returns the path to browser.json in the application's config directory.
    If browser.json doesn't exist in ytmusic-tui dir, checks legacy ytmusic-tui path for fallback.
    """
    config_dir = get_config_dir()
    config_dir.mkdir(parents=True, exist_ok=True)
    primary_path = config_dir / "browser.json"
    if not primary_path.exists():
        legacy_path = Path.home() / ".config" / "ytmusic-tui" / "browser.json"
        if legacy_path.exists():
            return legacy_path
    return primary_path



def test_authentication(filepath: Path) -> tuple[bool, str]:
    """
    Tests authentication by making a simple YouTube Music API request using saved browser.json.
    """
    try:
        yt = YTMusic(auth=str(filepath))
        _ = yt.get_library_subscriptions(limit=1)
        return True, ""
    except Exception as e:
        return False, str(e)


LOGIN_HTML = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>YouTube Music Authentication Setup</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #0b0b0f;
            --card-bg: rgba(22, 22, 30, 0.85);
            --card-border: rgba(255, 255, 255, 0.1);
            --primary: #ff2d55;
            --primary-hover: #e02447;
            --text-main: #f3f4f6;
            --text-muted: #9ca3af;
            --input-bg: #121218;
            --input-border: #2d2d3a;
            --success-bg: rgba(16, 185, 129, 0.15);
            --success-border: #10b981;
            --error-bg: rgba(239, 68, 68, 0.15);
            --error-border: #ef4444;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background-color: var(--bg-color);
            background-image: 
                radial-gradient(circle at 15% 20%, rgba(255, 45, 85, 0.18) 0%, transparent 40%),
                radial-gradient(circle at 85% 80%, rgba(139, 92, 246, 0.15) 0%, transparent 40%);
            color: var(--text-main);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            width: 100%;
            max-width: 680px;
            background: var(--card-bg);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            border: 1px solid var(--card-border);
            border-radius: 20px;
            padding: 36px;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.6);
        }
        .header { display: flex; align-items: center; gap: 14px; margin-bottom: 24px; }
        .logo-icon {
            width: 44px; height: 44px;
            background: linear-gradient(135deg, #ff2d55, #ff6b8b);
            border-radius: 12px;
            display: flex; align-items: center; justify-content: center;
            box-shadow: 0 4px 15px rgba(255, 45, 85, 0.4);
        }
        .logo-icon svg { width: 24px; height: 24px; fill: #ffffff; }
        h1 { font-size: 22px; font-weight: 700; color: #ffffff; letter-spacing: -0.5px; }
        .subtitle { font-size: 14px; color: var(--text-muted); margin-top: 2px; }
        .steps {
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid rgba(255, 255, 255, 0.06);
            border-radius: 12px; padding: 18px 20px; margin-bottom: 24px;
        }
        .step-item { display: flex; align-items: flex-start; gap: 12px; font-size: 14px; color: #d1d5db; margin-bottom: 10px; }
        .step-item:last-child { margin-bottom: 0; }
        .step-num {
            background: var(--primary); color: white; font-weight: 600; font-size: 12px;
            width: 20px; height: 20px; border-radius: 50%; display: flex; align-items: center;
            justify-content: center; flex-shrink: 0; margin-top: 1px;
        }
        .step-item a { color: #ff6b8b; text-decoration: none; }
        .step-item a:hover { text-decoration: underline; }
        .form-group { margin-bottom: 20px; }
        label { display: block; font-size: 14px; font-weight: 600; color: #e5e7eb; margin-bottom: 8px; }
        textarea {
            width: 100%; height: 180px; background-color: var(--input-bg);
            border: 1px solid var(--input-border); border-radius: 12px; padding: 14px;
            color: #f3f4f6; font-family: monospace; font-size: 13px; line-height: 1.5;
            resize: vertical; outline: none; transition: border-color 0.2s, box-shadow 0.2s;
        }
        textarea:focus { border-color: var(--primary); box-shadow: 0 0 0 3px rgba(255, 45, 85, 0.2); }
        .btn {
            width: 100%; padding: 14px; background: var(--primary); color: white;
            border: none; border-radius: 12px; font-size: 15px; font-weight: 600;
            cursor: pointer; transition: background 0.2s, transform 0.1s;
            display: flex; align-items: center; justify-content: center; gap: 8px;
        }
        .btn:hover { background: var(--primary-hover); }
        .btn:active { transform: scale(0.99); }
        .btn:disabled { opacity: 0.6; cursor: not-allowed; }
        .status-msg { margin-top: 20px; padding: 14px 16px; border-radius: 12px; font-size: 14px; line-height: 1.5; display: none; }
        .status-msg.success { display: block; background: var(--success-bg); border: 1px solid var(--success-border); color: #34d399; }
        .status-msg.error { display: block; background: var(--error-bg); border: 1px solid var(--error-border); color: #f87171; }
        .spinner {
            width: 18px; height: 18px; border: 2px solid rgba(255,255,255,0.3);
            border-top-color: white; border-radius: 50%; animation: spin 0.8s linear infinite; display: none;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo-icon">
                <svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 14.5v-9l6 4.5-6 4.5z"/></svg>
            </div>
            <div>
                <h1>YouTube Music Login</h1>
                <div class="subtitle">CLI Application Authentication Setup</div>
            </div>
        </div>

        <div class="steps">
            <div class="step-item">
                <span class="step-num">1</span>
                <span>Open <a href="https://music.youtube.com" target="_blank">YouTube Music</a> in your browser and ensure you are logged in.</span>
            </div>
            <div class="step-item">
                <span class="step-num">2</span>
                <span>Open DevTools (F12 or Right-Click -> Inspect) and go to the <strong>Network</strong> tab.</span>
            </div>
            <div class="step-item">
                <span class="step-num">3</span>
                <span>Click any API request (such as <code>/browse</code> or <code>/search</code>) and copy the <strong>Request Headers</strong> (or right-click -> Copy request headers / Copy as cURL).</span>
            </div>
        </div>

        <div class="form-group">
            <label for="headersInput">Paste Request Headers</label>
            <textarea id="headersInput" placeholder="Paste your raw headers or cURL command here..."></textarea>
        </div>

        <button id="submitBtn" class="btn" onclick="submitHeaders()">
            <div id="btnSpinner" class="spinner"></div>
            <span id="btnText">Authenticate & Save Credentials</span>
        </button>

        <div id="statusBox" class="status-msg"></div>
    </div>

    <script>
        async function submitHeaders() {
            const textarea = document.getElementById('headersInput');
            const submitBtn = document.getElementById('submitBtn');
            const btnText = document.getElementById('btnText');
            const btnSpinner = document.getElementById('btnSpinner');
            const statusBox = document.getElementById('statusBox');

            const headers = textarea.value.trim();
            if (!headers) {
                statusBox.className = 'status-msg error';
                statusBox.style.display = 'block';
                statusBox.textContent = 'Please paste your YouTube Music headers into the box above before submitting.';
                return;
            }

            submitBtn.disabled = true;
            btnSpinner.style.display = 'inline-block';
            btnText.textContent = 'Authenticating...';
            statusBox.style.display = 'none';

            try {
                const res = await fetch('/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ headers: headers })
                });

                const data = await res.json();

                if (res.ok && data.success) {
                    statusBox.className = 'status-msg success';
                    statusBox.style.display = 'block';
                    statusBox.innerHTML = '<strong>🎉 Authentication Successful!</strong><br><br>' +
                        'Credentials saved to <code>browser.json</code>.<br><br>' +
                        '👉 <strong>Next Step:</strong> You may now close this browser tab, return to your terminal (press <code>Ctrl+C</code> to stop the login server), and run your application normally without the <code>--login</code> flag!';
                    btnText.textContent = 'Login Successful';
                    btnSpinner.style.display = 'none';
                    textarea.disabled = true;
                } else {
                    statusBox.className = 'status-msg error';
                    statusBox.style.display = 'block';
                    statusBox.textContent = data.error || 'Authentication failed. Please check your headers and try again.';
                    submitBtn.disabled = false;
                    btnSpinner.style.display = 'none';
                    btnText.textContent = 'Authenticate & Save Credentials';
                }
            } catch (err) {
                statusBox.className = 'status-msg error';
                statusBox.style.display = 'block';
                statusBox.textContent = 'Connection error: Unable to reach the local authentication server.';
                submitBtn.disabled = false;
                btnSpinner.style.display = 'none';
                btnText.textContent = 'Authenticate & Save Credentials';
            }
        }
    </script>
</body>
</html>
"""


import socket
import threading
import webbrowser
from http.server import BaseHTTPRequestHandler, HTTPServer


def get_free_port(default_port: int = 8989) -> int:
    for port in range(default_port, default_port + 50):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            if s.connect_ex(("127.0.0.1", port)) != 0:
                return port
    return default_port


def sanitize_headers_for_ytmusicapi(raw: str) -> str:
    """
    Sanitizes raw headers copied from DevTools or cURL commands.
    Strips pseudo-headers (:authority, :path, :method, :scheme, :status),
    URL paths, and cURL command wrappers before passing to ytmusicapi.setup.
    """
    # If input is a cURL command, extract header flags (-H / --header)
    curl_matches = re.findall(r"(?:-H|--header)\s+['\"]([^'\"]+)['\"]", raw)
    if curl_matches:
        return "\n".join(curl_matches)

    lines = raw.splitlines()
    cleaned: list[str] = []
    skip_next = False

    for line in lines:
        l = line.strip()
        if not l:
            continue
        if skip_next:
            skip_next = False
            continue
        if l.startswith(":"):
            if ":" not in l[1:]:
                skip_next = True
            continue
        if l.startswith(("/", "POST ", "GET ", "OPTIONS ", "HEAD ", "PUT ", "DELETE ", "curl ", "HTTP/")):
            continue
        cleaned.append(l)

    return "\n".join(cleaned)


class ReuseHTTPServer(HTTPServer):
    allow_reuse_address = True


def process_raw_headers_via_ytmusicapi(raw_headers: str, filepath: Path | None = None) -> tuple[bool, str, Path]:
    """
    Passes raw_headers directly to ytmusicapi.setup(filepath, headers_raw) and tests authentication.
    Writes credentials to a temporary sibling file, tests authentication, and atomically replaces target_path on success.
    Returns (success, message, target_path).
    """
    target_path = filepath if filepath is not None else get_browser_json_path()
    target_path.parent.mkdir(parents=True, exist_ok=True)

    if not raw_headers or not raw_headers.strip():
        return False, "No headers provided. Please paste your request headers into the box.", target_path

    temp_file = tempfile.NamedTemporaryFile(dir=target_path.parent, prefix="browser_", suffix=".json", delete=False)
    temp_path = Path(temp_file.name)
    temp_file.close()

    try:
        sanitized = sanitize_headers_for_ytmusicapi(raw_headers)
        setup(filepath=str(temp_path), headers_raw=sanitized)

        success, err_msg = test_authentication(temp_path)
        if success:
            temp_path.replace(target_path)
            return True, "Login successful", target_path
        else:
            if temp_path.exists():
                temp_path.unlink()
            return False, f"Authentication failed: {err_msg}", target_path
    except Exception as e:
        if temp_path.exists():
            temp_path.unlink()
        return False, str(e), target_path


class WebLoginHandler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        pass

    def do_GET(self):
        if self.path in ("/", "/index.html"):
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(LOGIN_HTML.encode("utf-8"))
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path == "/api/login":
            content_length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(content_length).decode("utf-8")
            raw_headers = ""
            try:
                data = json.loads(body)
                raw_headers = data.get("headers", "")
            except Exception:
                raw_headers = body

            success, err_msg, browser_json_path = process_raw_headers_via_ytmusicapi(raw_headers)

            if success:
                resp = {"success": True, "message": "Login successful"}
                self._send_json(200, resp)
                print(f"\nLogin successful! Saved credentials to: {browser_json_path}")
                threading.Thread(target=self.server.shutdown).start()
            else:
                resp = {
                    "success": False,
                    "error": err_msg,
                }
                self._send_json(400, resp)
        else:
            self.send_response(404)
            self.end_headers()

    def _send_json(self, status_code: int, data: dict):
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode("utf-8"))


def start_web_login_server() -> None:
    port = get_free_port(8989)
    server_address = ("127.0.0.1", port)
    httpd = ReuseHTTPServer(server_address, WebLoginHandler)
    url = f"http://localhost:{port}"

    print("================================================================================")
    print("  YOUTUBE MUSIC WEB LOGIN SERVER IS RUNNING")
    print("================================================================================")
    print(f"  URL: {url}")
    print("--------------------------------------------------------------------------------")
    print("  Opening browser automatically...")
    print("  If your browser does not open automatically, open the URL above manually.")
    print("================================================================================\n")

    try:
        webbrowser.open(url)
    except Exception:
        pass

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\nLogin canceled.")
        sys.exit(1)


def run_login_flow(file_path: str | None = None) -> None:
    """
    Main interactive authentication setup flow.
    If file_path is provided or non-interactive stdin, reads headers directly.
    Otherwise starts local Web UI server.
    """
    if file_path:
        p = Path(file_path)
        if not p.exists():
            print(f"[Error] Specified file does not exist: {file_path}")
            sys.exit(1)
        raw_headers = p.read_text(encoding="utf-8")
        _process_login_headers(raw_headers)
        return

    if not sys.stdin.isatty():
        raw_headers = sys.stdin.read()
        _process_login_headers(raw_headers)
        return

    # Start tiny local HTTP server for browser Web UI login
    start_web_login_server()


def _process_login_headers(raw_headers: str) -> None:
    if not raw_headers.strip():
        print("[Error] No headers provided.")
        sys.exit(1)

    success, err_msg, browser_json_path = process_raw_headers_via_ytmusicapi(raw_headers)
    if success:
        print("\n================================================================================")
        print("  AUTHENTICATION SUCCESSFUL!")
        print(f"  Saved credentials to: {browser_json_path}")
        print("  You may now close your browser, exit the server (Ctrl+C), and run")
        print("  your program normally without the --login flag!")
        print("================================================================================\n")
    else:
        print(f"\n[Error] {err_msg}")
        sys.exit(1)
