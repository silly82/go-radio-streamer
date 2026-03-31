package streamer

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// parseICYMetadata extracts artist and track from ICY StreamTitle format
// Format: "Artist - Track" or just "Track"
func parseICYMetadata(streamTitle string) (artist, track string) {
	if strings.Contains(streamTitle, " - ") {
		parts := strings.SplitN(streamTitle, " - ", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", streamTitle
}

// getICYMetadata fetches metadata from HTTP stream headers
// This implements ICY Metadata protocol for radio streams
func getICYMetadata(streamURL string) Metadata {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		log.Printf("Error creating ICY metadata request: %v", err)
		return Metadata{}
	}

	// Request ICY metadata
	req.Header.Set("Icy-MetaData", "1")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching ICY metadata: %v", err)
		return Metadata{}
	}
	defer resp.Body.Close()

	// Get ICY-MetaInt header to know metadata block size
	metaInt := resp.Header.Get("Icy-MetaInt")
	if metaInt == "" {
		return Metadata{}
	}

	var icyMetaInt int
	fmt.Sscanf(metaInt, "%d", &icyMetaInt)

	if icyMetaInt <= 0 {
		return Metadata{}
	}

	// Skip audio bytes until the first metadata interval.
	buffer := make([]byte, icyMetaInt)
	_, err = io.ReadFull(resp.Body, buffer)
	if err != nil {
		log.Printf("Error reading ICY audio interval: %v", err)
		return Metadata{}
	}

	// Next byte contains the metadata block length in 16-byte units.
	lengthByte := make([]byte, 1)
	_, err = io.ReadFull(resp.Body, lengthByte)
	if err != nil {
		log.Printf("Error reading ICY metadata length: %v", err)
		return Metadata{}
	}

	metadataLength := int(lengthByte[0]) * 16
	if metadataLength <= 0 {
		return Metadata{}
	}

	metadataBlock := make([]byte, metadataLength)
	_, err = io.ReadFull(resp.Body, metadataBlock)
	if err != nil {
		log.Printf("Error reading ICY metadata block: %v", err)
		return Metadata{}
	}

	// Parse StreamTitle from metadata
	// Format: StreamTitle='Artist - Track';StreamUrl='';
	metadataStr := strings.TrimRight(string(metadataBlock), "\x00")
	streamTitle := ""

	// Extract StreamTitle value
	if idx := strings.Index(metadataStr, "StreamTitle='"); idx != -1 {
		startIdx := idx + len("StreamTitle='")
		endIdx := strings.Index(metadataStr[startIdx:], "'")
		if endIdx != -1 {
			streamTitle = metadataStr[startIdx : startIdx+endIdx]
		}
	}

	if streamTitle != "" {
		artist, track := parseICYMetadata(streamTitle)
		log.Printf("ICY Metadata: Artist='%s', Track='%s'", artist, track)
		return Metadata{
			Artist:  artist,
			Track:   track,
			Updated: time.Now(),
		}
	}

	return Metadata{}
}

// updateMetadataAsync fetches metadata asynchronously
func (s *Streamer) updateMetadataAsync(streamURL string) {
	go func() {
		metadata := getICYMetadata(streamURL)
		if metadata.Artist != "" || metadata.Track != "" {
			s.metadata = metadata
			log.Printf("Metadata updated: %s - %s", metadata.Artist, metadata.Track)
			s.PublishStatus(s.currentStation)
			// Publish updated metadata via heartbeat
			s.publishHeartbeat()
		}
	}()
}
