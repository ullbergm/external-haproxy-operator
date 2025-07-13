package haproxyclient

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/haproxytech/client-native/v6/models"
)

// GetBackend retrieves a backend by name
func (c *Client) GetBackend(name string) (*models.Backend, error) {
	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		SetResult(&models.Backend{}).
		SetQueryParam(queryKey, queryVal).
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s", c.config.BaseURL, name))
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	if resp.StatusCode() == 404 {
		return nil, nil
	}
	if resp.IsError() {
		return nil, ErrAPIResponse{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
			Operation:  "get backend",
		}
	}
	return resp.Result().(*models.Backend), nil
}

// EnsureBackend creates or updates a backend, ensuring it's managed by this controller
func (c *Client) EnsureBackend(backend *models.Backend) error {
	backend.Description = ManagedDescription

	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return fmt.Errorf("getting config version: %w", err)
	}

	existing, err := c.GetBackend(backend.Name)
	if err != nil {
		return fmt.Errorf("getting backend: %w", err)
	}

	var resp *resty.Response
	if existing == nil {
		// Create new backend
		resp, err = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(backend).
			SetQueryParam(queryKey, queryVal).
			Post(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends", c.config.BaseURL))
		c.transactionDirty = true
	} else if existing.Description != ManagedDescription {
		// Backend exists but is not managed by us
		return ErrNotManaged{
			ResourceType: "backend",
			ResourceName: backend.Name,
		}
	} else if !c.backendsEqual(existing, backend) {
		// Update existing backend
		resp, err = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(backend).
			SetQueryParam(queryKey, queryVal).
			Put(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s", c.config.BaseURL, backend.Name))
		c.transactionDirty = true
	} else {
		// Backend is already in desired state
		return nil
	}

	if err != nil {
		return fmt.Errorf("api request: %w", err)
	}
	if resp != nil && resp.IsError() {
		return ErrAPIResponse{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
			Operation:  "update backend",
		}
	}
	return nil
}

// ListBackends returns all backends managed by this controller
func (c *Client) ListBackends() ([]*models.Backend, error) {
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends", c.config.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	if resp.IsError() {
		return nil, ErrAPIResponse{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
			Operation:  "list backends",
		}
	}

	var all []*models.Backend
	if err := json.Unmarshal(resp.Body(), &all); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	var managed []*models.Backend
	for _, b := range all {
		if b.Description == ManagedDescription {
			managed = append(managed, b)
		}
	}
	return managed, nil
}

// DeleteBackend deletes a backend only if it's managed by this controller
func (c *Client) DeleteBackend(name string) error {
	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return fmt.Errorf("getting config version: %w", err)
	}

	b, err := c.GetBackend(name)
	if err != nil {
		return fmt.Errorf("getting backend: %w", err)
	}

	if b == nil {
		return ErrResourceNotFound{
			ResourceType: "backend",
			ResourceName: name,
		}
	}

	if b.Description != ManagedDescription {
		return ErrNotManaged{
			ResourceType: "backend",
			ResourceName: name,
		}
	}

	resp, err := c.client.R().
		SetQueryParam(queryKey, queryVal).
		Delete(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s", c.config.BaseURL, name))
	if err != nil {
		return fmt.Errorf("api request failed: %w", err)
	}
	if resp.IsError() && resp.StatusCode() != 404 {
		return ErrAPIResponse{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
			Operation:  "delete backend",
		}
	}
	if resp.StatusCode() == 200 || resp.StatusCode() == 204 || resp.StatusCode() == 202 {
		c.transactionDirty = true
	}
	return nil
}

// backendsEqual compares two backends for equality
func (c *Client) backendsEqual(a, b *models.Backend) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Name != b.Name {
		return false
	}

	var aAlg, bAlg string
	if a.Balance != nil && a.Balance.Algorithm != nil {
		aAlg = *a.Balance.Algorithm
	}
	if b.Balance != nil && b.Balance.Algorithm != nil {
		bAlg = *b.Balance.Algorithm
	}
	if aAlg != bAlg {
		return false
	}

	if a.AdvCheck != b.AdvCheck {
		return false
	}

	var aMode, bMode string
	if a.Mode != "" {
		aMode = a.Mode
	}
	if b.Mode != "" {
		bMode = b.Mode
	}
	if aMode != bMode {
		return false
	}

	if a.Description != b.Description {
		return false
	}
	return true
}

// http_check

