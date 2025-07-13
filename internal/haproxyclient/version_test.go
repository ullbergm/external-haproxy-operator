package haproxyclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestHAProxyClientWithServer(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	t.Helper()
	ts := httptest.NewServer(handler)
	config := HAProxyConfig{
		BaseURL: ts.URL,
		Timeout: 5 * time.Second,
	}
	client := NewHAProxyClient(config).(*Client)
	return client, ts.Close
}

func TestGetConfigVersion_PlainInt(t *testing.T) {
	client, closeFn := newTestHAProxyClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "42")
	})
	defer closeFn()
	version, err := client.GetConfigVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != 42 {
		t.Errorf("expected 42, got %d", version)
	}
}

func TestGetConfigVersion_JSON(t *testing.T) {
	client, closeFn := newTestHAProxyClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"_version":99}`)
	})
	defer closeFn()
	version, err := client.GetConfigVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != 99 {
		t.Errorf("expected 99, got %d", version)
	}
}

func TestGetConfigVersion_Invalid(t *testing.T) {
	client, closeFn := newTestHAProxyClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "notanumber")
	})
	defer closeFn()
	_, err := client.GetConfigVersion()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "failed to parse version") {
		t.Errorf("unexpected error: %v", err)
	}
}
