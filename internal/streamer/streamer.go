package streamer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go-radio-streamer/internal/config"
	"go-radio-streamer/pkg/aes67"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/hajimehoshi/go-mp3"
	"github.com/pion/rtp"
	"golang.org/x/net/ipv4"
)

const (
	maxRetries = 3
	retryDelay = 5 * time.Second
)

// Metadata holds song information from ICY tags
type Metadata struct {
	Artist  string    `json:"artist"`
	Track   string    `json:"track"`
	Updated time.Time `json:"updated"`
}

type Status struct {
	Running     bool   `json:"running"`
	Station     string `json:"station"`
	Artist      string `json:"artist"`
	Track       string `json:"track"`
	MetaUpdated int64  `json:"meta_updated"`
}

// Streamer manages the audio stream.
type Streamer struct {
	station        config.Station
	stopCh         chan struct{}
	running        bool
	currentStation string
	publishFunc    func(topic string, message string)
	conn           *net.UDPConn // Keep connection open during streaming
	streamDone     chan struct{}
	heartbeatTick  *time.Ticker
	heartbeatStop  chan struct{}
	metadata       Metadata
	sapAnnouncer   *aes67.SAPAnnouncer
}

// NewStreamer creates a new Streamer.
func NewStreamer(publishFunc func(topic string, message string)) (*Streamer, error) {
	return &Streamer{
		stopCh:      make(chan struct{}),
		streamDone:  make(chan struct{}),
		publishFunc: publishFunc,
	}, nil
}

// SetPublishFunc sets the publish function for MQTT.
func (s *Streamer) SetPublishFunc(publishFunc func(topic string, message string)) {
	s.publishFunc = publishFunc
}

// SetupMQTTClient is kept for compatibility with the current startup flow.
func (s *Streamer) SetupMQTTClient(broker, username, password string) error {
	return nil
}

func (s *Streamer) CurrentStatus() Status {
	return Status{
		Running:     s.running,
		Station:     s.currentStation,
		Artist:      s.metadata.Artist,
		Track:       s.metadata.Track,
		MetaUpdated: s.metadata.Updated.Unix(),
	}
}

// startHeartbeat starts publishing heartbeat messages every 30 seconds
func (s *Streamer) startHeartbeat(streamURL string) {
	if s.heartbeatTick != nil {
		s.heartbeatTick.Stop()
	}

	s.heartbeatStop = make(chan struct{})
	s.heartbeatTick = time.NewTicker(30 * time.Second)

	go func() {
		for {
			select {
			case <-s.heartbeatTick.C:
				if streamURL != "" {
					s.updateMetadataAsync(streamURL)
				}
				s.publishHeartbeat()
			case <-s.heartbeatStop:
				return
			}
		}
	}()

	log.Println("Heartbeat started (30 second interval)")
}

// stopHeartbeat stops the heartbeat
func (s *Streamer) stopHeartbeat() {
	if s.heartbeatTick != nil {
		s.heartbeatTick.Stop()
	}
	if s.heartbeatStop != nil {
		close(s.heartbeatStop)
	}
}

// publishHeartbeat publishes heartbeat with status and current station
func (s *Streamer) publishHeartbeat() {
	if s.publishFunc == nil {
		return
	}

	// Build heartbeat payload
	var status string
	if s.running && s.currentStation != "" {
		status = fmt.Sprintf("{\"status\":\"streaming\",\"station\":\"%s\",\"timestamp\":%d,\"artist\":\"%s\",\"track\":\"%s\",\"meta_updated\":%d}",
			s.currentStation, time.Now().Unix(), s.metadata.Artist, s.metadata.Track, s.metadata.Updated.Unix())
	} else {
		status = fmt.Sprintf("{\"status\":\"stopped\",\"timestamp\":%d}", time.Now().Unix())
	}

	// Publish to heartbeat topic
	s.publishFunc("gostreamer/heartbeat", status)

	log.Printf("Heartbeat published: %s", status)
}

// mqttMessageHandler handles incoming MQTT messages
func (s *Streamer) mqttMessageHandler(client MQTT.Client, msg MQTT.Message) {
	topic := msg.Topic()
	payload := string(msg.Payload())

	log.Printf("MQTT received: %s = %s", topic, payload)

	switch topic {
	case "gostreamer/play":
		// Parse station number from payload
		var stationNum int
		_, err := fmt.Sscanf(payload, "%d", &stationNum)
		if err != nil {
			log.Printf("Invalid station number: %s", payload)
			return
		}
		log.Printf("MQTT command: Play station %d", stationNum)
		// Station info would need to be passed separately or stored
	case "gostreamer/stop":
		log.Printf("MQTT command: Stop streaming")
		s.Stop()
	default:
		log.Printf("Unknown MQTT topic: %s", topic)
	}
}

