# Raspberry Pi Setup (Raspberry Pi OS)

Diese Anleitung beschreibt die Installation auf einem Raspberry Pi mit Standard Raspberry Pi OS (Bookworm).

Stand: 4. April 2026

## 1) Vorab-Check

```bash
uname -m
cat /etc/os-release
```

Empfehlung:
- `aarch64` (64-bit Raspberry Pi OS) bevorzugen
- 32-bit (`armv7l`) geht grundsätzlich auch, aber 64-bit ist stabiler/performanter

## 2) System vorbereiten

```bash
sudo apt update
sudo apt upgrade -y
sudo apt install -y ffmpeg git curl ca-certificates
```

Warum wichtig:
- `ffmpeg` ist zwingend notwendig (MP3 decode + resample zu `s24be`)

## 3) Go installieren (wichtig)

Das Projekt nutzt in `go.mod` aktuell `go 1.26`.
Je nach Raspberry Pi OS kann `apt` eine ältere Go-Version liefern.

Version prüfen:

```bash
go version
```

Wenn kleiner als `1.26`, Go manuell aktualisieren (Beispiel ARM64):

```bash
cd /tmp
curl -LO https://go.dev/dl/go1.26.0.linux-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.0.linux-arm64.tar.gz

echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
go version
```

Für 32-bit Pi stattdessen `linux-armv6l`/`linux-armv7l` passendes Archiv von go.dev verwenden.

## 4) Projekt holen und bauen

```bash
cd ~
git clone https://github.com/silly82/go-radio-streamer.git
cd go-radio-streamer
go build -o radio-streamer ./cmd
```

## 5) Konfiguration

### `stations.txt`

Beispiel:

```text
1. SRF-1 https://stream.srg-ssr.ch/m/rsrp1/mp3_128
2. SRF-2 https://stream.srg-ssr.ch/m/rsrp2/mp3_128
```

### `mqtt.conf` (optional)

Wenn nicht vorhanden, startet die App trotzdem (HTTP/Web-only).

```text
broker=tcp://192.168.188.62:1883
user=your_user
pass=your_password
```

## 6) Starten und testen

```bash
cd ~/go-radio-streamer
./radio-streamer
```

In zweitem Terminal:

```bash
curl -sS http://localhost:8080/api/status
curl -sS http://localhost:8080/api/stations
```

Web UI:
- `http://<pi-ip>:8080/`

## 7) Autostart via systemd (empfohlen)

Service-Datei anlegen:

```bash
sudo tee /etc/systemd/system/go-radio-streamer.service >/dev/null <<'EOF'
[Unit]
Description=Go Radio Streamer
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pi
WorkingDirectory=/home/pi/go-radio-streamer
ExecStart=/home/pi/go-radio-streamer/radio-streamer
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
```

Aktivieren:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now go-radio-streamer
sudo systemctl status go-radio-streamer
```

Logs:

```bash
journalctl -u go-radio-streamer -f
```

## 8) Netzwerk / Multicast Hinweise

- Stream sendet nach `239.0.0.1:5004` (UDP)
- SAP sendet nach `239.255.255.255:9875`
- WLAN/AP/Switch muss Multicast zulassen (IGMP Snooping korrekt konfigurieren)
- Bei Client-Problemen zuerst lokal prüfen:

```bash
sudo tcpdump -ni any udp and port 5004
```

## 9) Typische Probleme

### A) Build scheitert wegen Go-Version
- Ursache: Pi hat altes Go aus `apt`
- Lösung: Go wie oben manuell auf `1.26` aktualisieren

### B) `ffmpeg` nicht gefunden
- Lösung:

```bash
sudo apt install -y ffmpeg
ffmpeg -version
```

### C) Server läuft, aber kein Audio beim Empfänger
- Prüfen:
  1. `curl -sS http://localhost:8080/api/stream.sdp`
  2. Empfänger erwartet `L24/48000/2`, PT `97`
  3. Multicast im Netzwerk erlaubt

### D) Kein MQTT verfügbar
- Kein Blocker: HTTP/Web funktioniert ohne MQTT
- Für MQTT-Steuerung `mqtt.conf` korrekt setzen

## 10) Security-Hinweise für Pi

- `mqtt.conf` enthält Credentials und bleibt lokal (durch `.gitignore` ausgeschlossen)
- API hat aktuell keine Authentifizierung → nur im vertrauenswürdigen Netz betreiben
- Optional: Zugriff auf Port `8080` per Firewall/IP einschränken
