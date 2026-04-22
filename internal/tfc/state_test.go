package tfc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func stateVersionResponse(id string, serial int64) any {
	return map[string]any{
		"data": map[string]any{
			"id": id,
			"attributes": map[string]any{
				"serial":                    serial,
				"created-at":                time.Now().Format(time.RFC3339),
				"status":                    "finalized",
				"hosted-state-download-url": "https://example.com/state/" + id,
				"hosted-state-upload-url":   "",
				"terraform-version":         "1.5.0",
				"lineage":                   "abc-lineage",
				"finalized":                 true,
			},
		},
	}
}

func stateVersionListResponse(ids []string) any {
	data := make([]map[string]any, len(ids))
	for i, id := range ids {
		data[i] = map[string]any{
			"id": id,
			"attributes": map[string]any{
				"serial":     int64(i + 1),
				"created-at": time.Now().Format(time.RFC3339),
				"status":     "finalized",
			},
		}
	}
	return map[string]any{"data": data}
}

// --- ListStateVersions ---

func TestListStateVersions_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/state-versions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionListResponse([]string{"sv-001", "sv-002"}))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("前処理エラー: %v", err)
	}

	versions, err := c.ListStateVersions(t.Context())
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("バージョン数: got %d, want 2", len(versions))
	}
	if versions[0].ID != "sv-001" {
		t.Errorf("1件目 ID: got %q, want %q", versions[0].ID, "sv-001")
	}
}

// --- GetLatestStateVersion ---

func TestGetLatestStateVersion_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/current-state-version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionResponse("sv-latest", 5))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("前処理エラー: %v", err)
	}

	sv, err := c.GetLatestStateVersion(t.Context())
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if sv.ID != "sv-latest" {
		t.Errorf("ID: got %q, want %q", sv.ID, "sv-latest")
	}
	if sv.Attributes.Serial != 5 {
		t.Errorf("Serial: got %d, want 5", sv.Attributes.Serial)
	}
}

// --- GetStateVersion ---

func TestGetStateVersion_EmptyID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/current-state-version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionResponse("sv-latest", 3))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("前処理エラー: %v", err)
	}

	sv, err := c.GetStateVersion(t.Context(), "")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if sv.ID != "sv-latest" {
		t.Errorf("空ID時に latest が返っていない: got %q", sv.ID)
	}
}

func TestGetStateVersion_LatestKeyword(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/current-state-version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionResponse("sv-latest", 3))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("前処理エラー: %v", err)
	}

	sv, err := c.GetStateVersion(t.Context(), "latest")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if sv.ID != "sv-latest" {
		t.Errorf("'latest' キーワード時に latest が返っていない: got %q", sv.ID)
	}
}

func TestGetStateVersion_SpecificID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions/sv-abc123", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionResponse("sv-abc123", 7))
	})
	c := newTestClient(t, mux)

	sv, err := c.GetStateVersion(t.Context(), "sv-abc123")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if sv.ID != "sv-abc123" {
		t.Errorf("ID: got %q, want %q", sv.ID, "sv-abc123")
	}
}

// --- DownloadState ---

func TestDownloadState_Success(t *testing.T) {
	stateData := []byte(`{"version":4,"serial":1}`)
	downloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write(stateData)
	}))
	defer downloadSrv.Close()

	sv := &StateVersion{
		ID: "sv-001",
		Attributes: StateVersionAttrs{
			DownloadURL: downloadSrv.URL + "/state",
		},
	}

	mux := http.NewServeMux()
	c := newTestClient(t, mux)
	c.http = downloadSrv.Client()

	data, err := c.DownloadState(t.Context(), sv)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if string(data) != string(stateData) {
		t.Errorf("データ: got %q, want %q", string(data), string(stateData))
	}
}

func TestDownloadState_NoURL(t *testing.T) {
	sv := &StateVersion{
		ID:         "sv-001",
		Attributes: StateVersionAttrs{},
	}

	mux := http.NewServeMux()
	c := newTestClient(t, mux)

	_, err := c.DownloadState(t.Context(), sv)
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "no download URL") {
		t.Errorf("エラーメッセージ: got %q", err.Error())
	}
}

func TestDownloadState_HTTPError(t *testing.T) {
	downloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer downloadSrv.Close()

	sv := &StateVersion{
		ID: "sv-001",
		Attributes: StateVersionAttrs{
			DownloadURL: downloadSrv.URL + "/state",
		},
	}

	mux := http.NewServeMux()
	c := newTestClient(t, mux)
	c.http = downloadSrv.Client()

	_, err := c.DownloadState(t.Context(), sv)
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", 404)) {
		t.Errorf("エラーに HTTP 404 が含まれていない: %v", err)
	}
}
