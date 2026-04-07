package aes67

import (
	"net"
	"strings"
	"testing"
)

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	if ip == "" {
		t.Fatal("GetLocalIP returned empty string")
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		t.Fatalf("GetLocalIP returned invalid IP: %q", ip)
	}
	if parsed.IsLoopback() {
		t.Errorf("GetLocalIP returned loopback address: %q", ip)
	}
	// Must be IPv4
	if parsed.To4() == nil {
		t.Errorf("GetLocalIP returned non-IPv4 address: %q", ip)
	}
}

func TestBuildSDP_ExplicitOriginIP(t *testing.T) {
	sdp := BuildSDP("test", "239.1.2.3", "192.168.1.42", 5004, 97, DefaultPTPRefClock, 40)
	if !strings.Contains(sdp, "o=- 0 0 IN IP4 192.168.1.42") {
		t.Errorf("expected origin IP 192.168.1.42 in SDP, got:\n%s", sdp)
	}
}

func TestBuildSDP_AutoOriginIP(t *testing.T) {
	sdp := BuildSDP("test", "239.1.2.3", "", 5004, 97, DefaultPTPRefClock, 40)
	if strings.Contains(sdp, "o=- 0 0 IN IP4 0.0.0.0") {
		// 0.0.0.0 is only acceptable if there really is no non-loopback interface,
		// which is unusual in a real test environment.
		t.Logf("Warning: no non-loopback interface found, origin IP is 0.0.0.0")
	}
	// Either way the line must be present and well-formed.
	if !strings.Contains(sdp, "o=- 0 0 IN IP4 ") {
		t.Errorf("SDP missing origin line, got:\n%s", sdp)
	}
}

func TestBuildSDP_Defaults(t *testing.T) {
	// When tsRefClk is empty, BuildSDP should fall back to localmac.
	sdp := BuildSDP("", "", "10.0.0.1", 0, 0, "", 0)
	expected := []string{
		"s=gostreamer",
		"c=IN IP4 239.0.0.1/32",
		"m=audio 5004 RTP/AVP 97",
		"a=rtpmap:97 L24/48000/2",
		"a=ptime:40",
		"a=maxptime:40",
		"a=sendonly",
	}
	for _, e := range expected {
		if !strings.Contains(sdp, e) {
			t.Errorf("expected %q in SDP, got:\n%s", e, sdp)
		}
	}
	// Verify the localmac reference clock line has a properly-formatted MAC.
	refClk := LocalRefClock()
	if !strings.Contains(sdp, "a=ts-refclk:"+refClk) {
		t.Errorf("expected 'a=ts-refclk:%s' in SDP, got:\n%s", refClk, sdp)
	}
}

func TestBuildSDP_SendonlyAndMaxptime(t *testing.T) {
	sdp := BuildSDP("test", "239.1.2.3", "10.0.0.1", 5004, 97, DefaultPTPRefClock, 1)
	if !strings.Contains(sdp, "a=sendonly") {
		t.Errorf("SDP must contain 'a=sendonly' for AES67 compliance, got:\n%s", sdp)
	}
	if !strings.Contains(sdp, "a=maxptime:1") {
		t.Errorf("SDP must contain 'a=maxptime:1' matching ptime, got:\n%s", sdp)
	}
	if !strings.Contains(sdp, "a=ptime:1") {
		t.Errorf("SDP must contain 'a=ptime:1', got:\n%s", sdp)
	}
}

func TestBuildSDP_ExplicitPTPRefClock(t *testing.T) {
	sdp := BuildSDP("test", "239.1.2.3", "10.0.0.1", 5004, 97, DefaultPTPRefClock, 40)
	want := "a=ts-refclk:" + DefaultPTPRefClock
	if !strings.Contains(sdp, want) {
		t.Errorf("expected %q in SDP, got:\n%s", want, sdp)
	}
}

func TestBuildSDP_LocalRefClock(t *testing.T) {
	refClk := LocalRefClock()
	if !strings.HasPrefix(refClk, "localmac=") {
		t.Errorf("LocalRefClock() should start with 'localmac=', got %q", refClk)
	}
	sdp := BuildSDP("test", "239.1.2.3", "10.0.0.1", 5004, 97, refClk, 40)
	want := "a=ts-refclk:" + refClk
	if !strings.Contains(sdp, want) {
		t.Errorf("expected %q in SDP, got:\n%s", want, sdp)
	}
}

func TestGetLocalMAC(t *testing.T) {
	mac := GetLocalMAC()
	if mac == "" {
		t.Fatal("GetLocalMAC returned empty string")
	}
	// Should be either "00-00-00-00-00-00" (fallback) or a valid MAC
	parts := strings.Split(mac, "-")
	if len(parts) != 6 {
		t.Errorf("GetLocalMAC returned unexpected format: %q", mac)
	}
}
