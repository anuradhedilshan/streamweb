import json
import os
import signal
import subprocess
import threading
import time
from http import HTTPStatus
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent
DATA_DIR = BASE_DIR / "data"
OUTPUT_DIR = BASE_DIR / "stream_output"
STATIC_DIR = BASE_DIR / "static"
TEMPLATE_DIR = BASE_DIR / "templates"
CONFIG_PATH = DATA_DIR / "config.json"

DATA_DIR.mkdir(exist_ok=True)
OUTPUT_DIR.mkdir(exist_ok=True)

DEFAULT_CONFIG = {
    "channels": [
        {"id": "star-hindi-sd", "name": "Star Sports 1 Hindi (576p)", "url": "http://161.248.38.40:8000/play/a071/index.m3u8"},
        {"id": "star-hindi-hd", "name": "Star Sports 1 Hindi HD (1080p)", "url": "http://161.248.38.40:8000/play/a06v/index.m3u8"},
    ],
    "active_channel_id": "star-hindi-hd",
    "hls_time": 4,
    "hls_list_size": 30,
    "ffmpeg_threads": 0,
    "player_delay_seconds": 75,
}


class RelayManager:
    def __init__(self):
        self._process = None
        self._lock = threading.Lock()
        self.source_url = ""
        self.last_error = ""

    def _cleanup_output(self):
        for file in OUTPUT_DIR.glob("live*"):
            if file.is_file():
                file.unlink(missing_ok=True)

    def status(self):
        with self._lock:
            running = bool(self._process and self._process.poll() is None)
            return {
                "running": running,
                "source_url": self.source_url,
                "pid": self._process.pid if running and self._process else None,
                "last_error": self.last_error,
            }

    def _stop_unlocked(self):
        if not self._process or self._process.poll() is not None:
            self._process = None
            return
        pgid = os.getpgid(self._process.pid)
        os.killpg(pgid, signal.SIGTERM)
        for _ in range(30):
            if self._process.poll() is not None:
                break
            time.sleep(0.1)
        if self._process.poll() is None:
            os.killpg(pgid, signal.SIGKILL)
        self._process = None

    def start(self, source_url, hls_time, hls_list_size, ffmpeg_threads):
        with self._lock:
            self._stop_unlocked()
            self._cleanup_output()
            cmd = [
                "ffmpeg", "-hide_banner", "-loglevel", "warning",
                "-reconnect", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "5",
                "-i", source_url,
                "-c", "copy",
                "-threads", str(ffmpeg_threads),
                "-f", "hls",
                "-hls_time", str(hls_time),
                "-hls_list_size", str(hls_list_size),
                "-hls_flags", "delete_segments+append_list+program_date_time",
                "-hls_segment_filename", str(OUTPUT_DIR / "live_%05d.ts"),
                str(OUTPUT_DIR / "live.m3u8"),
            ]
            try:
                self._process = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, preexec_fn=os.setsid)
                self.source_url = source_url
                self.last_error = ""
            except FileNotFoundError:
                self._process = None
                self.last_error = "ffmpeg not installed on server"

    def stop(self):
        with self._lock:
            self._stop_unlocked()


relay_manager = RelayManager()


def load_config():
    if not CONFIG_PATH.exists():
        save_config(DEFAULT_CONFIG)
        return DEFAULT_CONFIG
    return json.loads(CONFIG_PATH.read_text())


def save_config(config):
    CONFIG_PATH.write_text(json.dumps(config, indent=2))


def validate_config(cfg):
    required = {"channels", "active_channel_id", "hls_time", "hls_list_size", "ffmpeg_threads", "player_delay_seconds"}
    if not required.issubset(cfg.keys()):
        raise ValueError("Missing config fields")
    if not cfg["channels"]:
        raise ValueError("At least one channel is required")
    if not any(c.get("id") == cfg["active_channel_id"] for c in cfg["channels"]):
        raise ValueError("active_channel_id not found")


class Handler(SimpleHTTPRequestHandler):
    def _json(self, payload, status=HTTPStatus.OK):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _serve_file(self, path, content_type="text/plain"):
        if not path.exists() or not path.is_file():
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        data = path.read_bytes()
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path == "/":
            return self._serve_file(TEMPLATE_DIR / "index.html", "text/html; charset=utf-8")
        if self.path == "/static/app.js":
            return self._serve_file(STATIC_DIR / "app.js", "application/javascript; charset=utf-8")
        if self.path == "/api/config":
            cfg = load_config()
            return self._json({**cfg, "relay": relay_manager.status()})
        if self.path.startswith("/hls/"):
            rel = self.path.replace("/hls/", "", 1)
            ctype = "application/vnd.apple.mpegurl" if rel.endswith(".m3u8") else "video/mp2t"
            return self._serve_file(OUTPUT_DIR / rel, ctype)
        self.send_error(HTTPStatus.NOT_FOUND)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length > 0 else b"{}"
        payload = json.loads(raw.decode() or "{}")

        if self.path == "/api/config":
            try:
                validate_config(payload)
            except ValueError as exc:
                return self._json({"error": str(exc)}, HTTPStatus.BAD_REQUEST)
            save_config(payload)
            source = next(c["url"] for c in payload["channels"] if c["id"] == payload["active_channel_id"])
            relay_manager.start(source, payload["hls_time"], payload["hls_list_size"], payload["ffmpeg_threads"])
            return self._json({"status": "saved_and_restarted"})

        if self.path == "/api/start":
            cfg = load_config()
            source = next((c["url"] for c in cfg["channels"] if c["id"] == cfg["active_channel_id"]), "")
            if not source:
                return self._json({"error": "No active source configured"}, HTTPStatus.BAD_REQUEST)
            relay_manager.start(source, cfg["hls_time"], cfg["hls_list_size"], cfg["ffmpeg_threads"])
            return self._json({"status": "started"})

        if self.path == "/api/stop":
            relay_manager.stop()
            return self._json({"status": "stopped"})

        self.send_error(HTTPStatus.NOT_FOUND)


def run_server(host="0.0.0.0", port=8080):
    cfg = load_config()
    source = next((c["url"] for c in cfg["channels"] if c["id"] == cfg["active_channel_id"]), "")
    if source:
        relay_manager.start(source, cfg["hls_time"], cfg["hls_list_size"], cfg["ffmpeg_threads"])

    httpd = ThreadingHTTPServer((host, port), Handler)
    print(f"Server listening on http://{host}:{port}")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        relay_manager.stop()
        httpd.server_close()


if __name__ == "__main__":
    run_server()
