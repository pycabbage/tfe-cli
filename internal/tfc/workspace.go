package tfc

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
)

func (c *Client) Lock(ctx context.Context, reason string) error {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return err
	}
	_, err = c.tfe.Workspaces.Lock(ctx, ws.ID, tfe.WorkspaceLockOptions{
		Reason: tfe.String(reason),
	})
	if err != nil {
		if strings.Contains(err.Error(), "locked") {
			return fmt.Errorf("workspace is already locked by another user")
		}
		return fmt.Errorf("locking workspace: %w", err)
	}
	c.workspace = nil
	return nil
}

func (c *Client) Unlock(ctx context.Context) error {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return err
	}
	_, err = c.tfe.Workspaces.Unlock(ctx, ws.ID)
	if err != nil {
		return fmt.Errorf("unlocking workspace: %w", err)
	}
	c.workspace = nil
	return nil
}

func (c *Client) GetCurrentUser(ctx context.Context) (*tfe.User, error) {
	user, err := c.tfe.Users.ReadCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting current user: %w", err)
	}
	return user, nil
}