// PublishStatus publishes the current status to MQTT
func (s *Streamer) PublishStatus(status string) {
	if s.publishFunc != nil {
		payload := fmt.Sprintf("{\"status\":\"%s\",\"station\":\"%s\",\"artist\":\"%s\",\"track\":\"%s\",\"meta_updated\":%d}",
			status, s.currentStation, s.metadata.Artist, s.metadata.Track, s.metadata.Updated.Unix())

		s.publishFunc("gostreamer/current", payload)
		log.Printf("Published status: %s", status)
	}
}

// SetPublishFunc sets the publish function for MQTT.

// setupMulticastSocket sets up a UDP socket for multicast sending.
func setupMulticastSocket(multicastAddr string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve multicast address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP: %w", err)
	}

	if err := conn.SetWriteBuffer(1 << 20); err != nil {
		log.Printf("Warning: failed to set UDP write buffer: %v", err)
	}

	// Set multicast TTL
	p := ipv4.NewPacketConn(conn)
	if err := p.SetMulticastTTL(32); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set multicast TTL: %w", err)
	}

	return conn, nil
}

// createRTPPacket creates an RTP packet with the given payload type.
func createRTPPacket(audioData []byte, payloadType uint8, seq uint16, timestamp uint32, ssrc uint32) ([]byte, error) {
	header := &rtp.Header{
		Version:        2,
		Padding:        false,
		Extension:      false,
		Marker:         false,
		PayloadType:    payloadType,
		SequenceNumber: seq,
		Timestamp:      timestamp,
		SSRC:           ssrc,
	}

	packet := &rtp.Packet{
		Header:  *header,
		Payload: audioData,
	}

	buf, err := packet.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RTP packet: %w", err)
	}

	return buf, nil
}

// decodeAndResampleWithFFmpeg uses FFmpeg to decode MP3 and resample to 48kHz.
func decodeAndResampleWithFFmpeg(streamURL string) (io.ReadCloser, error) {
	// Use FFmpeg to decode MP3 and resample to 48kHz
	// Output format: s24be (signed 24-bit big-endian PCM)
	cmd := exec.Command(
		"ffmpeg",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-thread_queue_size", "4096",
		"-i", streamURL,
		"-ar", "48000", // resample to 48kHz
		"-ac", "2", // 2 channels (stereo)
		"-acodec", "pcm_s24be",
		"-f", "s24be", // output format: signed 24-bit big-endian
		"-hide_banner",       // suppress FFmpeg banner
		"-loglevel", "error", // only log errors
		"pipe:1", // output to stdout
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get FFmpeg stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	// Return a custom reader that also handles cmd cleanup
	return &ffmpegReader{cmd: cmd, reader: stdout}, nil
}

// ffmpegReader wraps FFmpeg process and its stdout
type ffmpegReader struct {
	cmd    *exec.Cmd
	reader io.ReadCloser
}

func (fr *ffmpegReader) Read(p []byte) (n int, err error) {
	return fr.reader.Read(p)
}

func (fr *ffmpegReader) Close() error {
	var closeErr error
	if fr.reader != nil {
		if err := fr.reader.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			closeErr = err
		}
	}

	if fr.cmd == nil {
		return closeErr
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- fr.cmd.Wait()
	}()

	select {
	case waitErr := <-waitCh:
		if waitErr != nil && closeErr == nil {
			closeErr = waitErr
		}
	case <-time.After(2 * time.Second):
		if fr.cmd.Process != nil {
			_ = fr.cmd.Process.Kill()
		}
		waitErr := <-waitCh
		if waitErr != nil && closeErr == nil {
			closeErr = waitErr
		}
	}

	return closeErr
}

