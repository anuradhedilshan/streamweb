# Simple Stream Relay (M3U8 -> Your Server -> Web UI)

This project relays unstable IPTV/M3U8 streams through your own server so your users get a more stable playback URL with bigger buffering.

- Input: any M3U/M3U8 source URL.
- Server pulls stream with FFmpeg.
- Server re-publishes local HLS as `/hls/live.m3u8`.
- Web admin page (no auth) to change URLs/settings.

---

## 1) Server requirements (Ubuntu EC2)

- Ubuntu 22.04 or 24.04
- Public IP / DNS
- Python 3
- FFmpeg (required)

Install dependencies:

```bash
sudo apt update
sudo apt install -y python3 ffmpeg
```

---

## 2) Upload code to server

If using git:

```bash
git clone <your-repo-url> streamweb
cd streamweb
```

Or copy the folder manually, then:

```bash
cd /path/to/streamweb
```

---

## 3) Start the app (quick test)

```bash
python3 app.py
```

You should see:

```text
Server listening on http://0.0.0.0:8080
```

Then open in browser:

```text
http://YOUR_EC2_PUBLIC_IP:8080
```

If page not opening, open EC2 Security Group inbound port **8080**.

---

## 4) Keep app running in background

Simple way:

```bash
nohup python3 app.py > app.log 2>&1 &
```

Check process:

```bash
ps aux | grep app.py
```

Check logs:

```bash
tail -f app.log
```

Stop process:

```bash
pkill -f "python3 app.py"
```

---

## 5) Production way (systemd service)

Create service file:

```bash
sudo nano /etc/systemd/system/streamweb.service
```

Paste this (update path/user if needed):

```ini
[Unit]
Description=StreamWeb HLS Relay
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/streamweb
ExecStart=/usr/bin/python3 /home/ubuntu/streamweb/app.py
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

Enable/start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable streamweb
sudo systemctl start streamweb
```

Check status/logs:

```bash
sudo systemctl status streamweb
sudo journalctl -u streamweb -f
```

---

## 6) Configure stream URLs in web UI

Open:

```text
http://YOUR_EC2_PUBLIC_IP:8080
```

In **Admin (Unprotected)**:

1. Add/edit channels (`id`, `name`, `url`).
2. Select active channel radio button.
3. Set buffering values (recommended start):
   - `hls_time = 4`
   - `hls_list_size = 30` (about 2 minutes)
   - `player_delay_seconds = 60-90`
   - `ffmpeg_threads = 0` (auto)
4. Click **Save + Restart Relay**.

Player URL you can embed elsewhere:

```text
http://YOUR_EC2_PUBLIC_IP:8080/hls/live.m3u8
```

---

## 7) Your provided sample URLs

- `http://161.248.38.40:8000/play/a071/index.m3u8`
- `http://161.248.38.40:8000/play/a06v/index.m3u8`

These are already in default config and can be changed from the UI.

---

## 8) Optional: Nginx reverse proxy on port 80

Install nginx:

```bash
sudo apt install -y nginx
```

Create site file:

```bash
sudo nano /etc/nginx/sites-available/streamweb
```

Paste:

```nginx
server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

Enable + reload:

```bash
sudo ln -s /etc/nginx/sites-available/streamweb /etc/nginx/sites-enabled/streamweb
sudo nginx -t
sudo systemctl reload nginx
```

Now open:

```text
http://YOUR_EC2_PUBLIC_IP
```

---

## 9) Troubleshooting

### A) UI loads but stream does not play
- Make sure FFmpeg is installed:
  ```bash
  ffmpeg -version
  ```
- Check relay status in `/api/config`:
  ```bash
  curl http://127.0.0.1:8080/api/config
  ```
- Try another source URL (some IPTV links expire quickly).

### B) Playback stutters
- Increase buffer:
  - `hls_list_size`: 30 -> 45
  - `player_delay_seconds`: 75 -> 90
- Use EC2 region closer to source server.

### C) Port not reachable
- Open inbound in Security Group:
  - 8080 (or 80 if using nginx)
- Check UFW if enabled:
  ```bash
  sudo ufw status
  ```

---

## Security note

Admin UI is intentionally **unprotected** right now (as requested). Do not expose publicly for sensitive/private deployments without adding auth.
