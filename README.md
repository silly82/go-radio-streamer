# 🎙️ go-radio-streamer

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Raspberry%20Pi-2ea44f)](./RASPBERRY_PI_SETUP.md)

## 🇬🇧 English

Go service that takes an internet radio stream (MP3), decodes/resamples it via FFmpeg, and sends it as AES67-style RTP multicast (`L24/48000/2`) with Web UI, REST API, MQTT control, ICY metadata, and SAP/SDP announcement.

### 🚀 Installation

- Go (`1.26.x`)
- FFmpeg in `PATH`
- MQTT optional (without `mqtt.conf`, HTTP/Web still runs)

```bash
# Debian/Ubuntu / Raspberry Pi OS
sudo apt-get update
sudo apt-get install -y ffmpeg

cd /home/silly/go-radio-streamer
go build -o radio-streamer ./cmd
./radio-streamer
```

Server URL: `http://localhost:8080`

Raspberry Pi guide: `RASPBERRY_PI_SETUP.md`

### 🔌 API Quickstart

```bash
curl -sS http://localhost:8080/api/stations
curl -sS -X POST http://localhost:8080/api/play -H 'Content-Type: application/json' -d '{"station":1}'
curl -sS -X POST http://localhost:8080/api/stop
curl -sS http://localhost:8080/api/status
curl -sS http://localhost:8080/api/stream.sdp
```

### 📡 MQTT Topics

- Control: `gostreamer/play`, `gostreamer/stop`
- Status: `gostreamer/current`, `gostreamer/heartbeat`

### 📄 License

MIT (`LICENSE`)

---

## 🇨🇭 Deutsch

Go-Service, der einen Internet-Radiostream (MP3) per FFmpeg dekodiert/resampelt und als AES67-ähnlichen RTP-Multicast (`L24/48000/2`) mit Web-UI, REST-API, MQTT-Steuerung, ICY-Metadaten und SAP/SDP-Announcement sendet.

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

### `streamer.conf` (optional)

```text
# Multicast address for RTP streaming and SAP announcements
multicast_address=239.69.250.171:5004
```

If `streamer.conf` is absent the default is `239.69.250.171:5004`.

## ✨ Features

- RTP Multicast auf `239.69.250.171:5004` (konfigurierbar in `streamer.conf`)
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
- `originip` (Default: auto-detected local IP address)

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

### Hardware-Receiver (z. B. Nota 142)

Hardware-AES67-Receiver werten die SDP-Attribute strikt aus. Das SDP enthält `a=sendonly`, welches als Pflichtattribut für Sende-Sessions gemäß AES67-Standard gilt. Ohne dieses Attribut ignorieren viele Hardware-Receiver den Stream.

Prüfen Sie folgende Punkte:
- SDP enthält `a=sendonly` (ab dieser Version automatisch vorhanden)
- Multicast-Adresse und Port stimmen mit der Konfiguration überein: `239.69.250.171:5004`
- Receiver ist im gleichen Multicast-Netzwerksegment (IGMP Snooping aktiv?)
- Bei PTP-synchronisierten Receivern: `ptp_ref_clock` in `streamer.conf` setzen

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
├── streamer.conf
├── go.mod
└── README.md
```

---

Last updated: 4. April 2026
