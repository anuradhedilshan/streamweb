import json
import os
import signal
import subprocess
import threading
import time
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse

BASE_DIR = Path(__file__).resolve().parent
DATA_DIR = BASE_DIR / "data"
OUTPUT_DIR = BASE_DIR / "stream_output"
STATIC_DIR = BASE_DIR / "static"
TEMPLATE_PATH = BASE_DIR / "templates" / "index.html"
CONFIG_PATH = DATA_DIR / "config.json"
PLAYLIST_PATH = OUTPUT_DIR / "live.m3u8"

DATA_DIR.mkdir(exist_ok=True)
OUTPUT_DIR.mkdir(exist_ok=True)

DEFAULT_CONFIG = {
    "source_url": "http://161.248.38.40:8000/play/a06v/index.m3u8",
    "hls_time": 4,
    "buffer_minutes": 2,
    "ffmpeg_threads": 0,
    "player_delay_seconds": 75,
    "video_bitrate": "2500k",
    "audio_bitrate": "128k",
    "preset": "veryfast",
}


class RelayManager:
    def __init__(self):
        self._process = None
        self._lock = threading.Lock()
        self.started_at = 0.0
        self.last_error = ""
        self.last_exit_code = None
        self.current_source_url = ""
        self.effective_hls_list_size = 0

    def _cleanup_output(self):
        for item in OUTPUT_DIR.glob("live*"):
            if item.is_file():
                item.unlink(missing_ok=True)

    def _stop_unlocked(self):
        if not self._process or self._process.poll() is not None:
            self._process = None
            return
        pgid = os.getpgid(self._process.pid)
        os.killpg(pgid, signal.SIGTERM)
        for _ in range(40):
            if self._process.poll() is not None:
                break
            time.sleep(0.1)
        if self._process.poll() is None:
            os.killpg(pgid, signal.SIGKILL)
        self.last_exit_code = self._process.poll()
        self._process = None

    def start(self, cfg):
        with self._lock:
            self._stop_unlocked()
            self._cleanup_output()

            hls_list_size = max(15, int((cfg["buffer_minutes"] * 60) / cfg["hls_time"]))
            self.effective_hls_list_size = hls_list_size

            cmd = [
                "ffmpeg",
                "-hide_banner",
                "-loglevel",
                "warning",
                "-fflags",
                "+genpts+discardcorrupt",
                "-reconnect",
                "1",
                "-reconnect_streamed",
                "1",
                "-reconnect_delay_max",
                "5",
                "-i",
                cfg["source_url"],
                "-map",
                "0:v:0",
                "-map",
                "0:a:0?",
                "-c:v",
                "libx264",
                "-preset",
                cfg["preset"],
                "-tune",
                "zerolatency",
                "-pix_fmt",
                "yuv420p",
                "-g",
                "48",
                "-keyint_min",
                "48",
                "-sc_threshold",
                "0",
                "-b:v",
                cfg["video_bitrate"],
                "-c:a",
                "aac",
                "-b:a",
                cfg["audio_bitrate"],
                "-ac",
                "2",
                "-ar",
                "48000",
                "-threads",
                str(cfg["ffmpeg_threads"]),
                "-f",
                "hls",
                "-hls_time",
                str(cfg["hls_time"]),
                "-hls_list_size",
                str(hls_list_size),
                "-hls_flags",
                "append_list+independent_segments+program_date_time",
                "-hls_segment_type",
                "mpegts",
                "-hls_allow_cache",
                "1",
                "-hls_segment_filename",
                str(OUTPUT_DIR / "live_%05d.ts"),
                str(PLAYLIST_PATH),
            ]

            try:
                self._process = subprocess.Popen(
                    cmd,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    preexec_fn=os.setsid,
                )
                self.started_at = time.time()
                self.last_error = ""
                self.last_exit_code = None
                self.current_source_url = cfg["source_url"]
            except FileNotFoundError:
                self._process = None
                self.last_error = "ffmpeg not installed"

    def stop(self):
        with self._lock:
            self._stop_unlocked()

    def status(self):
        with self._lock:
            running = bool(self._process and self._process.poll() is None)
            if self._process and not running:
                self.last_exit_code = self._process.poll()
                self._process = None

            playlist_exists = PLAYLIST_PATH.exists()
            segment_count = len(list(OUTPUT_DIR.glob("live_*.ts")))
            playlist_age = None
            if playlist_exists:
                playlist_age = round(time.time() - PLAYLIST_PATH.stat().st_mtime, 1)

            return {
                "running": running,
                "pid": self._process.pid if running and self._process else None,
                "uptime_seconds": round(time.time() - self.started_at, 1) if running else 0,
                "source_url": self.current_source_url,
                "playlist_exists": playlist_exists,
                "playlist_age_seconds": playlist_age,
                "segment_count": segment_count,
                "effective_hls_list_size": self.effective_hls_list_size,
                "effective_buffer_seconds": self.effective_hls_list_size * load_config()["hls_time"],
                "last_exit_code": self.last_exit_code,
                "last_error": self.last_error,
            }


