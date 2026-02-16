# Pipeline Worker Scaffold

Responsibilities:

- Start ffmpeg/shaka pipeline from admin config
- Generate ABR renditions
- Package HLS
- Write assets to MinIO under `streams/{stream_id}/...`
- Emit heartbeat to control-plane every 10 seconds
- Report `last_manifest_at` and errors

## Runtime loop

- Poll desired state
- Ensure process running for `live`
- Stop process for `paused|disabled`
- Self-heal worker restart on transient failures
