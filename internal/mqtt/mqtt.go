package mqtt

import (
	"log"
	"strconv"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go-radio-streamer/internal/config"
	"go-radio-streamer/internal/streamer"
)

type Handler struct {
	streamer *streamer.Streamer
	stations []config.Station
	client   MQTT.Client
}

func NewHandler(s *streamer.Streamer, stations []config.Station) *Handler {
	return &Handler{
		streamer: s,
		stations: stations,
	}
}

func (h *Handler) SetupMQTT(broker string, username string, password string) {
	opts := MQTT.NewClientOptions().AddBroker(broker)
	opts.SetClientID("go-radio-streamer")
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetDefaultPublishHandler(h.messageHandler)

	h.client = MQTT.NewClient(opts)
	if token := h.client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("MQTT connection failed: %v", token.Error())
		return
	}

	// Subscribe to topics
	if token := h.client.Subscribe("radio/play", 0, nil); token.Wait() && token.Error() != nil {
		log.Printf("Failed to subscribe to radio/play: %v", token.Error())
	}
	if token := h.client.Subscribe("radio/stop", 0, nil); token.Wait() && token.Error() != nil {
		log.Printf("Failed to subscribe to radio/stop: %v", token.Error())
	}

	log.Printf("MQTT connected to %s", broker)
}

func (h *Handler) Publish(topic string, message string) {
	if h.client != nil && h.client.IsConnected() {
		token := h.client.Publish(topic, 0, false, message)
		token.Wait()
	}
}

func (h *Handler) messageHandler(client MQTT.Client, msg MQTT.Message) {
	topic := msg.Topic()
	payload := string(msg.Payload())

	log.Printf("MQTT received: %s = %s", topic, payload)

	switch topic {
	case "radio/play":
		num, err := strconv.Atoi(payload)
		if err != nil {
			log.Printf("Invalid station number: %s", payload)
			return
		}
		if num < 1 || num > len(h.stations) {
			log.Printf("Station number out of range: %d", num)
			return
		}
		station := h.stations[num-1]
		err = h.streamer.Start(station, "239.0.0.1:5004")
		if err != nil {
			log.Printf("Failed to start streaming: %v", err)
		} else {
			log.Printf("Started streaming station %d: %s", num, station.Name)
		}
	case "radio/stop":
		h.streamer.Stop()
		log.Println("Stopped streaming")
	default:
		log.Printf("Unknown topic: %s", topic)
	}
}