relay_manager = RelayManager()


def load_config():
    if not CONFIG_PATH.exists():
        save_config(DEFAULT_CONFIG)
        return DEFAULT_CONFIG.copy()
    raw = json.loads(CONFIG_PATH.read_text())
    cfg = DEFAULT_CONFIG.copy()
    cfg.update(raw)
    return cfg


def save_config(cfg):
    CONFIG_PATH.write_text(json.dumps(cfg, indent=2))


def validate_config(cfg):
    if not isinstance(cfg.get("source_url"), str) or not cfg["source_url"].strip():
        raise ValueError("source_url is required")
    scheme = urlparse(cfg["source_url"]).scheme.lower()
    if scheme not in {"http", "https"}:
        raise ValueError("source_url must be http/https")

    int_ranges = {
        "hls_time": (2, 10),
        "buffer_minutes": (1, 10),
        "ffmpeg_threads": (0, 16),
        "player_delay_seconds": (20, 240),
    }
    for key, (low, high) in int_ranges.items():
        val = int(cfg.get(key, 0))
        if not (low <= val <= high):
            raise ValueError(f"{key} must be between {low} and {high}")

    for key in ["video_bitrate", "audio_bitrate", "preset"]:
        if not isinstance(cfg.get(key), str) or not cfg[key].strip():
            raise ValueError(f"{key} is required")


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def _send_json(self, payload, status=HTTPStatus.OK):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    def _serve_file(self, path, content_type):
        if not path.exists() or not path.is_file():
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        data = path.read_bytes()
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        if path.suffix in {".m3u8", ".ts"}:
            self.send_header("Cache-Control", "public, max-age=2")
        else:
            self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path == "/":
            return self._serve_file(TEMPLATE_PATH, "text/html; charset=utf-8")
        if self.path == "/static/app.js":
            return self._serve_file(STATIC_DIR / "app.js", "application/javascript; charset=utf-8")
        if self.path == "/api/config":
            cfg = load_config()
            return self._send_json({"config": cfg, "status": relay_manager.status()})
        if self.path == "/api/status":
            return self._send_json(relay_manager.status())
        if self.path.startswith("/hls/"):
            rel = self.path.replace("/hls/", "", 1)
            target = OUTPUT_DIR / rel
            if rel.endswith(".m3u8"):
                return self._serve_file(target, "application/vnd.apple.mpegurl")
            if rel.endswith(".ts"):
                return self._serve_file(target, "video/mp2t")
            return self.send_error(HTTPStatus.NOT_FOUND)
        self.send_error(HTTPStatus.NOT_FOUND)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        data = self.rfile.read(length) if length > 0 else b"{}"
        try:
            payload = json.loads(data.decode("utf-8") or "{}")
        except json.JSONDecodeError:
            return self._send_json({"error": "invalid JSON"}, HTTPStatus.BAD_REQUEST)

        if self.path == "/api/config":
            cfg = DEFAULT_CONFIG.copy()
            cfg.update(payload)
            try:
                validate_config(cfg)
            except ValueError as exc:
                return self._send_json({"error": str(exc)}, HTTPStatus.BAD_REQUEST)
            save_config(cfg)
            relay_manager.start(cfg)
            return self._send_json({"status": "saved_and_restarted", "config": cfg})

        if self.path == "/api/start":
            cfg = load_config()
            relay_manager.start(cfg)
            return self._send_json({"status": "started"})

        if self.path == "/api/stop":
            relay_manager.stop()
            return self._send_json({"status": "stopped"})

        self.send_error(HTTPStatus.NOT_FOUND)


class BetterThreadingHTTPServer(ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = True


def run_server(host="0.0.0.0", port=8080):
    cfg = load_config()
    relay_manager.start(cfg)

    server = BetterThreadingHTTPServer((host, port), Handler)
    print(f"Server listening on http://{host}:{port}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        relay_manager.stop()
        server.server_close()


if __name__ == "__main__":
    run_server()
