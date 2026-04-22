package tfc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pycabbage/tfe-cli/internal/config"
)

type Client struct {
	http         *http.Client
	cfg          *config.Config
	workspace    *Workspace
	baseURL      string
	pollInterval time.Duration
}

func New(cfg *config.Config) *Client {
	return &Client{
		http:         &http.Client{Timeout: 30 * time.Second},
		cfg:          cfg,
		baseURL:      "https://app.terraform.io/api/v2",
		pollInterval: 3 * time.Second,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	return c.http.Do(req)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) post(ctx context.Context, path string, body io.Reader, out any) error {
	resp, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func parseAPIError(resp *http.Response) error {
	var errResp struct {
		Errors []struct {
			Status string `json:"status"`
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil || len(errResp.Errors) == 0 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	e := errResp.Errors[0]
	if e.Detail != "" {
		return fmt.Errorf("%s: %s", e.Title, e.Detail)
	}
	return fmt.Errorf("%s", e.Title)
}

func (c *Client) GetWorkspace(ctx context.Context) (*Workspace, error) {
	if c.workspace != nil {
		return c.workspace, nil
	}
	var result struct {
		Data Workspace `json:"data"`
	}
	path := fmt.Sprintf("/organizations/%s/workspaces/%s", c.cfg.Organization, c.cfg.WorkspaceName)
	if err := c.get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("getting workspace: %w", err)
	}
	c.workspace = &result.Data
	return c.workspace, nil
}
