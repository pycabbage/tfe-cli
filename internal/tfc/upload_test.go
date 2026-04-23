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
		t.Fatalf("creating temp file: %v", err)
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
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading state file") {
		t.Errorf("error does not contain 'reading state file': %v", err)
	}
}

func TestUploadState_Success(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT method: got %s", r.Method)
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
		t.Fatalf("unexpected error: %v", err)
	}
	if *pollCount < 2 {
		t.Errorf("poll count: got %d, want >= 2", *pollCount)
	}
	if !unlockCalled {
		t.Error("Unlock was not called")
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
		t.Fatal("expected error, got nil")
	}
	if unlockCalled {
		t.Error("Unlock was called despite Lock failure")
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
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "creating state version") {
		t.Errorf("error does not contain 'creating state version': %v", err)
	}
	if !unlockCalled {
		t.Error("Unlock was not called after Lock error (defer)")
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
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error is not context.DeadlineExceeded: %v", err)
	}
	if !unlockCalled {
		t.Error("Unlock was not called after timeout (defer)")
	}
}
