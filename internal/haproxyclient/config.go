package haproxyclient

import "time"

// Config holds the configuration for the HAProxy client
type Config struct {
	BaseURL  string
	Username string
	Password string
	Timeout  time.Duration
}
