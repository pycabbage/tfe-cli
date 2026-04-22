package tfc

import (
	"net/http"
	"strings"
	"testing"
)

func TestLock_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/lock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("メソッド: got %s, want POST", r.Method)
		}
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	c := newTestClient(t, mux)

	// キャッシュを事前に設定してワークスペース取得のリクエストを減らす
	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("前処理エラー: %v", err)
	}

	if err := c.Lock(t.Context(), "test reason"); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	// Lock後にキャッシュがクリアされること
	if c.workspace != nil {
		t.Error("Lock後にキャッシュがクリアされていない")
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
		t.Fatalf("前処理エラー: %v", err)
	}

	err := c.Lock(t.Context(), "test")
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "already locked") {
		t.Errorf("エラーメッセージに 'already locked' が含まれていない: %v", err)
	}
}

func TestUnlock_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	mux.HandleFunc("/workspaces/ws-001/actions/unlock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("メソッド: got %s, want POST", r.Method)
		}
		writeJSON(w, 200, workspaceResponse("ws-001"))
	})
	c := newTestClient(t, mux)

	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("前処理エラー: %v", err)
	}

	if err := c.Unlock(t.Context()); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	// Unlock後にキャッシュがクリアされること
	if c.workspace != nil {
		t.Error("Unlock後にキャッシュがクリアされていない")
	}
}

func TestGetAccountDetails_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/account/details", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"data": map[string]any{
				"id": "user-123",
				"attributes": map[string]any{
					"username": "alice",
					"email":    "alice@example.com",
					"two-factor": map[string]any{
						"enabled": true,
					},
				},
			},
		})
	})
	c := newTestClient(t, mux)

	details, err := c.GetAccountDetails(t.Context())
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if details.Data.Attributes.Username != "alice" {
		t.Errorf("Username: got %q, want %q", details.Data.Attributes.Username, "alice")
	}
	if details.Data.Attributes.Email != "alice@example.com" {
		t.Errorf("Email: got %q, want %q", details.Data.Attributes.Email, "alice@example.com")
	}
	if !details.Data.Attributes.TwoFactor.Enabled {
		t.Error("TwoFactor.Enabled: got false, want true")
	}
}

func TestGetAccountDetails_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/account/details", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, map[string]any{
			"errors": []map[string]any{
				{"status": "401", "title": "Unauthorized"},
			},
		})
	})
	c := newTestClient(t, mux)

	_, err := c.GetAccountDetails(t.Context())
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "getting account details") {
		t.Errorf("エラーメッセージに 'getting account details' が含まれていない: %v", err)
	}
}
