let state = null;
let hls = null;

function byId(id) {
  return document.getElementById(id);
}

function loadForm(cfg) {
  byId("source_url").value = cfg.source_url;
  byId("preset").value = cfg.preset;
  byId("hls_time").value = cfg.hls_time;
  byId("buffer_minutes").value = cfg.buffer_minutes;
  byId("player_delay_seconds").value = cfg.player_delay_seconds;
  byId("ffmpeg_threads").value = cfg.ffmpeg_threads;
  byId("video_bitrate").value = cfg.video_bitrate;
  byId("audio_bitrate").value = cfg.audio_bitrate;
}

function collectForm() {
  return {
    source_url: byId("source_url").value.trim(),
    preset: byId("preset").value,
    hls_time: Number(byId("hls_time").value),
    buffer_minutes: Number(byId("buffer_minutes").value),
    player_delay_seconds: Number(byId("player_delay_seconds").value),
    ffmpeg_threads: Number(byId("ffmpeg_threads").value),
    video_bitrate: byId("video_bitrate").value.trim(),
    audio_bitrate: byId("audio_bitrate").value.trim(),
  };
}

function renderStatus(status) {
  const healthy = status.running && status.playlist_exists && (status.playlist_age_seconds === null || status.playlist_age_seconds < 20);
  const css = healthy ? "status-ok" : "status-bad";
  byId("runtimeStatus").innerHTML = `
    <span class="${css}">Relay: ${healthy ? "HEALTHY" : "NOT HEALTHY"}</span><br>
    running=${status.running} pid=${status.pid ?? "-"} uptime=${status.uptime_seconds}s<br>
    segments_on_disk=${status.segment_count} playlist_age=${status.playlist_age_seconds ?? "-"}s<br>
    server_buffer=${status.effective_buffer_seconds ?? "-"}s list_size=${status.effective_hls_list_size ?? "-"}<br>
    source=${status.source_url || "-"}<br>
    last_error=${status.last_error || "none"} last_exit_code=${status.last_exit_code ?? "-"}
  `;
}

function setupPlayer(delaySeconds) {
  const video = byId("player");
  const src = "/hls/live.m3u8";

  if (hls) {
    hls.destroy();
    hls = null;
  }

  if (window.Hls && Hls.isSupported()) {
    hls = new Hls({
      lowLatencyMode: false,
      liveSyncDuration: Number(delaySeconds || 75),
      maxBufferLength: 240,
      maxMaxBufferLength: 360,
      backBufferLength: 180,
      enableWorker: true,
    });
    hls.loadSource(src);
    hls.attachMedia(video);
  } else {
    video.src = src;
  }
}

async function loadState() {
  const res = await fetch("/api/config", { cache: "no-store" });
  state = await res.json();
  loadForm(state.config);
  renderStatus(state.status);
  setupPlayer(state.config.player_delay_seconds);
}

async function saveConfig() {
  const payload = collectForm();
  const res = await fetch("/api/config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  const text = await res.text();
  byId("message").textContent = res.ok ? "Saved and relay restarted." : `Save failed: ${text}`;
  await loadState();
}

async function startRelay() {
  await fetch("/api/start", { method: "POST" });
  byId("message").textContent = "Relay start requested.";
  await loadState();
}

async function stopRelay() {
  await fetch("/api/stop", { method: "POST" });
  byId("message").textContent = "Relay stopped.";
  await loadState();
}

async function refreshStatusOnly() {
  const res = await fetch("/api/status", { cache: "no-store" });
  if (!res.ok) return;
  const status = await res.json();
  renderStatus(status);
}

loadState();
setInterval(refreshStatusOnly, 5000);
