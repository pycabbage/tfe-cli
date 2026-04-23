package tfc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/pycabbage/tfe-cli/internal/config"
)

func newTestClient(t *testing.T, mux *http.ServeMux) *Client {
	t.Helper()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	srv := httptest.NewServer(http.StripPrefix("/api/v2", mux))
	t.Cleanup(srv.Close)
	tfeClient, err := tfe.NewClient(&tfe.Config{
		Token:      "test-token",
		Address:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("tfe client: %v", err)
	}
	return &Client{
		tfe:          tfeClient,
		cfg:          &config.Config{Organization: "myorg", WorkspaceName: "myws"},
		pollInterval: 1 * time.Millisecond,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func workspaceResponse(id string) any {
	return map[string]any{
		"data": map[string]any{
			"id":   id,
			"type": "workspaces",
			"attributes": map[string]any{
				"name":   "myws",
				"locked": false,
			},
		},
	}
}

func TestGetWorkspace_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-abc123"))
	})
	c := newTestClient(t, mux)

	ws, err := c.GetWorkspace(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.ID != "ws-abc123" {
		t.Errorf("Workspace ID: got %q, want %q", ws.ID, "ws-abc123")
	}
}

func TestGetWorkspace_Caching(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		writeJSON(w, 200, workspaceResponse("ws-cached"))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if callCount != 1 {
		t.Errorf("HTTP request count: got %d, want 1 (second call should use cache)", callCount)
	}
}

func TestGetWorkspace_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]any{
			"errors": []map[string]any{
				{"status": "404", "title": "Not Found"},
			},
		})
	})
	c := newTestClient(t, mux)

	_, err := c.GetWorkspace(t.Context())
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
