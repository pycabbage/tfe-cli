package tfc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pycabbage/tfe-cli/internal/config"
)

func newTestClient(t *testing.T, mux *http.ServeMux) *Client {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := New(&config.Config{
		APIToken:      "test-token",
		Organization:  "myorg",
		WorkspaceName: "myws",
	})
	c.baseURL = srv.URL
	c.http = srv.Client()
	return c
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func workspaceResponse(id string) any {
	return map[string]any{
		"data": map[string]any{
			"id": id,
			"attributes": map[string]any{
				"name": "myws",
			},
		},
	}
}

// --- parseAPIError ---

func TestParseAPIError_WithDetail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 422, map[string]any{
			"errors": []map[string]any{
				{"status": "422", "title": "Unprocessable Entity", "detail": "invalid value"},
			},
		})
	})
	c := newTestClient(t, mux)
	err := c.get(t.Context(), "/test", nil)
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	want := "Unprocessable Entity: invalid value"
	if err.Error() != want {
		t.Errorf("エラーメッセージ: got %q, want %q", err.Error(), want)
	}
}

func TestParseAPIError_WithoutDetail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 404, map[string]any{
			"errors": []map[string]any{
				{"status": "404", "title": "Not Found"},
			},
		})
	})
	c := newTestClient(t, mux)
	err := c.get(t.Context(), "/test", nil)
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if err.Error() != "Not Found" {
		t.Errorf("エラーメッセージ: got %q, want %q", err.Error(), "Not Found")
	}
}

func TestParseAPIError_NonJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = fmt.Fprint(w, "internal server error")
	})
	c := newTestClient(t, mux)
	err := c.get(t.Context(), "/test", nil)
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if err.Error() != "HTTP 500" {
		t.Errorf("エラーメッセージ: got %q, want %q", err.Error(), "HTTP 500")
	}
}

// --- get ---

func TestGet_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("メソッド: got %s, want GET", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization ヘッダーが正しくない: %s", r.Header.Get("Authorization"))
		}
		writeJSON(w, 200, map[string]string{"key": "value"})
	})
	c := newTestClient(t, mux)

	var out map[string]string
	if err := c.get(t.Context(), "/data", &out); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if out["key"] != "value" {
		t.Errorf("レスポンス: got %v, want {key:value}", out)
	}
}

// --- post ---

func TestPost_SuccessNoOut(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("メソッド: got %s, want POST", r.Method)
		}
		w.WriteHeader(200)
	})
	c := newTestClient(t, mux)
	if err := c.post(t.Context(), "/action", nil, nil); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
}

func TestPost_SuccessWithOut(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 201, map[string]string{"id": "abc123"})
	})
	c := newTestClient(t, mux)

	var out map[string]string
	if err := c.post(t.Context(), "/create", nil, &out); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if out["id"] != "abc123" {
		t.Errorf("レスポンス id: got %q, want %q", out["id"], "abc123")
	}
}

// --- GetWorkspace ---

func TestGetWorkspace_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/organizations/myorg/workspaces/myws", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, workspaceResponse("ws-abc123"))
	})
	c := newTestClient(t, mux)

	ws, err := c.GetWorkspace(t.Context())
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
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
		t.Fatalf("1回目: %v", err)
	}
	if _, err := c.GetWorkspace(t.Context()); err != nil {
		t.Fatalf("2回目: %v", err)
	}

	if callCount != 1 {
		t.Errorf("HTTPリクエスト回数: got %d, want 1（2回目はキャッシュから取得）", callCount)
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
		t.Fatal("エラーが期待されたが発生しなかった")
	}
}
