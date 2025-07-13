package haproxyclient

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/haproxytech/client-native/v6/models"
)

// GetFrontend retrieves a frontend by name
func (c *Client) GetFrontend(name string) (*models.Frontend, error) {
	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return nil, err
	}
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		SetResult(&models.Frontend{}).
		SetQueryParam(queryKey, queryVal).
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s", c.config.BaseURL, name))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 404 {
		return nil, nil
	}
	return resp.Result().(*models.Frontend), nil
}

// EnsureFrontend creates or updates a frontend, ensuring it's managed by this controller
func (c *Client) EnsureFrontend(frontend *models.Frontend) error {
	frontend.Description = ManagedDescription

	var (
		resp *resty.Response
		err  error
	)

	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return err
	}

	existing, err := c.GetFrontend(frontend.Name)
	if err != nil {
		return err
	}

	if existing == nil {
		resp, err = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(frontend).
			SetQueryParam(queryKey, queryVal).
			Post(fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends", c.config.BaseURL))
		c.transactionDirty = true
	} else if existing.Description != ManagedDescription {
		return fmt.Errorf("frontend %q exists but is not managed by this controller; refusing to update or overwrite", frontend.Name)
	} else if !c.frontendsEqual(existing, frontend) {
		resp, err = c.client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(frontend).
			SetQueryParam(queryKey, queryVal).
			Put(fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s", c.config.BaseURL, frontend.Name))
		c.transactionDirty = true
	}

	if err != nil {
		return err
	}
	if resp != nil && resp.IsError() {
		return fmt.Errorf("failed to update frontend, status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

// frontendsEqual compares two frontends for equality
func (c *Client) frontendsEqual(a, b *models.Frontend) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Name != b.Name || a.Mode != b.Mode || a.DefaultBackend != b.DefaultBackend {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	return true
}

// Bind functions

// EnsureBind creates or updates a bind configuration for a frontend
func (c *Client) EnsureBind(frontend string, bind *models.Bind) error {
	var (
		resp *resty.Response
		err  error
	)

	url := fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/binds", c.config.BaseURL, frontend)

	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return err
	}

	// List all binds for the frontend
	resp, err = c.client.R().
		SetHeader("Accept", "application/json").
		Get(url)
	if err != nil {
		return err
	}

	var binds []models.Bind
	if err := json.Unmarshal(resp.Body(), &binds); err != nil {
		return err
	}

	// Check if bind already exists
	var found *models.Bind
	for _, b := range binds {
		if c.bindsEqual(&b, bind) {
			found = &b
			break
		}
	}

	if found != nil {
		return nil // Already exists and is equal
	}

	// Create new bind
	resp, err = c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(bind).
		SetQueryParam(queryKey, queryVal).
		Post(url)
	c.transactionDirty = true
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("failed to create bind, status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

// bindsEqual compares two bind configurations for equality
func (c *Client) bindsEqual(a, b *models.Bind) bool {
	if a.Address != b.Address {
		return false
	}
	if (a.Port == nil) != (b.Port == nil) {
		return false
	}
	if a.Port != nil && b.Port != nil && *a.Port != *b.Port {
		return false
	}
	return true
}

// Backend Switching Rule
// ListBackendSwitchingRules returns all backend switching rules for a frontend
func (c *Client) ListBackendSwitchingRules(frontend string) ([]*models.BackendSwitchingRule, error) {
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/backend_switching_rules", c.config.BaseURL, frontend))
	if err != nil {
		return nil, err
	}

	var rules []*models.BackendSwitchingRule
	if err := json.Unmarshal(resp.Body(), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// EnsureBackendSwitchingRule creates or updates a backend switching rule
func (c *Client) EnsureBackendSwitchingRule(frontend string, rule *models.BackendSwitchingRule) error {
	var (
		resp *resty.Response
		err  error
	)

	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return err
	}

	// List all existing rules
	ruleList, err := c.ListBackendSwitchingRules(frontend)
	if err != nil {
		return err
	}

	// Check if rule already exists
	for _, r := range ruleList {
		if r.CondTest == rule.CondTest && r.Name == rule.Name {
			return nil // Already exists
		}
	}

	// Add new rule and update all rules
	ruleList = append(ruleList, rule)
	resp, err = c.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(ruleList).
		SetQueryParam(queryKey, queryVal).
		Put(fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/backend_switching_rules", c.config.BaseURL, frontend))
	c.transactionDirty = true
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("failed to upsert backend switching rules, status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

// DeleteBackendSwitchingRuleByIndex deletes a backend switching rule by index
func (c *Client) DeleteBackendSwitchingRuleByIndex(frontend string, index int64) error {
	var (
		resp *resty.Response
		err  error
	)

	queryKey, queryVal, err := c.getVersionOrTransactionParam()
	if err != nil {
		return err
	}

	resp, err = c.client.R().
		SetQueryParam(queryKey, queryVal).
		Delete(fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/backend_switching_rules/%d", c.config.BaseURL, frontend, index))
	if err != nil {
		return err
	}
	if resp.IsError() && resp.StatusCode() != 404 {
		return fmt.Errorf("failed to delete backend switching rule: %v", resp.String())
	}
	if resp.StatusCode() == 200 || resp.StatusCode() == 204 || resp.StatusCode() == 202 {
		c.transactionDirty = true
	}
	return nil
}
