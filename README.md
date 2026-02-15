# Stream Relay (Single Source M3U8 -> Buffered Web Playback)

This app runs **one managed relay** on your server and serves many clients from that same output.

- Admin sets one upstream `source_url` (m3u8).
- Server pulls it once with FFmpeg.
- Server re-encodes to browser-friendly H.264 + AAC HLS.
- Server keeps a rolling **few-minute buffer** window.
- All web clients watch from server output `/hls/live.m3u8`.

## Key improvement

- Buffer is configured in **minutes** (`buffer_minutes`) and calculated to HLS list size automatically.
- This means your server stores a rolling window and clients read from that buffered stream.
- Multi-client serving is supported through threaded HTTP server (`ThreadingHTTPServer`).

## Run on Ubuntu EC2

```bash
sudo apt update
sudo apt install -y python3 ffmpeg
cd /path/to/streamweb
python3 app.py
```

Open in browser:

```text
http://YOUR_EC2_PUBLIC_IP:8080
```

If unreachable, allow inbound TCP **8080** in EC2 Security Group.

## Run in background

```bash
nohup python3 app.py > app.log 2>&1 &
tail -f app.log
```

Stop:

```bash
pkill -f "python3 app.py"
```

## Admin config fields

- `source_url`: your m3u8 URL
- `preset`: ffmpeg speed preset (`veryfast` default)
- `hls_time`: segment duration (seconds)
- `buffer_minutes`: rolling buffer duration kept on server (minutes)
- `player_delay_seconds`: player delay
- `ffmpeg_threads`: `0` auto, or fixed threads
- `video_bitrate`: e.g. `2500k`
- `audio_bitrate`: e.g. `128k`

## Recommended for unstable source

- `hls_time = 4`
- `buffer_minutes = 2` to `3`
- `player_delay_seconds = 75` to `120`
- `preset = veryfast`

## API endpoints

- `GET /api/config` => config + runtime status
- `GET /api/status` => runtime status only
- `POST /api/config` => save config + restart relay
- `POST /api/start` => start relay
- `POST /api/stop` => stop relay

## Security note

Admin is intentionally unprotected for now (as requested). Add authentication before public production use.
