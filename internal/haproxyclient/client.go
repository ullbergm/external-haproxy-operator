package haproxyclient

import (
	"strconv"
	"time"

	"github.com/ullbergm/external-haproxy-operator/monitoring"

	"github.com/go-resty/resty/v2"
)

// Transaction represents a HAProxy Data Plane API transaction
type Transaction struct {
	ID      string `json:"id"`
	Version int    `json:"_version"`
	Status  string `json:"status"`
}

// Client provides HAProxy Data Plane API operations
type Client struct {
	client               *resty.Client
	config               HAProxyConfig
	currentTransactionID string
	transactionDirty     bool // Tracks if any create, update, or delete was performed
}

// Config holds the configuration for the HAProxy client
type HAProxyConfig struct {
	BaseURL  string
	Username string
	Password string
	Timeout  time.Duration
}

// NewHAProxyClient creates a new HAProxy client that implements the HAProxyClient interface
func NewHAProxyClient(config HAProxyConfig) HAProxyClient {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	client := resty.New().
		SetBasicAuth(config.Username, config.Password).
		SetDisableWarn(true).
		SetTimeout(timeout)

	return &Client{
		client:           client,
		config:           config,
		transactionDirty: false,
	}
}

// StartTransaction starts a new transaction and returns its ID.
// Requires passing the current config version as a query parameter.
// Returns a 202 status code if successful, or a 409 error if there are too many transactions.
// StartTransaction starts a new transaction and sets it as the current transaction.
func (c *Client) StartTransaction() (Transaction, error) {
	version, err := c.GetConfigVersion()
	if err != nil {
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, err
	}

	resp, err := c.client.R().
		SetQueryParam("version", strconv.FormatInt(version, 10)).
		SetResult(&Transaction{}).
		Post(c.config.BaseURL + "/v3/services/haproxy/transactions")
	if err != nil {
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, err
	}
	if resp.StatusCode() == 409 {
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, &APIError{StatusCode: 409, Body: "Too many transactions"}
	}
	if resp.IsError() {
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, restyErr(resp)
	}
	tr := resp.Result().(*Transaction)
	c.currentTransactionID = tr.ID
	c.transactionDirty = false
	return *tr, nil
}

// CommitTransaction commits the transaction with the given ID.
// The forceReload parameter controls whether to force a reload (default: false).
// Returns 200 if successfully committed, 202 if accepted and reload requested,
// 400 for bad request, 404 if not found, 406 if cannot be handled.
// CommitTransaction commits the current or specified transaction and clears it from the client.
func (c *Client) CommitTransaction(id string, forceReload bool) (Transaction, error) {
	if !c.transactionDirty {
		// No changes, delete the transaction instead of committing
		err := c.DeleteTransaction(id)
		if err != nil {
			monitoring.HAProxyClientErrorCountTotal.Inc()
		}
		c.transactionDirty = false
		return Transaction{}, err
	}

	resp, err := c.client.R().
		SetQueryParam("force_reload", boolToString(forceReload)).
		SetResult(&Transaction{}).
		Put(c.config.BaseURL + "/v3/services/haproxy/transactions/" + id)
	if err != nil {
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, err
	}

	switch resp.StatusCode() {
	case 200, 202:
		tr := resp.Result().(*Transaction)
		if c.currentTransactionID == id {
			c.currentTransactionID = ""
		}
		return *tr, nil
	case 400:
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, &APIError{StatusCode: 400, Body: "Bad request"}
	case 404:
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, &APIError{StatusCode: 404, Body: "Resource not found"}
	case 406:
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return Transaction{}, &APIError{StatusCode: 406, Body: "Resource cannot be handled"}
	default:
		if resp.IsError() {
			monitoring.HAProxyClientErrorCountTotal.Inc()
			return Transaction{}, restyErr(resp)
		}
		tr := resp.Result().(*Transaction)
		if c.currentTransactionID == id {
			c.currentTransactionID = ""
		}
		return *tr, nil
	}
}

// boolToString converts a bool to its string representation ("true" or "false").
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// DeleteTransaction deletes (aborts) the transaction with the given ID.
// Returns nil if deleted (204), or APIError if not found (404).
// DeleteTransaction deletes (aborts) the transaction with the given ID and clears it if it was current.
func (c *Client) DeleteTransaction(id string) error {
	resp, err := c.client.R().
		Delete(c.config.BaseURL + "/v3/services/haproxy/transactions/" + id)
	if err != nil {
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return err
	}
	switch resp.StatusCode() {
	case 204:
		if c.currentTransactionID == id {
			c.currentTransactionID = ""
		}
		return nil
	case 404:
		monitoring.HAProxyClientErrorCountTotal.Inc()
		return &APIError{StatusCode: 404, Body: "Transaction not found"}
	default:
		if resp.IsError() {
			monitoring.HAProxyClientErrorCountTotal.Inc()
			return restyErr(resp)
		}
		return nil
	}
}

// restyErr extracts error details from a resty.Response
func restyErr(resp *resty.Response) error {
	// Could we just do this and assume the restyErr function is called every time?
	// monitoring.HAProxyClientErrorCountTotal.Inc()
	return &APIError{StatusCode: resp.StatusCode(), Body: resp.String()}
}

// APIError represents an error from the HAProxy Data Plane API
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return "HAProxy API error: " + e.Body
}

// getVersionOrTransactionParam returns the query key and value for version or transaction_id
func (c *Client) getVersionOrTransactionParam() (string, string, error) {
	if c.currentTransactionID != "" {
		return "transaction_id", c.currentTransactionID, nil
	}
	version, err := c.GetConfigVersion()
	if err != nil {
		return "", "", err
	}
	return "version", strconv.FormatInt(version, 10), nil
}