// streamAudio handles the audio streaming loop.
func (s *Streamer) streamAudio(conn *net.UDPConn, streamURL string) {
	// Use FFmpeg to decode and resample
	audioReader, err := decodeAndResampleWithFFmpeg(streamURL)
	if err != nil {
		log.Printf("Failed to setup FFmpeg: %v", err)
		s.running = false
		close(s.streamDone)
		return
	}
	defer audioReader.Close()
	defer func() {
		s.running = false
		close(s.streamDone)
	}()

	seq := uint16(0)
	timestamp := uint32(0)
	ssrc := rand.Uint32()

	const (
		rtpPayloadType = 97
		channels       = 2
		samplesPerMs   = 48
		bytesPerL24    = 3
		ptimeMs        = 40
		bufferPackets  = 256
		prebufferPkts  = 25
	)

	samplesPerPacket := samplesPerMs * ptimeMs
	packetDuration := time.Duration(ptimeMs) * time.Millisecond
	totalSamples := samplesPerPacket * channels

	packetBytes := totalSamples * bytesPerL24
	pktCh := make(chan []byte, bufferPackets)
	readErrCh := make(chan error, 1)
	silencePayload := make([]byte, packetBytes)

	go func() {
		defer close(pktCh)
		readBuf := make([]byte, packetBytes)
		for {
			select {
			case <-s.stopCh:
				return
			default:
			}

			n, err := io.ReadFull(audioReader, readBuf)
			if err != nil {
				if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
					return
				}
				select {
				case readErrCh <- err:
				default:
				}
				return
			}
			if n != packetBytes {
				continue
			}

			pkt := make([]byte, packetBytes)
			copy(pkt, readBuf)
			select {
			case pktCh <- pkt:
			case <-s.stopCh:
				return
			}
		}
	}()

	pending := make([][]byte, 0, prebufferPkts)
	prebufferTimeout := time.NewTimer(3 * time.Second)
	defer prebufferTimeout.Stop()

prebufferLoop:
	for len(pending) < prebufferPkts {
		select {
		case <-s.stopCh:
			return
		case err := <-readErrCh:
			if err != nil {
				log.Printf("Audio reader error: %v", err)
			}
			return
		case pkt, ok := <-pktCh:
			if !ok {
				break prebufferLoop
			}
			pending = append(pending, pkt)
		case <-prebufferTimeout.C:
			break prebufferLoop
		}
	}

	log.Printf("RTP prebuffer ready: %d packets (%d ms)", len(pending), len(pending)*ptimeMs)

	ticker := time.NewTicker(packetDuration)
	defer ticker.Stop()
	underrunCount := 0

	for {
		select {
		case <-s.stopCh:
			return
		case err := <-readErrCh:
			if err != nil {
				log.Printf("Audio reader error: %v", err)
			}
			if len(pending) == 0 && pktCh == nil {
				return
			}
		case pkt, ok := <-pktCh:
			if ok {
				pending = append(pending, pkt)
			} else {
				pktCh = nil
			}
		case <-ticker.C:
			payload := silencePayload
			if len(pending) > 0 {
				payload = pending[0]
				pending = pending[1:]
				underrunCount = 0
			} else {
				underrunCount++
				if underrunCount == 1 || underrunCount%25 == 0 {
					log.Printf("RTP buffer underrun: sending silence (count=%d)", underrunCount)
				}
			}

			rtpBuf, err := createRTPPacket(payload, rtpPayloadType, seq, timestamp, ssrc)
			if err != nil {
				log.Printf("Failed to create RTP packet: %v", err)
				continue
			}

			_, err = conn.Write(rtpBuf)
			if err != nil {
				if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "closed network connection") {
					return
				}
				log.Printf("Failed to send RTP packet: %v", err)
			}

			seq++
			timestamp += uint32(samplesPerPacket)
		}
	}
}

