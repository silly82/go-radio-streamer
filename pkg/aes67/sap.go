package aes67

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"time"
)

const DefaultSAPAddress = "239.255.255.255:9875"

type SAPAnnouncer struct {
	conn      *net.UDPConn
	target    *net.UDPAddr
	sourceIP  net.IP
	interval  time.Duration
	ticker    *time.Ticker
	stopCh    chan struct{}
	messageID uint16
	sdp       string
	running   bool
}

func NewSAPAnnouncer(targetAddress string, interval time.Duration, sdp string) (*SAPAnnouncer, error) {
	if targetAddress == "" {
		targetAddress = DefaultSAPAddress
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	target, err := net.ResolveUDPAddr("udp4", targetAddress)
	if err != nil {
		return nil, fmt.Errorf("resolve SAP target: %w", err)
	}

	conn, err := net.DialUDP("udp4", nil, target)
	if err != nil {
		return nil, fmt.Errorf("dial SAP target: %w", err)
	}

	localAddr, _ := conn.LocalAddr().(*net.UDPAddr)
	sourceIP := net.IPv4zero
	if localAddr != nil && localAddr.IP != nil {
		sourceIP = localAddr.IP.To4()
		if sourceIP == nil {
			sourceIP = net.IPv4zero
		}
	}

	return &SAPAnnouncer{
		conn:      conn,
		target:    target,
		sourceIP:  sourceIP,
		interval:  interval,
		stopCh:    make(chan struct{}),
		messageID: uint16(rand.Intn(65535)),
		sdp:       sdp,
	}, nil
}

func (a *SAPAnnouncer) Start() {
	if a == nil || a.running {
		return
	}

	a.running = true
	a.send(false)
	a.ticker = time.NewTicker(a.interval)

	go func() {
		for {
			select {
			case <-a.ticker.C:
				a.send(false)
			case <-a.stopCh:
				return
			}
		}
	}()
}

func (a *SAPAnnouncer) Stop() {
	if a == nil {
		return
	}

	if a.running {
		a.send(true)
	}

	if a.ticker != nil {
		a.ticker.Stop()
	}
	if a.stopCh != nil {
		close(a.stopCh)
		a.stopCh = nil
	}
	if a.conn != nil {
		a.conn.Close()
		a.conn = nil
	}
	a.running = false
}

func (a *SAPAnnouncer) send(deletion bool) {
	if a == nil || a.conn == nil {
		return
	}

	packet := a.buildPacket(deletion)
	_, _ = a.conn.Write(packet)
}

func (a *SAPAnnouncer) buildPacket(deletion bool) []byte {
	payloadType := []byte("application/sdp")

	header := byte(0x20)
	if deletion {
		header |= 0x04
	}

	sourceIP := a.sourceIP.To4()
	if sourceIP == nil {
		sourceIP = net.IPv4zero
	}

	packet := make([]byte, 0, 8+len(payloadType)+1+len(a.sdp))
	packet = append(packet, header)
	packet = append(packet, 0x00)

	msgID := make([]byte, 2)
	binary.BigEndian.PutUint16(msgID, a.messageID)
	packet = append(packet, msgID...)

	packet = append(packet, sourceIP...)
	packet = append(packet, payloadType...)
	packet = append(packet, 0x00)
	packet = append(packet, []byte(a.sdp)...)

	return packet
}
