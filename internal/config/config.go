package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Station struct {
	Name string
	URL  string
}

type MQTTConfig struct {
	Broker   string
	User     string
	Password string
}

type StreamerConfig struct {
	MulticastAddress string
}

func LoadStations(path string) ([]Station, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open station file: %w", err)
	}
	defer file.Close()

	var stations []Station
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue // Or return an error for malformed lines
		}

		// First part is "X.", remove the dot.
		numStr := strings.TrimSuffix(parts[0], ".")
		if _, err := strconv.Atoi(numStr); err != nil {
			continue // Or return an error
		}

		stations = append(stations, Station{
			Name: parts[1],
			URL:  parts[2],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading station file: %w", err)
	}

	return stations, nil
}

func LoadMQTTConfig(path string) (*MQTTConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open MQTT config file: %w", err)
	}
	defer file.Close()

	config := &MQTTConfig{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Or return an error
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "broker":
			config.Broker = value
		case "user":
			config.User = value
		case "pass":
			config.Password = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading MQTT config file: %w", err)
	}

	return config, nil
}

const DefaultMulticastAddress = "239.69.250.171:5004"

func LoadStreamerConfig(path string) (*StreamerConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open streamer config file: %w", err)
	}
	defer file.Close()

	cfg := &StreamerConfig{
		MulticastAddress: DefaultMulticastAddress,
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "multicast_address":
			cfg.MulticastAddress = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading streamer config file: %w", err)
	}

	return cfg, nil
}
