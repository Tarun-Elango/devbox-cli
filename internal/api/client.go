package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"devbox-cli/internal/config"
)

// Client is an HTTP client that injects auth headers for every devbox API request.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	cfg        *config.Config // nil for unauthenticated clients (e.g. login)
}

// New creates a Client with the given base URL and auth token.
// cfg is nil; no transparent refresh is performed.
func New(baseURL, token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		token:      token,
	}
}

// NewWithTimeout creates a Client with a custom HTTP timeout.
// Use this for long-blocking calls such as box creation, which waits for EC2
// status checks to pass before returning (can take several minutes).
func NewWithTimeout(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		token:      token,
	}
}

// NewDefault creates a Client by loading config from ~/.devbox/config.json.
// The full config is attached so the client can transparently refresh tokens.
func NewDefault() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    cfg.ServerURL,
		token:      cfg.Token,
		cfg:        cfg,
	}, nil
}

// execute builds and fires a single HTTP request without any token refresh logic.
func (c *Client) execute(method, path string, body any) (*http.Response, error) {
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

// do executes an HTTP request with:
//   - Phase 4: pre-flight expiry check — refreshes proactively if the access
//     token is expired before sending the request.
//   - Phase 3: transparent 401 retry — if the server returns 401, attempts one
//     refresh and retries. If the refresh itself fails (refresh token expired),
//     returns a clear "session expired" error.
func (c *Client) do(method, path string, body any) (*http.Response, error) {
	// Fast-fail if there are no credentials at all — avoids a round-trip that
	// would return an opaque 401/403 with no actionable message.
	if c.cfg != nil && c.cfg.Token == "" && c.cfg.RefreshToken == "" {
		return nil, fmt.Errorf("not logged in — please run `devbox login`")
	}

	// Pre-flight: refresh before the request if we already know the token is stale.
	if c.cfg != nil && c.cfg.RefreshToken != "" && c.cfg.IsTokenExpired() {
		if err := refreshAccessToken(c.cfg); err != nil {
			return nil, fmt.Errorf("session expired — please run `devbox login`")
		}
		c.token = c.cfg.Token
	}

	resp, err := c.execute(method, path, body)
	if err != nil {
		return nil, err
	}

	// 401 retry: the token may have expired mid-session or the pre-flight check
	// had no expiry info. Attempt one silent refresh then retry.
	if resp.StatusCode == http.StatusUnauthorized && c.cfg != nil && c.cfg.RefreshToken != "" {
		resp.Body.Close() // close the original 401 response body before retrying
		if err := refreshAccessToken(c.cfg); err != nil {
			return nil, fmt.Errorf("session expired — please run `devbox login`")
		}
		c.token = c.cfg.Token
		return c.execute(method, path, body)
	}

	return resp, nil
}

// refreshAccessToken POSTs the stored refresh token to /v1/auth/refresh,
// then updates cfg.Token and cfg.TokenExpiry in-place and persists to disk.
func refreshAccessToken(cfg *config.Config) error {
	payload, err := json.Marshal(map[string]string{"refreshToken": cfg.RefreshToken})
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Post(cfg.ServerURL+"/v1/auth/refresh", "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh rejected (status %d)", resp.StatusCode)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Token == "" {
		return fmt.Errorf("invalid refresh response")
	}
	cfg.Token = result.Token
	cfg.TokenExpiry = config.ParseTokenExpiry(result.Token)
	return config.Save(cfg)
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

// FailBox prints a clean error for a box subcommand and exits.
func FailBox(cmd string, err error) {
	fmt.Fprintf(os.Stderr, "%s failed: %s\n", cmd, err)
	os.Exit(1)
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
		return errors.New(ParseErrorBody(body))
	}
	return nil
}

// ParseErrorBody extracts a human-readable message from an API error response.
// Handles JSON objects ({error, detail, message}), plain JSON strings, and raw text.
func ParseErrorBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "request failed"
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err == nil {
		if msg := jsonStringField(obj, "message"); msg != "" {
			return msg
		}
		if detail := jsonStringField(obj, "detail"); detail != "" {
			return cleanSpringDetail(detail)
		}
		if errMsg := jsonStringField(obj, "error"); errMsg != "" {
			return errMsg
		}
	}

	var str string
	if err := json.Unmarshal(body, &str); err == nil && str != "" {
		return str
	}

	return trimmed
}

func jsonStringField(obj map[string]json.RawMessage, key string) string {
	raw, ok := obj[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || s == "" {
		return ""
	}
	return s
}

// cleanSpringDetail turns `404 NOT_FOUND "Box not found: abc"` into `Box not found: abc`.
func cleanSpringDetail(detail string) string {
	if i := strings.Index(detail, `"`); i >= 0 {
		if j := strings.LastIndex(detail, `"`); j > i {
			return detail[i+1 : j]
		}
	}
	return detail
}
