package config

import (
	"os"
	"reflect"
	"testing"
)
func TestLoadStations(t *testing.T) {
	// Create a temporary stations file
	content := []byte(`1. SRF-1 http://stream.srg-ssr.ch/drs1/mp3_128.m3u
2. SRF-2 http://stream.srg-ssr.ch/drs2/mp3_128.m3u
`)
	tmpfile, err := os.CreateTemp("", "stations.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	expectedStations := []Station{
		{Name: "SRF-1", URL: "http://stream.srg-ssr.ch/drs1/mp3_128.m3u"},
		{Name: "SRF-2", URL: "http://stream.srg-ssr.ch/drs2/mp3_128.m3u"},
	}

	stations, err := LoadStations(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadStations failed: %v", err)
	}

	if !reflect.DeepEqual(stations, expectedStations) {
		t.Errorf("LoadStations returned %v, expected %v", stations, expectedStations)
	}
}

func TestLoadStreamerConfig(t *testing.T) {
	content := []byte("multicast_address=239.69.250.171:5004\n")
	tmpfile, err := os.CreateTemp("", "streamer.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadStreamerConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadStreamerConfig failed: %v", err)
	}
	if cfg.MulticastAddress != "239.69.250.171:5004" {
		t.Errorf("expected MulticastAddress 239.69.250.171:5004, got %q", cfg.MulticastAddress)
	}
}

func TestLoadStreamerConfig_Defaults(t *testing.T) {
	// Empty file should use defaults
	tmpfile, err := os.CreateTemp("", "streamer.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	cfg, err := LoadStreamerConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadStreamerConfig failed: %v", err)
	}
	if cfg.MulticastAddress != DefaultMulticastAddress {
		t.Errorf("expected default MulticastAddress %q, got %q", DefaultMulticastAddress, cfg.MulticastAddress)
	}
}
