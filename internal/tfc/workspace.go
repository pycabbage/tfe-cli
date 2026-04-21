package tfc

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) Lock(ctx context.Context, reason string) error {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return err
	}
	body := strings.NewReader(fmt.Sprintf(`{"reason":"%s"}`, reason))
	if err := c.post(ctx, "/workspaces/"+ws.ID+"/actions/lock", body, nil); err != nil {
		if strings.Contains(err.Error(), "locked") {
			return fmt.Errorf("workspace is already locked by another user")
		}
		return fmt.Errorf("locking workspace: %w", err)
	}
	c.workspace = nil // キャッシュをクリア
	return nil
}

func (c *Client) Unlock(ctx context.Context) error {
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return err
	}
	if err := c.post(ctx, "/workspaces/"+ws.ID+"/actions/unlock", nil, nil); err != nil {
		return fmt.Errorf("unlocking workspace: %w", err)
	}
	c.workspace = nil // キャッシュをクリア
	return nil
}

func (c *Client) GetAccountDetails(ctx context.Context) (*AccountDetails, error) {
	var result AccountDetails
	if err := c.get(ctx, "/account/details", &result); err != nil {
		return nil, fmt.Errorf("getting account details: %w", err)
	}
	return &result, nil
}
