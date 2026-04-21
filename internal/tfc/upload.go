package tfc

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func (c *Client) UploadState(ctx context.Context, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading state file: %w", err)
	}

	// Step 1: Lock
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

	// serial と lineage を state ファイルから取得
	var stateData struct {
		Serial  int64  `json:"serial"`
		Lineage string `json:"lineage"`
	}
	_ = json.Unmarshal(data, &stateData)

	// MD5 ハッシュを計算
	hash := md5.Sum(data)
	md5b64 := base64.StdEncoding.EncodeToString(hash[:])

	// Step 2: State バージョンを作成（コンテンツは送信しない）
	fmt.Println("Creating state version...")
	ws, err := c.GetWorkspace(ctx)
	if err != nil {
		return err
	}

	reqBody := map[string]any{
		"data": map[string]any{
			"type": "state-versions",
			"attributes": map[string]any{
				"serial":  stateData.Serial,
				"lineage": stateData.Lineage,
				"md5":     md5b64,
			},
			"relationships": map[string]any{
				"workspace": map[string]any{
					"data": map[string]any{
						"type": "workspaces",
						"id":   ws.ID,
					},
				},
			},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	var svResult struct {
		Data StateVersion `json:"data"`
	}
	if err := c.post(ctx, "/state-versions", bytes.NewReader(bodyBytes), &svResult); err != nil {
		return fmt.Errorf("creating state version: %w", err)
	}
	sv := &svResult.Data

	// Step 3: hosted-state-upload-url へ PUT
	fmt.Println("Uploading state...")
	uploadURL := sv.Attributes.UploadURL
	if uploadURL == "" {
		return fmt.Errorf("state version has no upload URL")
	}
	if err := c.putState(ctx, uploadURL, data); err != nil {
		return fmt.Errorf("uploading state: %w", err)
	}

	// Step 4: finalized されるまでポーリング
	fmt.Println("Waiting for finalization...")
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := c.pollFinalized(pollCtx, sv.ID); err != nil {
		return fmt.Errorf("waiting for finalization: %w", err)
	}

	// Step 5: Unlock（defer で実行済みだが、正常終了時はここで実行）
	locked = false
	fmt.Println("Unlocking workspace...")
	if err := c.Unlock(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Client) putState(ctx context.Context, url string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) pollFinalized(ctx context.Context, svID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
		sv, err := c.GetStateVersion(ctx, svID)
		if err != nil {
			return err
		}
		if sv.Attributes.Finalized {
			return nil
		}
	}
}
