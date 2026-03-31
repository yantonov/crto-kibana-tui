package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is a thin HTTP wrapper for the OpenSearch _search API.
type Client struct {
	httpClient *http.Client
	username   string
	password   string
}

// NewClient creates a Client with the given Basic Auth credentials.
func NewClient(username, password string) *Client {
	return &Client{
		httpClient: &http.Client{},
		username:   username,
		password:   password,
	}
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
	req.SetBasicAuth(c.username, c.password)

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
