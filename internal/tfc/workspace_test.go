package tfc

import (
	"net/http"
	"strings"
	"testing"
)

func userResponse() any {
	return map[string]any{
		"data": map[string]any{
			"id":   "user-123",
			"type": "users",
			"attributes": map[string]any{
				"username": "alice",
				"email":    "alice@example.com",
				"two-factor": map[string]any{
					"enabled": true,
				},
			},
		},
	}
}

func TestLock_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/lock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("setup error: %v", err)
	}
	if err := c.Lock(t.Context(), "test reason"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.workspace != nil {
		t.Error("cache not cleared after Lock")
	}
}

func TestLock_AlreadyLocked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/lock", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 409, map[string]any{
			"errors": []map[string]any{
				{"status": "409", "title": "Workspace already locked"},
			},
		})
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("setup error: %v", err)
	}
	err := c.Lock(t.Context(), "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already locked") {
		t.Errorf("error does not contain 'already locked': %v", err)
	}
}

func TestUnlock_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/unlock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("setup error: %v", err)
	}
	if err := c.Unlock(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.workspace != nil {
		t.Error("cache not cleared after Unlock")
	}
}

func TestGetCurrentUser_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/account/details", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, userResponse())
	})
	c := newTestClient(t, mux)

	user, err := c.GetCurrentUser(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("Username: got %q, want %q", user.Username, "alice")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email: got %q, want %q", user.Email, "alice@example.com")
	}
	if user.TwoFactor == nil || !user.TwoFactor.Enabled {
		t.Error("TwoFactor.Enabled: got false, want true")
	}
}

func TestGetCurrentUser_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/account/details", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, map[string]any{
			"errors": []map[string]any{
				{"status": "401", "title": "Unauthorized"},
			},
		})
	})
	c := newTestClient(t, mux)

	_, err := c.GetCurrentUser(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting current user") {
		t.Errorf("error does not contain 'getting current user': %v", err)
	}
}
