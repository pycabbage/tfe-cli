package tfc

import (
	"context"
	"fmt"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/pycabbage/tfe-cli/internal/config"
)

type Client struct {
	tfe          *tfe.Client
	cfg          *config.Config
	workspace    *tfe.Workspace
	pollInterval time.Duration
}

func New(cfg *config.Config) (*Client, error) {
	tfeClient, err := tfe.NewClient(&tfe.Config{
		Token:   cfg.APIToken,
		Address: "https://app.terraform.io",
	})
	if err != nil {
		return nil, fmt.Errorf("creating tfe client: %w", err)
	}
	return &Client{
		tfe:          tfeClient,
		cfg:          cfg,
		pollInterval: 3 * time.Second,
	}, nil
}

func (c *Client) GetWorkspace(ctx context.Context) (*tfe.Workspace, error) {
	if c.workspace != nil {
		return c.workspace, nil
	}
	ws, err := c.tfe.Workspaces.Read(ctx, c.cfg.Organization, c.cfg.WorkspaceName)
	if err != nil {
		return nil, fmt.Errorf("getting workspace: %w", err)
	}
	c.workspace = ws
	return c.workspace, nil
}
