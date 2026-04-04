# 🎙️ go-radio-streamer

Go service that takes an internet radio stream (MP3), decodes/resamples it via FFmpeg, and sends it as AES67-style RTP multicast (`L24/48000/2`) with Web UI, REST API, MQTT control, ICY metadata, and SAP/SDP announcement.

## 🚀 Installation

### Voraussetzungen
- Go (aktuell mit Go `1.26.x` gebaut)
- FFmpeg in `PATH`
- MQTT optional (ohne `mqtt.conf` läuft HTTP/Web trotzdem)

### FFmpeg installieren

```bash
# Debian/Ubuntu / Raspberry Pi OS
sudo apt-get update
sudo apt-get install -y ffmpeg

# macOS
brew install ffmpeg
```

### Projekt bauen & starten

```bash
cd /home/silly/go-radio-streamer
go build -o radio-streamer ./cmd
./radio-streamer
```

Server läuft dann auf `http://localhost:8080`.

### Raspberry Pi
- Vollständige Anleitung: `RASPBERRY_PI_SETUP.md`

## ⚙️ Konfiguration

Die App liest zwei Dateien aus dem Projekt-Root.

### `stations.txt`

```text
1. SRF-1 https://stream.srg-ssr.ch/m/rsrp1/mp3_128
2. SRF-2 https://stream.srg-ssr.ch/m/rsrp2/mp3_128
```

Hinweise:
- Erstes Token muss `<nummer>.` sein
- Zweites Token ist Stationsname (ein Token)
- Rest ist URL

### `mqtt.conf` (optional)

```text
broker=tcp://192.168.188.62:1883
user=your_user
pass=your_password
```

## ✨ Features

- RTP Multicast auf `239.0.0.1:5004`
- Audioformat: `L24/48000/2` (Payload Type `97`)
- Stabiler Senderwechsel über `POST /api/play` (kontrollierter Stop/Start)
- Stabilitätsorientiertes Tuning (`ptime=40`, Prebuffer, Silence bei Underrun)
- Web UI unter `/` mit Metadatenanzeige
- REST API für Steuerung, Status und SDP
- MQTT Steuerung + Status/Heartbeat
- SAP-Announcement auf `239.255.255.255:9875`

## 🌐 Web UI

Aufrufen:

```text
http://localhost:8080/
```

Funktionen:
- Start/Stop
- Senderwechsel ohne separaten Stop (Backend macht atomaren Switch)
- Anzeige von aktiver Station + `artist`/`track`

## 🔌 REST API

### `GET /api/stations`

```bash
curl -sS http://localhost:8080/api/stations
```

### `POST /api/play`

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

```bash
curl -sS http://localhost:8080/api/status
```

Beispielantwort:

```json
{
  "running": true,
  "station": "SRF-1",
  "artist": "...",
  "track": "...",
  "meta_updated": 1774982390
}
```

### `GET /api/stream.sdp`

```bash
curl -sS http://localhost:8080/api/stream.sdp
```

Query-Parameter:
- `ip` (Default `239.0.0.1`)
- `port` (Default `5004`)
- `pt` (Default `97`)
- `ptime` (Default `40`)
- `ptp` (Default `IEEE1588-2008:00-00-00-00-00-00-00-00:0`)

## 📡 MQTT

### Subscribe (Control)
- `gostreamer/play` (Payload: z. B. `1`)
- `gostreamer/stop` (Payload optional)

### Publish (Status)
- `gostreamer/current` (JSON)
- `gostreamer/heartbeat` (JSON, alle 30s)

Beispiele:

```bash
# subscribe
mosquitto_sub -h 192.168.188.62 -u your_user -P your_password -t 'gostreamer/#'

# play station 1
mosquitto_pub -h 192.168.188.62 -u your_user -P your_password -t 'gostreamer/play' -m '1'

# stop
mosquitto_pub -h 192.168.188.62 -u your_user -P your_password -t 'gostreamer/stop' -m ''
```

## 📣 SAP / Discovery

Beim Start wird SDP per SAP angekündigt:
- Ziel: `239.255.255.255:9875`
- Intervall: 30 Sekunden

Beim Stop wird ein SAP Delete-Announcement gesendet.

## 🧠 Technische Hinweise

- FFmpeg-Output: `pcm_s24be` (`s24be` raw)
- RTP-Clock: 48 kHz
- Größerer UDP-Write-Buffer für glatteres Senden
- Fokus auf Kontinuität statt ultra-niedriger Latenz

## 🛠️ Troubleshooting

### API antwortet nicht

```bash
ps aux | grep radio-streamer
curl -sS http://localhost:8080/api/status
```

### Kein Audio am Receiver
1. RTP-Verkehr prüfen:

```bash
tcpdump -ni any udp and port 5004
```

2. SDP prüfen:

```bash
curl -sS http://localhost:8080/api/stream.sdp
```

3. Receiver muss `L24/48000/2`, PT `97` erwarten.

### Audio knackt/stottert
- Defaults sind stabilitätsorientiert (`ptime=40`, Prebuffer, Silence bei Underrun)
- Receiver-Jitterbuffer erhöhen
- Host-/Netzlast und Paketverlust prüfen

### FFmpeg fehlt

```bash
ffmpeg -version
```

### MQTT nicht verfügbar oder `mqtt.conf` fehlt
- Server startet trotzdem (HTTP/Web-only)
- MQTT wird aktiviert, sobald gültige `mqtt.conf` vorhanden ist

## 👨‍💻 Entwicklung

```bash
cd /home/silly/go-radio-streamer
go test ./...
go build -o radio-streamer ./cmd
```

## 📄 Lizenz

Dieses Projekt steht unter der MIT-Lizenz. Details siehe `LICENSE`.

## 🗂️ Projektlayout

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