// EnsureBackendHTTPCheck sets HTTP check configuration for a backend
func (c *Client) EnsureBackendHTTPCheck(backend string, httpChecks []*models.HTTPCheck) error {
	current, err := c.getCurrentHTTPChecks(backend)
	if err != nil {
		return err
	}
	if c.httpChecksEqual(current, httpChecks) {
		return nil
	}

	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return err
	}

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(httpChecks).
		SetQueryParam(queryKey, queryVal).
		Put(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/http_checks", c.config.BaseURL, backend))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("failed to update http_check, status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

func (c *Client) getCurrentHTTPChecks(backend string) ([]*models.HTTPCheck, error) {
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/http_checks", c.config.BaseURL, backend))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 404 {
		return nil, nil
	}

	var checks []*models.HTTPCheck
	if err := json.Unmarshal(resp.Body(), &checks); err != nil {
		return nil, err
	}
	return checks, nil
}

// httpChecksEqual compares two HTTP check slices for equality
func (c *Client) httpChecksEqual(a, b []*models.HTTPCheck) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !c.httpCheckEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// httpCheckEqual compares two HTTP checks for equality
func (c *Client) httpCheckEqual(a, b *models.HTTPCheck) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Type != b.Type || a.Method != b.Method || a.URI != b.URI {
		return false
	}
	if len(a.CheckHeaders) != len(b.CheckHeaders) {
		return false
	}
	for i := range a.CheckHeaders {
		ha := a.CheckHeaders[i]
		hb := b.CheckHeaders[i]
		if ha == nil || hb == nil {
			if ha != hb {
				return false
			}
			continue
		}
		if (ha.Name == nil) != (hb.Name == nil) ||
			(ha.Name != nil && hb.Name != nil && *ha.Name != *hb.Name) {
			return false
		}
		if (ha.Fmt == nil) != (hb.Fmt == nil) ||
			(ha.Fmt != nil && hb.Fmt != nil && *ha.Fmt != *hb.Fmt) {
			return false
		}
	}
	return true
}

// Server

// GetServer retrieves a server from a backend
func (c *Client) GetServer(backend, name string) (*models.Server, error) {
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		SetResult(&models.Server{}).
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers/%s", c.config.BaseURL, backend, name))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 404 {
		return nil, nil
	}
	return resp.Result().(*models.Server), nil
}

// EnsureServer creates or updates a server in a backend
func (c *Client) EnsureServer(backend string, server *models.Server) error {
	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return err
	}

	existing, err := c.GetServer(backend, server.Name)
	if err != nil {
		return err
	}

	var resp *resty.Response
	if existing == nil {
		resp, err = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(server).
			SetQueryParam(queryKey, queryVal).
			Post(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers", c.config.BaseURL, backend))
		c.transactionDirty = true
	} else if !c.serversEqual(existing, server) {
		resp, err = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(server).
			SetQueryParam(queryKey, queryVal).
			Put(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers/%s", c.config.BaseURL, backend, server.Name))
		c.transactionDirty = true
	}

	if err != nil {
		return err
	}
	if resp != nil && resp.IsError() {
		return fmt.Errorf("failed to update server, status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

// ListServers lists all servers in a backend
func (c *Client) ListServers(backend string) ([]*models.Server, error) {
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers", c.config.BaseURL, backend))
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	if resp.IsError() {
		return nil, ErrAPIResponse{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
			Operation:  "list servers",
		}
	}

	var servers []*models.Server
	if err := json.Unmarshal(resp.Body(), &servers); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}
	return servers, nil
}

// DeleteServer deletes a server from a backend
func (c *Client) DeleteServer(backend, name string) error {
	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return fmt.Errorf("getting config version: %w", err)
	}

	resp, err := c.client.R().
		SetQueryParam(queryKey, queryVal).
		Delete(fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers/%s", c.config.BaseURL, backend, name))
	if err != nil {
		return fmt.Errorf("api request failed: %w", err)
	}
	if resp.IsError() && resp.StatusCode() != 404 {
		return ErrAPIResponse{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
			Operation:  "delete server",
		}
	}
	if resp.StatusCode() == 200 || resp.StatusCode() == 204 || resp.StatusCode() == 202 {
		c.transactionDirty = true
	}
	return nil
}

// serversEqual compares two servers for equality
func (c *Client) serversEqual(a, b *models.Server) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Name != b.Name || a.Address != b.Address {
		return false
	}

	if (a.Port == nil) != (b.Port == nil) {
		return false
	}
	if a.Port != nil && b.Port != nil && *a.Port != *b.Port {
		return false
	}

	if !strings.EqualFold(a.Check, b.Check) {
		return false
	}
	return true
}
