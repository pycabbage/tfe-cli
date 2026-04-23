package tfc

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
)

func (c *Client) ListStateVersions(ctx context.Context) ([]*tfe.StateVersion, error) {
	list, err := c.tfe.StateVersions.List(ctx, &tfe.StateVersionListOptions{
		ListOptions:  tfe.ListOptions{PageSize: 10},
		Organization: c.cfg.Organization,
		Workspace:    c.cfg.WorkspaceName,
	})
	if err != nil {
		return nil, fmt.Errorf("listing state versions: %w", err)
	}
	return list.Items, nil
}

func (c *Client) GetLatestStateVersion(ctx context.Context) (*tfe.StateVersion, error) {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return nil, err
	}
	sv, err := c.tfe.StateVersions.ReadCurrent(ctx, ws.ID)
	if err != nil {
		return nil, fmt.Errorf("getting current state version: %w", err)
	}
	return sv, nil
}

func (c *Client) GetStateVersion(ctx context.Context, id string) (*tfe.StateVersion, error) {
	if id == "" || id == "latest" {
		return c.GetLatestStateVersion(ctx)
	}
	sv, err := c.tfe.StateVersions.Read(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting state version %s: %w", id, err)
	}
	return sv, nil
}

func (c *Client) DownloadState(ctx context.Context, sv *tfe.StateVersion) ([]byte, error) {
	if sv.DownloadURL == "" {
		return nil, fmt.Errorf("state version has no download URL")
	}
	data, err := c.tfe.StateVersions.Download(ctx, sv.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("downloading state: %w", err)
	}
	return data, nil
}
