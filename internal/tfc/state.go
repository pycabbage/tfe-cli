package tfc

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) ListStateVersions(ctx context.Context) ([]StateVersion, error) {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []StateVersion `json:"data"`
	}
	path := fmt.Sprintf("/workspaces/%s/state-versions?page[size]=10&page[number]=1", ws.ID)
	if err := c.get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("listing state versions: %w", err)
	}
	return result.Data, nil
}

func (c *Client) GetLatestStateVersion(ctx context.Context) (*StateVersion, error) {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data StateVersion `json:"data"`
	}
	if err := c.get(ctx, "/workspaces/"+ws.ID+"/current-state-version", &result); err != nil {
		return nil, fmt.Errorf("getting current state version: %w", err)
	}
	return &result.Data, nil
}

func (c *Client) GetStateVersion(ctx context.Context, id string) (*StateVersion, error) {
	if id == "" || id == "latest" {
		return c.GetLatestStateVersion(ctx)
	}
	var result struct {
		Data StateVersion `json:"data"`
	}
	if err := c.get(ctx, "/state-versions/"+id, &result); err != nil {
		return nil, fmt.Errorf("getting state version %s: %w", id, err)
	}
	return &result.Data, nil
}

func (c *Client) DownloadState(ctx context.Context, sv *StateVersion) ([]byte, error) {
	url := sv.Attributes.DownloadURL
	if url == "" {
		return nil, fmt.Errorf("state version has no download URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading state: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}
	return data, nil
}
