package tfc

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testStateJSON = `{"version":4,"serial":3,"lineage":"test-lineage-abc"}`

func writeTempState(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.tfstate")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("テンプファイル作成: %v", err)
	}
	return path
}

// buildUploadMux は UploadState テスト用の共通エンドポイントを持つ mux を組み立てる。
// stateVersionsHandler: POST /state-versions のハンドラ
// pollHandler: GET /state-versions/sv-new001 のハンドラ
// unlockCalled: Unlock が呼ばれたかどうかを記録するポインタ
func buildUploadMux(
	t *testing.T,
	stateVersionsHandler http.HandlerFunc,
	pollHandler http.HandlerFunc,
	unlockCalled *bool,
) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/lock", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/unlock", func(w http.ResponseWriter, r *http.Request) {
		*unlockCalled = true
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/state-versions", stateVersionsHandler)
	if pollHandler != nil {
		mux.HandleFunc("/state-versions/sv-new001", pollHandler)
	}
	return mux
}

// stateVersionsOKHandler は指定した uploadURL を含むレスポンスを返す POST /state-versions ハンドラ。
func stateVersionsOKHandler(uploadURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 201, map[string]any{
			"data": map[string]any{
				"id": "sv-new001",
				"attributes": map[string]any{
					"serial":                    int64(3),
					"created-at":                time.Now().Format(time.RFC3339),
					"status":                    "pending",
					"hosted-state-download-url": "",
					"hosted-state-upload-url":   uploadURL,
					"terraform-version":         "1.5.0",
					"lineage":                   "test-lineage-abc",
					"finalized":                 false,
				},
			},
		})
	}
}

// finalizedAfterN は n 回目の呼び出しで finalized: true を返すポーリングハンドラを返す。
func finalizedAfterN(afterN int) (http.HandlerFunc, *int) {
	count := 0
	return func(w http.ResponseWriter, r *http.Request) {
		count++
		writeJSON(w, 200, map[string]any{
			"data": map[string]any{
				"id": "sv-new001",
				"attributes": map[string]any{
					"serial":    int64(3),
					"finalized": count >= afterN,
					"status":    "finalized",
				},
			},
		})
	}, &count
}

// neverFinalized は常に finalized: false を返すポーリングハンドラ。
var neverFinalized http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"data": map[string]any{
			"id": "sv-new001",
			"attributes": map[string]any{
				"serial":    int64(3),
				"finalized": false,
				"status":    "pending",
			},
		},
	})
}

// --- TestUploadState_FileNotFound ---

func TestUploadState_FileNotFound(t *testing.T) {
	c := newTestClient(t, http.NewServeMux())

	err := c.UploadState(t.Context(), "/nonexistent/path/test.tfstate")
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "reading state file") {
		t.Errorf("エラーに 'reading state file' が含まれていない: %v", err)
	}
}

// --- TestUploadState_Success ---

func TestUploadState_Success(t *testing.T) {
	// 期待する MD5 を事前計算
	data := []byte(testStateJSON)
	hash := md5.Sum(data)
	wantMD5 := base64.StdEncoding.EncodeToString(hash[:])

	// アップロード先サーバー
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("PUT メソッド期待: got %s", r.Method)
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(uploadSrv.Close)

	// POST /state-versions でリクエストボディを検証
	stateVersionsHandler := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("ボディパース失敗: %v", err)
		}
		attrs, _ := req["data"].(map[string]any)["attributes"].(map[string]any)
		if fmt.Sprintf("%v", attrs["serial"]) != "3" {
			t.Errorf("serial: got %v, want 3", attrs["serial"])
		}
		if attrs["lineage"] != "test-lineage-abc" {
			t.Errorf("lineage: got %v, want test-lineage-abc", attrs["lineage"])
		}
		if attrs["md5"] != wantMD5 {
			t.Errorf("md5: got %v, want %v", attrs["md5"], wantMD5)
		}
		stateVersionsOKHandler(uploadSrv.URL + "/")(w, r)
	}

	pollHandler, pollCount := finalizedAfterN(2)
	unlockCalled := false
	mux := buildUploadMux(t, stateVersionsHandler, pollHandler, &unlockCalled)

	c := newTestClient(t, mux)
	c.pollInterval = 1 * time.Millisecond

	if err := c.UploadState(t.Context(), writeTempState(t, testStateJSON)); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if *pollCount < 2 {
		t.Errorf("ポーリング回数: got %d, want >= 2", *pollCount)
	}
	if !unlockCalled {
		t.Error("Unlock が呼ばれなかった")
	}
}

