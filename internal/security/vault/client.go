// Package vault provides a minimal Vault API client facade (token auth stub).
package vault

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Client is a thin HTTP client for KV v2 secrets (paths only; full impl TBD).
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

// NewClient creates a Vault client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Health hits /v1/sys/health when configured.
func (c *Client) Health(ctx context.Context) error {
	if c.BaseURL == "" {
		return fmt.Errorf("vault: base URL not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/sys/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
