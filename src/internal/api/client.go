package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"devbox-cli/internal/config"
)

// Client is an HTTP client that injects auth headers for every devbox API request.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// New creates a Client with the given base URL and auth token.
func New(baseURL, token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		token:      token,
	}
}

// NewDefault creates a Client by loading config from ~/.devbox/config.json.
func NewDefault() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return New(cfg.ServerURL, cfg.Token), nil
}

// do executes an HTTP request, injecting Authorization and Content-Type headers.
func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	return resp, nil
}

// Get performs a GET request to the given API path.
func (c *Client) Get(path string) (*http.Response, error) {
	return c.do(http.MethodGet, path, nil)
}

// Post performs a POST request with a JSON-encoded body.
func (c *Client) Post(path string, body any) (*http.Response, error) {
	return c.do(http.MethodPost, path, body)
}

// Delete performs a DELETE request to the given API path.
func (c *Client) Delete(path string) (*http.Response, error) {
	return c.do(http.MethodDelete, path, nil)
}

// DecodeJSON reads and closes the response body, decoding it as JSON into dst.
func DecodeJSON(resp *http.Response, dst any) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// CheckStatus returns an error if the response status code is >= 400,
// reading and closing the body in that case.
func CheckStatus(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