// Start begins streaming the given station.
func (s *Streamer) Start(station config.Station, multicastAddress string) error {
	if s.running {
		if s.currentStation == station.Name {
			log.Printf("Station %s is already running", station.Name)
			return nil
		}

		log.Printf("Switching station from %s to %s", s.currentStation, station.Name)
		s.Stop()
		time.Sleep(200 * time.Millisecond)
	}

	s.station = station
	s.currentStation = station.Name
	log.Printf("Starting stream for %s", s.station.Name)

	// Setup multicast socket
	conn, err := setupMulticastSocket(multicastAddress)
	if err != nil {
		return fmt.Errorf("failed to setup multicast socket: %w", err)
	}

	// Store connection for later cleanup
	s.conn = conn

	// Get stream URL
	streamURL, err := getStreamURLFromM3U(s.station.URL)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to get stream URL: %w", err)
	}

	// Fetch ICY metadata for artist/track in background
	s.updateMetadataAsync(streamURL)

	// Start streaming in a goroutine
	s.running = true
	s.streamDone = make(chan struct{})
	s.startHeartbeat(streamURL) // Start heartbeat when streaming begins
	go s.streamAudio(conn, streamURL)

	multicastIP, portStr, splitErr := net.SplitHostPort(multicastAddress)
	if splitErr == nil {
		port, portErr := strconv.Atoi(portStr)
		if portErr == nil {
			sdp := aes67.BuildSDP("gostreamer", multicastIP, "", port, 97, aes67.DefaultPTPRefClock, aes67.DefaultPtimeMs)
			announcer, announceErr := aes67.NewSAPAnnouncer(aes67.DefaultSAPAddress, 30*time.Second, sdp)
			if announceErr != nil {
				log.Printf("Failed to start SAP announcer: %v", announceErr)
			} else {
				s.sapAnnouncer = announcer
				s.sapAnnouncer.Start()
				log.Printf("SAP announcement started for %s", multicastAddress)
			}
		}
	}

	log.Printf("Streaming %s to %s", s.station.Name, multicastAddress)
	s.PublishStatus(s.currentStation)
	s.publishHeartbeat() // Publish initial heartbeat
	return nil
}

// Stop halts the current stream.
func (s *Streamer) Stop() {
	if !s.running {
		log.Println("Stream is not running, nothing to stop")
		return
	}
	log.Println("Stopping stream, running flag was:", s.running)
	s.running = false
	log.Println("Running flag set to false:", s.running)
	close(s.stopCh)

	// Stop heartbeat
	s.stopHeartbeat()
	if s.sapAnnouncer != nil {
		s.sapAnnouncer.Stop()
		s.sapAnnouncer = nil
	}

	select {
	case <-s.streamDone:
	case <-time.After(2 * time.Second):
		log.Println("Timed out waiting for stream loop to stop")
	}

	// Close connection
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	s.currentStation = ""
	s.metadata = Metadata{}
	s.stopCh = make(chan struct{}) // Re-create for next start
	s.PublishStatus("stopped")
	s.publishHeartbeat() // Publish final heartbeat
	log.Println("Stream stopped successfully")
}

func getStreamURLFromM3U(m3uURL string) (string, error) {
	resp, err := http.Get(m3uURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "http") {
			return line, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("no stream URL found in M3U file")
}

// stream handles the audio fetching, decoding, and sending.
func (s *Streamer) stream() {
	var err error
	var streamURL string

	if strings.HasSuffix(s.station.URL, ".m3u") {
		streamURL, err = getStreamURLFromM3U(s.station.URL)
		if err != nil {
			log.Printf("failed to get stream URL from M3U: %v", err)
			s.running = false
			return
		}
	} else {
		streamURL = s.station.URL
	}

	for retries := 0; retries < maxRetries; retries++ {
		select {
		case <-s.stopCh:
			return
		default:
		}

		log.Printf("Connecting to stream: %s", streamURL)
		resp, err := http.Get(streamURL)
		if err != nil {
			log.Printf("failed to connect to stream: %v. Retrying in %v...", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		err = s.handleStream(resp)
		if err != nil {
			if err == io.EOF {
				log.Println("Stream ended. Reconnecting...")
				continue
			}
			log.Printf("error handling stream: %v. Retrying in %v...", err, retryDelay)
			time.Sleep(retryDelay)
		}
	}
	log.Printf("failed to stream from %s after %d retries", streamURL, maxRetries)
	s.running = false
}

func (s *Streamer) handleStream(resp *http.Response) error {
	defer resp.Body.Close()
	decoder, err := mp3.NewDecoder(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create mp3 decoder: %w", err)
	}

	if decoder.SampleRate() != 48000 {
		log.Printf("Warning: MP3 sample rate (%d) does not match AES67 sample rate (48000). Audio may be distorted.", decoder.SampleRate())
	}

	buf := make([]byte, 1500) // A common MTU size
	for {
		select {
		case <-s.stopCh:
			log.Println("Stopping stream loop")
			return nil
		default:
			n, err := decoder.Read(buf)
			if err != nil {
				return err // This will be handled by the retry loop in stream()
			}
			if n > 0 {
				// Placeholder: _, err := s.aes67Sender.Write(buf[:n])
				log.Printf("Would send %d bytes via AES67", n)
				// if err != nil {
				// 	return fmt.Errorf("failed to send aes67 packet: %w", err)
				// }
			}
		}
	}
}
