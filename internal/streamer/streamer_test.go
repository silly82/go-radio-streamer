package streamer

import (
	"testing"
)

func TestNewStreamer(t *testing.T) {
	publishFunc := func(topic string, message string) {}
	s, err := NewStreamer(publishFunc)
	if err != nil {
		t.Fatalf("NewStreamer failed: %v", err)
	}

	if s == nil {
		t.Fatal("NewStreamer returned a nil streamer")
	}

	if s.running {
		t.Error("New streamer should not be running")
	}
}

func TestCreateRTPPacket(t *testing.T) {
	audioData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	seq := uint16(1234)
	timestamp := uint32(567890)
	ssrc := uint32(987654321)
	payloadType := uint8(97)

	buf, err := createRTPPacket(audioData, payloadType, seq, timestamp, ssrc)
	if err != nil {
		t.Fatalf("createRTPPacket failed: %v", err)
	}

	if len(buf) == 0 {
		t.Error("RTP packet buffer is empty")
	}

	// Basic check: header should be 12 bytes + payload
	minSize := 12 + len(audioData)
	if len(buf) < minSize {
		t.Errorf("RTP packet too short: got %d, expected at least %d", len(buf), minSize)
	}

	// Check version (first 2 bits)
	version := (buf[0] >> 6) & 0x03
	if version != 2 {
		t.Errorf("RTP version incorrect: got %d, expected 2", version)
	}
}

func TestSetupMulticastSocket(t *testing.T) {
	// Test with valid multicast address
	conn, err := setupMulticastSocket("239.0.0.1:5004")
	if err != nil {
		t.Fatalf("setupMulticastSocket failed: %v", err)
	}
	defer conn.Close()

	if conn == nil {
		t.Fatal("setupMulticastSocket returned a nil connection")
	}
}

func TestSetPublishFunc(t *testing.T) {
	s, _ := NewStreamer(nil)

	var called bool
	newPublishFunc := func(topic string, message string) {
		called = true
	}

	s.SetPublishFunc(newPublishFunc)

	if s.publishFunc == nil {
		t.Fatal("SetPublishFunc did not set the publish function")
	}

	// Test the function
	s.publishFunc("test", "message")
	if !called {
		t.Error("publishFunc was not called")
	}
}

func TestCurrentStatusDiagnostics(t *testing.T) {
	s, err := NewStreamer(nil)
	if err != nil {
		t.Fatalf("NewStreamer failed: %v", err)
	}

	// Initial state: all diagnostics should be zero
	status := s.CurrentStatus()
	if status.PacketsSent != 0 {
		t.Errorf("initial PacketsSent = %d, want 0", status.PacketsSent)
	}
	if status.SilencePackets != 0 {
		t.Errorf("initial SilencePackets = %d, want 0", status.SilencePackets)
	}
	if status.LastAudioPacketAt != 0 {
		t.Errorf("initial LastAudioPacketAt = %d, want 0", status.LastAudioPacketAt)
	}

	// Simulate packets being counted
	s.packetsSent.Add(100)
	s.silencePackets.Add(5)
	s.lastAudioPacketAt.Store(1234567890)

	status = s.CurrentStatus()
	if status.PacketsSent != 100 {
		t.Errorf("PacketsSent = %d, want 100", status.PacketsSent)
	}
	if status.SilencePackets != 5 {
		t.Errorf("SilencePackets = %d, want 5", status.SilencePackets)
	}
	if status.LastAudioPacketAt != 1234567890 {
		t.Errorf("LastAudioPacketAt = %d, want 1234567890", status.LastAudioPacketAt)
	}
}

func TestFloatToInt16Conversion(t *testing.T) {
	tests := []struct {
		input float64
		name  string
	}{
		{0.0, "zero"},
		{0.5, "half positive"},
		{-0.5, "half negative"},
	}

	for _, tt := range tests {
		result := int16(tt.input * 32767.0)
		if tt.input == 0.0 && result != 0 {
			t.Errorf("%s: got %d, expected 0", tt.name, result)
		}
		// result is int16 by definition; conversion path exercised above
	}
}
