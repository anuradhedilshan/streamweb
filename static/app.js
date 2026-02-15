let config = null;

function channelTemplate(channel, isActive) {
  return `
  <div class="channel">
    <label><input type="radio" name="active_channel" value="${channel.id}" ${isActive ? "checked" : ""}> Active</label><br>
    <input placeholder="id" value="${channel.id}" data-field="id" />
    <input placeholder="name" value="${channel.name}" data-field="name" style="width: 40%" />
    <input placeholder="url" value="${channel.url}" data-field="url" style="width: 95%" />
  </div>`;
}

function render() {
  if (!config) return;
  document.getElementById("hls_time").value = config.hls_time;
  document.getElementById("hls_list_size").value = config.hls_list_size;
  document.getElementById("ffmpeg_threads").value = config.ffmpeg_threads;
  document.getElementById("player_delay_seconds").value = config.player_delay_seconds;

  const channelsDiv = document.getElementById("channels");
  channelsDiv.innerHTML = config.channels
    .map((c) => channelTemplate(c, c.id === config.active_channel_id))
    .join("");

  const relay = config.relay;
  document.getElementById("streamStatus").textContent = relay.running
    ? `Relay running (pid ${relay.pid}) source: ${relay.source_url}`
    : "Relay stopped";
}

function collectConfig() {
  const channelEls = Array.from(document.querySelectorAll("#channels .channel"));
  const channels = channelEls.map((el, idx) => {
    const inputs = el.querySelectorAll("input[data-field]");
    const channel = {
      id: inputs[0].value.trim() || `channel-${idx + 1}`,
      name: inputs[1].value.trim() || `Channel ${idx + 1}`,
      url: inputs[2].value.trim(),
    };
    return channel;
  });

  const active = document.querySelector("input[name='active_channel']:checked")?.value || channels[0]?.id;

  return {
    channels,
    active_channel_id: active,
    hls_time: Number(document.getElementById("hls_time").value),
    hls_list_size: Number(document.getElementById("hls_list_size").value),
    ffmpeg_threads: Number(document.getElementById("ffmpeg_threads").value),
    player_delay_seconds: Number(document.getElementById("player_delay_seconds").value),
  };
}

async function loadConfig() {
  const res = await fetch("/api/config");
  config = await res.json();
  render();
  setupPlayer();
}

function setupPlayer() {
  const video = document.getElementById("player");
  const streamUrl = "/hls/live.m3u8";
  if (Hls.isSupported()) {
    const hls = new Hls({
      lowLatencyMode: false,
      liveSyncDuration: Number(config.player_delay_seconds || 75),
      maxBufferLength: 180,
      maxMaxBufferLength: 240,
      backBufferLength: 180,
    });
    hls.loadSource(streamUrl);
    hls.attachMedia(video);
  } else {
    video.src = streamUrl;
  }
}

function addChannel() {
  config.channels.push({ id: `new-${Date.now()}`, name: "New channel", url: "" });
  render();
}

async function saveConfig() {
  const payload = collectConfig();
  const res = await fetch("/api/config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  document.getElementById("message").textContent = res.ok ? "Saved + restarted" : `Save error: ${await res.text()}`;
  await loadConfig();
}

async function startRelay() {
  await fetch("/api/start", { method: "POST" });
  await loadConfig();
}

async function stopRelay() {
  await fetch("/api/stop", { method: "POST" });
  await loadConfig();
}

loadConfig();
