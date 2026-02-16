# MPV Launcher Scaffold

CLI commands to implement:

- `player login`
- `player play <stream_id>`
- `player logout`

## Playback flow

1. Login and store token.
2. `POST /playback/start` with stream id.
3. Receive `play_url` and session id.
4. Launch `mpv <play_url>`.
5. Start heartbeat every 10 seconds to `/playback/heartbeat`.
6. If status `blocked`, stop mpv process and exit.

## Runtime requirements

- keep session token refreshed
- retry transient API/network failures
- print points balance and spend rate periodically
