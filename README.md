# Simple Stream Relay (M3U8 -> Your Server -> Web UI)

Very simple project for Ubuntu AWS EC2:

- Input: any M3U/M3U8 source URL.
- Server pulls stream with FFmpeg.
- Server re-publishes local HLS (`/hls/live.m3u8`) with bigger buffer.
- Web admin page (no auth) to change URLs/settings.

## Features

- Unprotected admin panel (`/`) to edit channels and choose active channel.
- FFmpeg relay with reconnect flags.
- Adjustable buffer via `hls_time` + `hls_list_size` (ex: 4s * 30 = ~120s).
- Adjustable player delay (`player_delay_seconds`) to avoid source jitter.

## Quick install (Ubuntu EC2)

```bash
sudo apt update
sudo apt install -y ffmpeg python3 python3-pip
cd /workspace/streamweb
python3 app.py
```

Open: `http://YOUR_EC2_IP:8080`

## Default channels

- `http://161.248.38.40:8000/play/a071/index.m3u8`
- `http://161.248.38.40:8000/play/a06v/index.m3u8`

You can replace with any M3U8 source in admin UI.

## Performance tips

- Use EC2 near source region (lower latency).
- Start with:
  - `hls_time = 4`
  - `hls_list_size = 30` (~2 min playlist)
  - `player_delay_seconds = 60-90`
- Keep `ffmpeg_threads = 0` (auto) unless CPU bottleneck.
- Put app behind Nginx reverse proxy for production.

## Run in background

```bash
nohup python3 app.py > app.log 2>&1 &
```
