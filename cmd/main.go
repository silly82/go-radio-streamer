package main

import (
	"errors"
	"fmt"
	"go-radio-streamer/internal/api"
	"go-radio-streamer/internal/config"
	"go-radio-streamer/internal/mqtt"
	"go-radio-streamer/internal/streamer"
	"go-radio-streamer/internal/web"
	"log"
	"net/http"
	"os"
)

func main() {
	stations, err := config.LoadStations("stations.txt")
	if err != nil {
		log.Fatalf("failed to load stations: %v", err)
	}

	mqttConfig, err := config.LoadMQTTConfig("mqtt.conf")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Warning: mqtt.conf not found, MQTT disabled (HTTP/Web UI still available)")
		} else {
			log.Printf("Warning: failed to load MQTT config, MQTT disabled: %v", err)
		}
		mqttConfig = nil
	}

	streamerConfig, err := config.LoadStreamerConfig("streamer.conf")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Warning: streamer.conf not found, using default multicast address %s", config.DefaultMulticastAddress)
		} else {
			log.Printf("Warning: failed to load streamer config, using defaults: %v", err)
		}
		streamerConfig = &config.StreamerConfig{MulticastAddress: config.DefaultMulticastAddress}
	}

	// Create streamer with nil publish func initially
	s, err := streamer.NewStreamer(nil)
	if err != nil {
		log.Fatalf("failed to create streamer: %v", err)
	}

	if mqttConfig != nil {
		// Setup MQTT client in streamer
		err = s.SetupMQTTClient(mqttConfig.Broker, mqttConfig.User, mqttConfig.Password)
		if err != nil {
			log.Printf("Warning: MQTT setup failed: %v", err)
		}

		// Setup MQTT handler (legacy, kept for API compatibility)
		mqttHandler := mqtt.NewHandler(s, stations, streamerConfig.MulticastAddress)
		mqttHandler.SetupMQTT(mqttConfig.Broker, mqttConfig.User, mqttConfig.Password)

		// Set publish func
		s.SetPublishFunc(mqttHandler.Publish)
	}

	// Setup router
	router := api.NewRouter(s, stations, streamerConfig.MulticastAddress)
	web.SetupRoutes(router.Router)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
