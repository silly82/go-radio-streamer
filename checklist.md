# Final Checklist (Kurzfassung)

Stand: 4. April 2026

## ✅ Wichtigste erreichte Punkte

- [x] RTP-Multicast-Streaming läuft auf `239.0.0.1:5004`
- [x] Audio-Format auf `L24/48000/2` umgestellt (AES67-kompatibler)
- [x] RTP Payload Type auf `97` vereinheitlicht
- [x] REST-API vorhanden: `GET /api/stations`, `GET /api/status`, `GET /api/stream.sdp`, `POST /api/play`, `POST /api/stop`
- [x] Web-UI zeigt aktiven Sender + Metadaten (`artist`, `track`)
- [x] ICY-Metadaten-Parsing repariert (korrekte `icy-metaint`-Verarbeitung)
- [x] MQTT-Steuerung aktiv über `gostreamer/play` und `gostreamer/stop`
- [x] MQTT-Status aktiv über `gostreamer/current` und `gostreamer/heartbeat`
- [x] SAP-Ankündigung aktiv auf `239.255.255.255:9875`
- [x] SDP-Generierung verfügbar über `GET /api/stream.sdp`
- [x] Shutdown-Race/`closed network connection`-Spam behoben
- [x] Stabiler Senderwechsel implementiert (sauberer Stop/Start im Backend)
- [x] Server-Start ohne `mqtt.conf` bzw. ohne MQTT-Broker möglich (HTTP/Web-only)

## ✅ Stabilitäts-Optimierungen (gegen Stottern)

- [x] FFmpeg-Ausgabe direkt als `pcm_s24be` (`s24be`)
- [x] Größere UDP-Write-Buffer gesetzt
- [x] Reconnect/Queue-Optionen für FFmpeg gesetzt
- [x] Sender-Queue + Prebuffer eingebaut
- [x] `ptime` auf Stabilität getrimmt (aktuell Default `40` ms)
- [x] Bei Buffer-Underrun wird Stille gesendet (kontinuierlicher RTP-Takt)

## ⚠️ Aktueller Zustand

- [x] Build/Test grundsätzlich erfolgreich (`go test ./...`, `go build`)
- [ ] Audio ist in manchen Setups noch nicht komplett frei von Haken

## 🔜 Nächste sinnvolle Schritte

- [ ] Optionales "Ultra-Stable" Profil ergänzen (z. B. `ptime=60`, größerer Prebuffer)
- [ ] Laufzeit-Profile per API/Config (`low|normal|high`) hinzufügen
- [ ] Receiver-spezifische Jitterbuffer-Empfehlungen in `docs/RECEIVER.md` dokumentieren
- [ ] 30–60 Minuten Dauertest mit Log-Auswertung (Underrun-Zähler, Paketverlust)

## 📌 Doku-Status

- [x] `README.md` auf aktuellen technischen Stand gebracht
- [x] Wichtigste Punkte in diese Datei (`checklist.md`) übertragen
