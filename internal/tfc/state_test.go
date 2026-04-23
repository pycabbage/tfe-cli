package tfc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func stateVersionResponse(id string, serial int64) any {
	return map[string]any{
		"data": map[string]any{
			"id":   id,
			"type": "state-versions",
			"attributes": map[string]any{
				"serial":                    serial,
				"created-at":                time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				"status":                    "finalized",
				"hosted-state-download-url": "https://example.com/state/" + id,
				"terraform-version":         "1.5.0",
			},
		},
	}
}

func stateVersionListResponse(ids []string) any {
	data := make([]map[string]any, len(ids))
	for i, id := range ids {
		data[i] = map[string]any{
			"id":   id,
			"type": "state-versions",
			"attributes": map[string]any{
				"serial":     int64(i + 1),
				"created-at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				"status":     "finalized",
			},
		}
	}
	return map[string]any{
		"data": data,
		"meta": map[string]any{
			"pagination": map[string]any{
				"current-page": 1,
				"page-size":    10,
				"total-pages":  1,
				"total-count":  len(ids),
			},
		},
	}
}

func TestListStateVersions_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionListResponse([]string{"sv-001", "sv-002"}))
	})
	c := newTestClient(t, mux)

	versions, err := c.ListStateVersions(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("version count: got %d, want 2", len(versions))
	}
	if versions[0].ID != "sv-001" {
		t.Errorf("first ID: got %q, want %q", versions[0].ID, "sv-001")
	}
}

func TestGetLatestStateVersion_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/current-state-version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionResponse("sv-latest", 5))
	})
	c := newTestClient(t, mux)

	sv, err := c.GetLatestStateVersion(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sv.ID != "sv-latest" {
		t.Errorf("ID: got %q, want %q", sv.ID, "sv-latest")
	}
	if sv.Serial != 5 {
		t.Errorf("Serial: got %d, want 5", sv.Serial)
	}
}

func TestGetStateVersion_EmptyID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/current-state-version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, stateVersionResponse("sv-latest", 3))
	})
	c := newTestClient(t, mux)

	sv, err := c.GetStateVersion(t.Context(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sv.ID != "sv-latest" {
		t.Errorf("expected latest for empty ID: got %q", sv.ID)
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

	sv, err := c.GetStateVersion(t.Context(), "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sv.ID != "sv-latest" {
		t.Errorf("expected latest for 'latest' keyword: got %q", sv.ID)
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
		t.Fatalf("unexpected error: %v", err)
	}
	if sv.ID != "sv-abc123" {
		t.Errorf("ID: got %q, want %q", sv.ID, "sv-abc123")
	}
}

func TestDownloadState_Success(t *testing.T) {
	stateData := []byte(`{"version":4,"serial":1}`)
	downloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write(stateData)
	}))
	defer downloadSrv.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions/sv-dl001", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"data": map[string]any{
				"id":   "sv-dl001",
				"type": "state-versions",
				"attributes": map[string]any{
					"serial":                    int64(1),
					"created-at":                time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					"status":                    "finalized",
					"hosted-state-download-url": downloadSrv.URL + "/state",
				},
			},
		})
	})
	c := newTestClient(t, mux)

	sv, err := c.GetStateVersion(t.Context(), "sv-dl001")
	if err != nil {
		t.Fatalf("failed to get state version: %v", err)
	}
	data, err := c.DownloadState(t.Context(), sv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(stateData) {
		t.Errorf("data: got %q, want %q", string(data), string(stateData))
	}
}

func TestDownloadState_NoURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions/sv-nourl", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"data": map[string]any{
				"id":   "sv-nourl",
				"type": "state-versions",
				"attributes": map[string]any{
					"serial":     int64(1),
					"created-at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					"status":     "pending",
				},
			},
		})
	})
	c := newTestClient(t, mux)

	sv, err := c.GetStateVersion(t.Context(), "sv-nourl")
	if err != nil {
		t.Fatalf("failed to get state version: %v", err)
	}
	_, err = c.DownloadState(t.Context(), sv)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no download URL") {
		t.Errorf("error message: got %q", err.Error())
	}
}

func TestDownloadState_HTTPError(t *testing.T) {
	downloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer downloadSrv.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/state-versions/sv-err001", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"data": map[string]any{
				"id":   "sv-err001",
				"type": "state-versions",
				"attributes": map[string]any{
					"serial":                    int64(1),
					"created-at":                time.Now().UTC().Format("2006-01-02T15:04:05Z"),
					"status":                    "finalized",
					"hosted-state-download-url": downloadSrv.URL + "/state",
				},
			},
		})
	})
	c := newTestClient(t, mux)

	sv, err := c.GetStateVersion(t.Context(), "sv-err001")
	if err != nil {
		t.Fatalf("failed to get state version: %v", err)
	}
	_, err = c.DownloadState(t.Context(), sv)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "downloading state") {
		t.Errorf("error does not contain 'downloading state': %v", err)
	}
}
