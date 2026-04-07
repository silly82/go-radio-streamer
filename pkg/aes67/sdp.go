package aes67

import (
	"fmt"
	"net"
)

const DefaultPTPRefClock = "IEEE1588-2008:00-00-00-00-00-00-00-00:0"
const DefaultPtimeMs = 40

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

func BuildSDP(sessionName, multicastIP, originIP string, port, payloadType int, ptpRefClock string, ptimeMs int) string {
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
	if ptpRefClock == "" {
		ptpRefClock = DefaultPTPRefClock
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
		"a=ts-refclk:ptp=%s\r\n"+
		"a=mediaclk:direct=0\r\n",
		originIP,
		sessionName,
		multicastIP,
		port,
		payloadType,
		payloadType,
		ptimeMs,
		ptpRefClock,
	)
}
