package aes67

import "fmt"

const DefaultPTPRefClock = "IEEE1588-2008:00-00-00-00-00-00-00-00:0"
const DefaultPtimeMs = 40

func BuildSDP(sessionName, multicastIP string, port, payloadType int, ptpRefClock string, ptimeMs int) string {
	if sessionName == "" {
		sessionName = "gostreamer"
	}
	if multicastIP == "" {
		multicastIP = "239.0.0.1"
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
		"o=- 0 0 IN IP4 0.0.0.0\r\n"+
		"s=%s\r\n"+
		"c=IN IP4 %s/32\r\n"+
		"t=0 0\r\n"+
		"m=audio %d RTP/AVP %d\r\n"+
		"a=rtpmap:%d L24/48000/2\r\n"+
		"a=ptime:%d\r\n"+
		"a=ts-refclk:ptp=%s\r\n"+
		"a=mediaclk:direct=0\r\n",
		sessionName,
		multicastIP,
		port,
		payloadType,
		payloadType,
		ptimeMs,
		ptpRefClock,
	)
}
