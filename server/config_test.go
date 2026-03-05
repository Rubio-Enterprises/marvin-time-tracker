package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("MARVIN_API_TOKEN", "test-token")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MarvinAPIToken != "test-token" {
		t.Errorf("expected MarvinAPIToken test-token, got %s", cfg.MarvinAPIToken)
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("expected default ListenAddr :8080, got %s", cfg.ListenAddr)
	}
	if cfg.StateFilePath != "./state.json" {
		t.Errorf("expected default StateFilePath ./state.json, got %s", cfg.StateFilePath)
	}
	if cfg.PollIntervalActive != 30*time.Second {
		t.Errorf("expected default PollIntervalActive 30s, got %v", cfg.PollIntervalActive)
	}
	if cfg.PollIntervalIdle != 5*time.Minute {
		t.Errorf("expected default PollIntervalIdle 5m, got %v", cfg.PollIntervalIdle)
	}
	if cfg.APNsBundleID != "com.strubio.MarvinTimeTracker" {
		t.Errorf("expected default APNsBundleID, got %s", cfg.APNsBundleID)
	}
}

func TestLoadConfigMissingToken(t *testing.T) {
	os.Unsetenv("MARVIN_API_TOKEN")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing MARVIN_API_TOKEN")
	}
}

func TestLoadConfigCustomValues(t *testing.T) {
	t.Setenv("MARVIN_API_TOKEN", "tok")
	t.Setenv("LISTEN_ADDR", ":9090")
	t.Setenv("POLL_INTERVAL_ACTIVE", "10s")
	t.Setenv("STATE_FILE_PATH", "/tmp/state.json")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ListenAddr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.ListenAddr)
	}
	if cfg.PollIntervalActive != 10*time.Second {
		t.Errorf("expected 10s, got %v", cfg.PollIntervalActive)
	}
	if cfg.StateFilePath != "/tmp/state.json" {
		t.Errorf("expected /tmp/state.json, got %s", cfg.StateFilePath)
	}
}
