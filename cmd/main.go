package main

import (
	"fmt"
	"go-radio-streamer/internal/api"
	"go-radio-streamer/internal/config"
	"go-radio-streamer/internal/mqtt"
	"go-radio-streamer/internal/streamer"
	"go-radio-streamer/internal/web"
	"log"
	"net/http"
)

func main() {
	stations, err := config.LoadStations("stations.txt")
	if err != nil {
		log.Fatalf("failed to load stations: %v", err)
	}

	mqttConfig, err := config.LoadMQTTConfig("mqtt.conf")
	if err != nil {
		log.Fatalf("failed to load MQTT config: %v", err)
	}

	// Create streamer with nil publish func initially
	s, err := streamer.NewStreamer(nil)
	if err != nil {
		log.Fatalf("failed to create streamer: %v", err)
	}

	// Setup MQTT client in streamer
	err = s.SetupMQTTClient(mqttConfig.Broker, mqttConfig.User, mqttConfig.Password)
	if err != nil {
		log.Printf("Warning: MQTT setup failed: %v", err)
	}

	// Setup MQTT handler (legacy, kept for API compatibility)
	mqttHandler := mqtt.NewHandler(s, stations)
	mqttHandler.SetupMQTT(mqttConfig.Broker, mqttConfig.User, mqttConfig.Password)

	// Set publish func
	s.SetPublishFunc(mqttHandler.Publish)

	// Setup router
	router := api.NewRouter(s, stations)
	web.SetupRoutes(router.Router)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