// --- TestUploadState_LockFailed ---

func TestUploadState_LockFailed(t *testing.T) {
	unlockCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/lock", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 500, map[string]any{
			"errors": []map[string]any{{"status": "500", "title": "Internal Server Error"}},
		})
	})
	mux.HandleFunc("/workspaces/ws-001/actions/unlock", func(w http.ResponseWriter, r *http.Request) {
		unlockCalled = true
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})

	c := newTestClient(t, mux)
	if err := c.UploadState(t.Context(), writeTempState(t, testStateJSON)); err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if unlockCalled {
		t.Error("Lock が失敗したのに Unlock が呼ばれた")
	}
}

// --- TestUploadState_CreateStateVersionFailed ---

func TestUploadState_CreateStateVersionFailed(t *testing.T) {
	unlockCalled := false
	mux := buildUploadMux(t,
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 422, map[string]any{
				"errors": []map[string]any{{"status": "422", "title": "Invalid serial"}},
			})
		},
		nil,
		&unlockCalled,
	)

	c := newTestClient(t, mux)
	err := c.UploadState(t.Context(), writeTempState(t, testStateJSON))
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "creating state version") {
		t.Errorf("エラーに 'creating state version' が含まれていない: %v", err)
	}
	if !unlockCalled {
		t.Error("Lock 後のエラーで Unlock が呼ばれなかった（defer）")
	}
}

// --- TestUploadState_UploadURLMissing ---

func TestUploadState_UploadURLMissing(t *testing.T) {
	unlockCalled := false
	mux := buildUploadMux(t,
		func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 201, map[string]any{
				"data": map[string]any{
					"id":         "sv-new001",
					"attributes": map[string]any{"hosted-state-upload-url": ""},
				},
			})
		},
		nil,
		&unlockCalled,
	)

	c := newTestClient(t, mux)
	err := c.UploadState(t.Context(), writeTempState(t, testStateJSON))
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "no upload URL") {
		t.Errorf("エラーに 'no upload URL' が含まれていない: %v", err)
	}
	if !unlockCalled {
		t.Error("Unlock が呼ばれなかった（defer）")
	}
}

// --- TestUploadState_PutFailed ---

func TestUploadState_PutFailed(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = fmt.Fprint(w, "upload error")
	}))
	t.Cleanup(uploadSrv.Close)

	unlockCalled := false
	mux := buildUploadMux(t,
		stateVersionsOKHandler(uploadSrv.URL+"/"),
		nil,
		&unlockCalled,
	)

	c := newTestClient(t, mux)
	err := c.UploadState(t.Context(), writeTempState(t, testStateJSON))
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "uploading state") {
		t.Errorf("エラーに 'uploading state' が含まれていない: %v", err)
	}
	if !unlockCalled {
		t.Error("Unlock が呼ばれなかった（defer）")
	}
}

// --- TestUploadState_FinalizationTimeout ---

func TestUploadState_FinalizationTimeout(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(uploadSrv.Close)

	unlockCalled := false
	mux := buildUploadMux(t,
		stateVersionsOKHandler(uploadSrv.URL+"/"),
		neverFinalized,
		&unlockCalled,
	)

	c := newTestClient(t, mux)
	c.pollInterval = 1 * time.Millisecond

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	err := c.UploadState(ctx, writeTempState(t, testStateJSON))
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("エラーが context.DeadlineExceeded でない: %v", err)
	}
	if !unlockCalled {
		t.Error("タイムアウト後に Unlock が呼ばれなかった（defer）")
	}
}
