package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func NewClient(client *http.Client, baseURL string, apiKey string) *Client {
	return &Client{client: client, baseURL: baseURL, apiKey: apiKey}
}

func (c *Client) Get(ctx context.Context, endpoint string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s", c.baseURL, endpoint), nil)
	if err != nil {
		return err
	}

	req.Header.Set("x-cg-pro-api-key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CoinGecko API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
