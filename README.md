# go-radio-streamer

Go service that takes an internet radio stream (MP3), decodes/resamples it via FFmpeg, and sends it as AES67-style RTP multicast (`L24/48000/2`) with Web UI, REST API, MQTT control, ICY metadata, and SAP/SDP announcement.

## Features

- RTP multicast audio to `239.0.0.1:5004`
- Audio format: `L24/48000/2` (payload type `97`)
- Clean station switching with controlled stop/start on `POST /api/play`
- Stability-first sender tuning (not low-latency):
  - default `ptime=40`
  - prebuffering before send
  - underrun handling with silence packets
- Web UI at `/` for play/stop and current metadata
- REST API for stations, play/stop, status, and SDP
- MQTT control + status/heartbeat topics
- SAP announcement (`239.255.255.255:9875`) carrying SDP

## Requirements

- Go (project currently built with Go 1.26.x)
- FFmpeg in `PATH`
- MQTT broker optional (`mqtt.conf` may be missing; HTTP/Web still starts)

Raspberry Pi deployment:
- See `RASPBERRY_PI_SETUP.md` for a full Raspberry Pi OS installation guide.

Examples:

```bash
# Debian/Ubuntu
sudo apt-get update
sudo apt-get install -y ffmpeg

# macOS
brew install ffmpeg
```

## Configuration

The app reads two files from the project root.

### `stations.txt`

Format per line:

```text
1. SRF-1 https://stream.srg-ssr.ch/m/rsrp1/mp3_128
2. SRF-2 https://stream.srg-ssr.ch/m/rsrp2/mp3_128
```

Notes:
- First token must be `<number>.`
- Second token is station name (single token)
- Remaining token(s) are treated as URL

### `mqtt.conf`

```text
broker=tcp://192.168.188.62:1883
user=your_user
pass=your_password
```

## Build & Run

```bash
cd /home/silly/go-radio-streamer
go build -o radio-streamer ./cmd
./radio-streamer
```

Server listens on `http://localhost:8080`.

## Web UI

Open:

```text
http://localhost:8080/
```

Functions:
- Start/stop streaming
- Senderwechsel ohne separaten Stop (Backend übernimmt atomaren Switch)
- Show active station
- Show metadata (`artist`, `track`) via status polling

## REST API

### `GET /api/stations`

Returns configured stations.

```bash
curl -sS http://localhost:8080/api/stations
```

### `POST /api/play`

Start station by index (1-based).

```bash
curl -sS -X POST http://localhost:8080/api/play \
  -H 'Content-Type: application/json' \
  -d '{"station":1}'
```

### `POST /api/stop`

```bash
curl -sS -X POST http://localhost:8080/api/stop
```

### `GET /api/status`

Current runtime status:

```json
{
  "running": true,
  "station": "SRF-1",
  "artist": "...",
  "track": "...",
  "meta_updated": 1774982390
}
```

```bash
curl -sS http://localhost:8080/api/status
```

### `GET /api/stream.sdp`

Generates SDP matching the stream settings.

```bash
curl -sS http://localhost:8080/api/stream.sdp
```

Supported query params:
- `ip` (default `239.0.0.1`)
- `port` (default `5004`)
- `pt` (default `97`)
- `ptime` (default `40`)
- `ptp` (default `IEEE1588-2008:00-00-00-00-00-00-00-00:0`)

## MQTT

### Subscribed topics (control)

- `gostreamer/play` payload: plain station number, e.g. `1`
- `gostreamer/stop` payload: ignored (can be empty)

### Published topics (status)

- `gostreamer/current` JSON status updates
- `gostreamer/heartbeat` JSON heartbeat every 30s

Examples:

```bash
# subscribe
mosquitto_sub -h 192.168.188.62 -u your_user -P your_password -t 'gostreamer/#'

# play station 1
mosquitto_pub -h 192.168.188.62 -u your_user -P your_password -t 'gostreamer/play' -m '1'

# stop
mosquitto_pub -h 192.168.188.62 -u your_user -P your_password -t 'gostreamer/stop' -m ''
```

## SAP / Discovery

When streaming starts, the app sends SAP announcements with SDP to:

- `239.255.255.255:9875`
- interval: 30s

When streaming stops, a SAP deletion announce is sent.

## Technical Notes

- FFmpeg output format is `pcm_s24be` (`s24be` raw)
- RTP timestamps run at 48 kHz clock
- UDP socket uses increased write buffer for smoother sending
- Sender prioritizes continuity over low latency

## Troubleshooting

### No server response on API

Check service is running:

```bash
ps aux | grep radio-streamer
curl -sS http://localhost:8080/api/status
```

### No audio on receiver

1. Verify RTP traffic exists:

```bash
tcpdump -ni any udp and port 5004
```

2. Verify SDP matches receiver setup:

```bash
curl -sS http://localhost:8080/api/stream.sdp
```

3. Ensure receiver expects `L24/48000/2`, PT `97`.

### Audio stutter / crackle

- Current defaults are stability-first (`ptime=40`, prebuffering, silence on underrun)
- If needed, increase receiver-side jitter buffer
- Check host/network load and packet loss

### FFmpeg not found

Install FFmpeg and verify:

```bash
ffmpeg -version
```

### MQTT unavailable / no `mqtt.conf`

- Server starts anyway (HTTP/Web only mode)
- MQTT control/status is disabled until valid `mqtt.conf` is present

### Raspberry Pi install

- Full setup (packages, Go version, systemd, multicast checks): `RASPBERRY_PI_SETUP.md`

## Development

```bash
cd /home/silly/go-radio-streamer
go test ./...
go build -o radio-streamer ./cmd
```

## Project Layout

```text
go-radio-streamer/
├── RASPBERRY_PI_SETUP.md
├── cmd/main.go
├── internal/
│   ├── api/handlers.go
│   ├── config/config.go
│   ├── mqtt/mqtt.go
│   ├── streamer/
│   │   ├── metadata.go
│   │   ├── streamer.go
│   │   └── streamer_test.go
│   └── web/
│       ├── static/
│       └── web.go
├── pkg/aes67/
│   ├── sap.go
│   ├── sdp.go
│   └── stream.go
├── stations.txt
├── mqtt.conf
├── go.mod
└── README.md
```

---

Last updated: 4. April 2026
