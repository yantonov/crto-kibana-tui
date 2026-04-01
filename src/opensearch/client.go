package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is a thin HTTP wrapper for the OpenSearch _search API.
type Client struct {
	httpClient    *http.Client
	sessionCookie string // set after Login(); takes priority over basic auth
	username      string // fallback for diag mode
	password      string
}

// NewClient creates an unauthenticated Client. Call Login() before use.
func NewClient() *Client {
	return &Client{httpClient: &http.Client{}}
}

// NewClientWithBasicAuth creates a Client using HTTP Basic Auth.
// Used by diagnostic mode where credentials come from env vars.
func NewClientWithBasicAuth(username, password string) *Client {
	return &Client{
		httpClient: &http.Client{},
		username:   username,
		password:   password,
	}
}

// Login authenticates against Kibana's /auth/login endpoint and stores the
// returned session cookie. The cookie is reused for all subsequent requests.
func (c *Client) Login(ctx context.Context, kibanaURL, username, password string) error {
	payload, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return fmt.Errorf("marshal login payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kibanaURL+"/auth/login", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("osd-xsrf", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid credentials")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed (HTTP %d): %s", resp.StatusCode, body)
	}

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return fmt.Errorf("no session cookie returned by login endpoint")
	}
	parts := make([]string, len(cookies))
	for i, ck := range cookies {
		parts[i] = ck.Name + "=" + ck.Value
	}
	c.sessionCookie = strings.Join(parts, "; ")
	return nil
}

// IsAuthenticated reports whether a session cookie has been obtained.
func (c *Client) IsAuthenticated() bool {
	return c.sessionCookie != ""
}

// addAuth injects authentication onto the request: session cookie if
// available, otherwise HTTP Basic Auth.
func (c *Client) addAuth(req *http.Request) {
	if c.sessionCookie != "" {
		req.Header.Set("Cookie", c.sessionCookie)
	} else {
		req.SetBasicAuth(c.username, c.password)
	}
}

// Ping makes a lightweight GET /api/status request to verify that the Kibana
// endpoint is reachable and the credentials are accepted.
func (c *Client) Ping(ctx context.Context, kibanaURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kibanaURL+"/api/status", nil)
	if err != nil {
		return fmt.Errorf("build ping request: %w", err)
	}
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid credentials (HTTP 401)")
	}
	return nil
}

// Search executes a _search request via the Kibana console proxy.
// body must be a JSON-serialisable map (as returned by BuildQuery).
func (c *Client) Search(ctx context.Context, kibanaURL, indexPattern string, body map[string]interface{}) (*SearchResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/api/console/proxy?path=%s%%2F_search&method=POST", kibanaURL, indexPattern)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("osd-xsrf", "true")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opensearch returned %d: %s", resp.StatusCode, body)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}
