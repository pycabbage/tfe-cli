package tfc

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

func (c *Client) UploadState(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading state file: %w", err)
	}

	fmt.Println("Locking workspace...")
	if err := c.Lock(ctx, "tfe-cli state upload"); err != nil {
		return err
	}
	locked := true
	defer func() {
		if locked {
			fmt.Println("Unlocking workspace...")
			_ = c.Unlock(context.Background())
		}
	}()

	var stateData struct {
		Serial  int64  `json:"serial"`
		Lineage string `json:"lineage"`
	}
	_ = json.Unmarshal(data, &stateData)

	hash := md5.Sum(data)
	md5b64 := base64.StdEncoding.EncodeToString(hash[:])

	fmt.Println("Creating state version...")
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return err
	}

	sv, err := c.tfe.StateVersions.Upload(ctx, ws.ID, tfe.StateVersionUploadOptions{
		StateVersionCreateOptions: tfe.StateVersionCreateOptions{
			Lineage: tfe.String(stateData.Lineage),
			MD5:     tfe.String(md5b64),
			Serial:  tfe.Int64(stateData.Serial),
		},
		RawState: data,
	})
	if err != nil {
		return fmt.Errorf("creating state version: %w", err)
	}

	fmt.Println("Waiting for finalization...")
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := c.pollFinalized(pollCtx, sv.ID); err != nil {
		return fmt.Errorf("waiting for finalization: %w", err)
	}

	locked = false
	fmt.Println("Unlocking workspace...")
	return c.Unlock(ctx)
}

func (c *Client) pollFinalized(ctx context.Context, svID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.pollInterval):
		}
		sv, err := c.tfe.StateVersions.Read(ctx, svID)
		if err != nil {
			return err
		}
		if sv.Status == tfe.StateVersionFinalized {
			return nil
		}
	}
}
