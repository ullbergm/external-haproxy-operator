package haproxyclient

import (
	"testing"
	"time"
)

func TestConfigFields(t *testing.T) {
	cfg := Config{
		BaseURL:  "http://localhost",
		Username: "user",
		Password: "pass",
		Timeout:  10 * time.Second,
	}

	if cfg.BaseURL != "http://localhost" {
		t.Errorf("expected BaseURL 'http://localhost', got '%s'", cfg.BaseURL)
	}
	if cfg.Username != "user" {
		t.Errorf("expected Username 'user', got '%s'", cfg.Username)
	}
	if cfg.Password != "pass" {
		t.Errorf("expected Password 'pass', got '%s'", cfg.Password)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", cfg.Timeout)
	}
}
