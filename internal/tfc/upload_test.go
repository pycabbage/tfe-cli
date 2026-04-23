package tfc

import (
	"context"
	"errors"
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

func buildUploadMux(t *testing.T, pollHandler http.HandlerFunc, unlockCalled *bool) *http.ServeMux {
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
	if pollHandler != nil {
		mux.HandleFunc("/state-versions/sv-new001", pollHandler)
	}
	return mux
}

func finalizedAfterN(afterN int) (http.HandlerFunc, *int) {
	count := 0
	return func(w http.ResponseWriter, r *http.Request) {
		count++
		status := "pending"
		if count >= afterN {
			status = "finalized"
		}
		writeJSON(w, 200, map[string]any{
			"data": map[string]any{
				"id":   "sv-new001",
				"type": "state-versions",
				"attributes": map[string]any{
					"serial":     int64(3),
					"created-at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					"status":     status,
				},
			},
		})
	}, &count
}

var neverFinalized http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"data": map[string]any{
			"id":   "sv-new001",
			"type": "state-versions",
			"attributes": map[string]any{
				"serial":     int64(3),
				"created-at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				"status":     "pending",
			},
		},
	})
}

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

func TestUploadState_Success(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("PUT メソッド期待: got %s", r.Method)
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(uploadSrv.Close)

	pollHandler, pollCount := finalizedAfterN(2)
	unlockCalled := false
	mux := buildUploadMux(t, pollHandler, &unlockCalled)
	mux.HandleFunc("/workspaces/ws-001/state-versions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 201, map[string]any{
			"data": map[string]any{
				"id":   "sv-new001",
				"type": "state-versions",
				"attributes": map[string]any{
					"serial":                    int64(3),
					"created-at":                time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					"status":                    "pending",
					"hosted-state-upload-url":   uploadSrv.URL + "/upload",
					"hosted-state-download-url": "",
				},
			},
		})
	})

	c := newTestClient(t, mux)
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

func TestUploadState_CreateStateVersionFailed(t *testing.T) {
	unlockCalled := false
	mux := buildUploadMux(t, nil, &unlockCalled)
	mux.HandleFunc("/workspaces/ws-001/state-versions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 422, map[string]any{
			"errors": []map[string]any{{"status": "422", "title": "Invalid serial"}},
		})
	})

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

func TestUploadState_FinalizationTimeout(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(uploadSrv.Close)

	unlockCalled := false
	mux := buildUploadMux(t, neverFinalized, &unlockCalled)
	mux.HandleFunc("/workspaces/ws-001/state-versions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 201, map[string]any{
			"data": map[string]any{
				"id":   "sv-new001",
				"type": "state-versions",
				"attributes": map[string]any{
					"serial":                  int64(3),
					"created-at":              time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					"status":                  "pending",
					"hosted-state-upload-url": uploadSrv.URL + "/upload",
				},
			},
		})
	})

	c := newTestClient(t, mux)
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
