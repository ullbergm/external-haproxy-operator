package haproxyclient

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GetConfigVersion returns the current HAProxy configuration version
func (c *Client) GetConfigVersion() (int64, error) {
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/version", c.config.BaseURL))
	if err != nil {
		return 0, err
	}

	body := strings.TrimSpace(string(resp.Body()))
	version, err := strconv.ParseInt(body, 10, 64)
	if err != nil {
		// Try parsing as JSON object with "_version"
		var v struct {
			Version int64 `json:"_version"`
		}
		if err2 := json.Unmarshal(resp.Body(), &v); err2 == nil {
			return v.Version, nil
		}
		return 0, fmt.Errorf("failed to parse version from body: %q: %w", body, err)
	}
	return version, nil
}
