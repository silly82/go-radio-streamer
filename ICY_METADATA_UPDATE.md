# ICY Metadata Integration - Manual Update Guide

## Files Created/Modified:

### 1. NEW: internal/streamer/metadata.go
✅ Already created with ICY parsing functions

### 2. MODIFY: internal/streamer/streamer.go

#### Add to imports (line ~5):
```go
	"encoding/json"
```

#### Add Metadata struct (after constants, ~line 29):
```go
// Metadata holds song information from ICY tags
type Metadata struct {
	Artist    string    `json:"artist"`
	Track     string    `json:"track"`
	Updated   time.Time `json:"updated"`
}
```

#### Add to Streamer struct (line ~43):
```go
	metadata       Metadata
```

#### Update publishHeartbeat() function (~line 130):
Replace the existing function with:
```go
// publishHeartbeat publishes heartbeat with status, station and metadata
func (s *Streamer) publishHeartbeat() {
	if s.mqttClient == nil || !s.mqttClient.IsConnected() {
		return
	}

	// Build heartbeat payload with metadata
	var status string
	if s.running && s.currentStation != "" {
		// Include metadata if available
		if s.metadata.Artist != "" || s.metadata.Track != "" {
			status = fmt.Sprintf(
				"{\"status\":\"streaming\",\"station\":\"%s\",\"artist\":\"%s\",\"track\":\"%s\",\"timestamp\":%d}",
				s.currentStation, s.metadata.Artist, s.metadata.Track, time.Now().Unix())
		} else {
			status = fmt.Sprintf(
				"{\"status\":\"streaming\",\"station\":\"%s\",\"timestamp\":%d}",
				s.currentStation, time.Now().Unix())
		}
	} else {
		status = fmt.Sprintf("{\"status\":\"stopped\",\"timestamp\":%d}", time.Now().Unix())
	}

	// Publish to heartbeat topic
	token := s.mqttClient.Publish("gostreamer/heartbeat", 0, false, status)
	token.Wait()

	log.Printf("Heartbeat published: %s", status)
}
```

#### Call metadata update in Start() (~line 380):
After: `s.startHeartbeat()`
Add: `s.updateMetadataAsync(streamURL)`

## How it works:

1. **When streaming starts**: 
   - Calls `s.updateMetadataAsync(streamURL)` 
   - ICY metadata fetched in background goroutine

2. **Every 30 seconds (heartbeat)**:
   - Includes artist and track if available
   - Publishes to `gostreamer/heartbeat` topic

3. **MQTT Payload Example**:
```json
{
  "status": "streaming",
  "station": "SRF-1",
  "artist": "Artist Name",
  "track": "Song Title",
  "timestamp": 1743468942
}
```

## Testing:

```bash
# Subscribe to heartbeat
mosquitto_sub -h BROKER -u USER -P PASS -t "gostreamer/#"

# Watch metadata updates
# Should see: {"status":"streaming",...,"artist":"...","track":"..."}
```

## Build:
```bash
CGO_ENABLED=0 go build -o radio-streamer ./cmd
```
