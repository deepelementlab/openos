package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client provides OPA (Open Policy Agent) integration.
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new OPA client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		timeout: 5 * time.Second,
	}
}

// EvaluateRequest represents a policy evaluation request.
type EvaluateRequest struct {
	Input map[string]interface{} `json:"input"`
}

// EvaluateResponse represents a policy evaluation response.
type EvaluateResponse struct {
	Result bool                   `json:"result"`
	Decision map[string]interface{} `json:"decision,omitempty"`
}

// Evaluate evaluates a policy decision.
func (c *Client) Evaluate(ctx context.Context, policyPath string, input map[string]interface{}) (*EvaluateResponse, error) {
	reqBody := EvaluateRequest{Input: input}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/data/%s", c.baseURL, policyPath)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result EvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// QueryData queries OPA data.
func (c *Client) QueryData(ctx context.Context, path string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/data/%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// PolicyEvaluator wraps OPA for policy evaluation.
type PolicyEvaluator struct {
	client *Client
	cache  map[string]*cacheEntry
}

// cacheEntry represents a cached policy decision.
type cacheEntry struct {
	result    *EvaluateResponse
	cachedAt  time.Time
}

// NewPolicyEvaluator creates a new policy evaluator.
func NewPolicyEvaluator(client *Client) *PolicyEvaluator {
	return &PolicyEvaluator{
		client: client,
		cache:  make(map[string]*cacheEntry),
	}
}

// Evaluate evaluates a policy with optional caching.
func (e *PolicyEvaluator) Evaluate(ctx context.Context, policyPath string, input map[string]interface{}, useCache bool) (bool, error) {
	// Check cache
	if useCache {
		cacheKey := generateCacheKey(policyPath, input)
		if entry, exists := e.cache[cacheKey]; exists {
			if time.Since(entry.cachedAt) < 5*time.Minute {
				return entry.result.Result, nil
			}
		}
	}

	result, err := e.client.Evaluate(ctx, policyPath, input)
	if err != nil {
		return false, err
	}

	// Cache result
	if useCache {
		cacheKey := generateCacheKey(policyPath, input)
		e.cache[cacheKey] = &cacheEntry{
			result:   result,
			cachedAt: time.Now(),
		}
	}

	return result.Result, nil
}

// generateCacheKey generates a cache key for a policy evaluation.
func generateCacheKey(policyPath string, input map[string]interface{}) string {
	// Simplified cache key generation
	return fmt.Sprintf("%s:%v", policyPath, input)
}

// ClearCache clears the policy evaluation cache.
func (e *PolicyEvaluator) ClearCache() {
	e.cache = make(map[string]*cacheEntry)
}
