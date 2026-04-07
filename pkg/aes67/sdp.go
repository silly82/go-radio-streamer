package aes67

import (
	"fmt"
	"net"
)

// DefaultPTPRefClock is the full ts-refclk value for a placeholder PTP clock.
// It includes the "ptp=" prefix as required by BuildSDP.
// For production use, replace the clock ID with the actual PTP grandmaster clock ID.
const DefaultPTPRefClock = "ptp=IEEE1588-2008:00-00-00-00-00-00-00-00:0"
const DefaultPtimeMs = 40

// GetLocalMAC returns the MAC address of the first active non-loopback network
// interface as a hyphen-separated uppercase string (e.g. "AA-BB-CC-DD-EE-FF").
// Falls back to "00-00-00-00-00-00" if no suitable interface is found.
func GetLocalMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "00-00-00-00-00-00"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) == 6 {
			hw := iface.HardwareAddr
			return fmt.Sprintf("%02X-%02X-%02X-%02X-%02X-%02X",
				hw[0], hw[1], hw[2], hw[3], hw[4], hw[5])
		}
	}
	return "00-00-00-00-00-00"
}

// LocalRefClock returns the full ts-refclk value using the local machine's MAC
// address (e.g. "localmac=AA-BB-CC-DD-EE-FF").  Use this when no external PTP
// grandmaster clock is available on the network.
func LocalRefClock() string {
	return "localmac=" + GetLocalMAC()
}

// GetLocalIP returns the first non-loopback, non-unspecified IPv4 address of
// the host. It falls back to "0.0.0.0" when no suitable address is found.
func GetLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "0.0.0.0"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "0.0.0.0"
}

// BuildSDP builds an AES67-compatible SDP description.
//
// tsRefClk is the full value for the a=ts-refclk attribute, e.g.:
//   - "ptp=IEEE1588-2008:AA-BB-CC-DD-EE-FF-00-00:0"  (external PTP grandmaster)
//   - "localmac=AA-BB-CC-DD-EE-FF"                    (local system clock)
//
// When tsRefClk is empty, LocalRefClock() is used as the default so that
// receivers without PTP synchronisation can still play the stream.
func BuildSDP(sessionName, multicastIP, originIP string, port, payloadType int, tsRefClk string, ptimeMs int) string {
	if sessionName == "" {
		sessionName = "gostreamer"
	}
	if multicastIP == "" {
		multicastIP = "239.0.0.1"
	}
	if originIP == "" {
		originIP = GetLocalIP()
	}
	if port == 0 {
		port = 5004
	}
	if payloadType == 0 {
		payloadType = 97
	}
	if tsRefClk == "" {
		tsRefClk = LocalRefClock()
	}
	if ptimeMs <= 0 {
		ptimeMs = DefaultPtimeMs
	}

	return fmt.Sprintf("v=0\r\n"+
		"o=- 0 0 IN IP4 %s\r\n"+
		"s=%s\r\n"+
		"c=IN IP4 %s/32\r\n"+
		"t=0 0\r\n"+
		"m=audio %d RTP/AVP %d\r\n"+
		"a=rtpmap:%d L24/48000/2\r\n"+
		"a=ptime:%d\r\n"+
		"a=ts-refclk:%s\r\n"+
		"a=mediaclk:direct=0\r\n",
		originIP,
		sessionName,
		multicastIP,
		port,
		payloadType,
		payloadType,
		ptimeMs,
		tsRefClk,
	)
}
