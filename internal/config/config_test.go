package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unsetenv は指定した環境変数を削除し、テスト終了時に元に戻す。
func unsetenv(t *testing.T, key string) {
	t.Helper()
	orig, exists := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("os.Unsetenv(%s): %v", key, err)
	}
	if exists {
		t.Cleanup(func() { _ = os.Setenv(key, orig) })
	} else {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}
}

func writeEnvFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0600); err != nil {
		t.Fatalf(".env 作成失敗: %v", err)
	}
}

// --- TestLoad ---

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envs    map[string]string
		wantErr []string
	}{
		{
			name: "全環境変数が設定済み",
			envs: map[string]string{
				"TFC_API_TOKEN":      "mytoken",
				"TFC_ORGANIZATION":   "myorg",
				"TFC_WORKSPACE_NAME": "myws",
			},
		},
		{
			name: "TFC_API_TOKEN が未設定",
			envs: map[string]string{
				"TFC_ORGANIZATION":   "myorg",
				"TFC_WORKSPACE_NAME": "myws",
			},
			wantErr: []string{"TFC_API_TOKEN"},
		},
		{
			name: "TFC_ORGANIZATION が未設定",
			envs: map[string]string{
				"TFC_API_TOKEN":      "mytoken",
				"TFC_WORKSPACE_NAME": "myws",
			},
			wantErr: []string{"TFC_ORGANIZATION"},
		},
		{
			name: "TFC_WORKSPACE_NAME が未設定",
			envs: map[string]string{
				"TFC_API_TOKEN":    "mytoken",
				"TFC_ORGANIZATION": "myorg",
			},
			wantErr: []string{"TFC_WORKSPACE_NAME"},
		},
		{
			name:    "全環境変数が未設定",
			envs:    map[string]string{},
			wantErr: []string{"TFC_API_TOKEN", "TFC_ORGANIZATION", "TFC_WORKSPACE_NAME"},
		},
	}

	allKeys := []string{"TFC_API_TOKEN", "TFC_ORGANIZATION", "TFC_WORKSPACE_NAME"}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// .env を誤って拾わないよう空の一時ディレクトリをカレントにする
			t.Chdir(t.TempDir())

			for _, k := range allKeys {
				t.Setenv(k, "")
			}
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			cfg, err := Load()

			if len(tc.wantErr) == 0 {
				if err != nil {
					t.Fatalf("エラーが予期されていないのに発生: %v", err)
				}
				if cfg.APIToken != tc.envs["TFC_API_TOKEN"] {
					t.Errorf("APIToken: got %q, want %q", cfg.APIToken, tc.envs["TFC_API_TOKEN"])
				}
				if cfg.Organization != tc.envs["TFC_ORGANIZATION"] {
					t.Errorf("Organization: got %q, want %q", cfg.Organization, tc.envs["TFC_ORGANIZATION"])
				}
				if cfg.WorkspaceName != tc.envs["TFC_WORKSPACE_NAME"] {
					t.Errorf("WorkspaceName: got %q, want %q", cfg.WorkspaceName, tc.envs["TFC_WORKSPACE_NAME"])
				}
			} else {
				if err == nil {
					t.Fatal("エラーが期待されたが発生しなかった")
				}
				for _, wantMsg := range tc.wantErr {
					if !strings.Contains(err.Error(), wantMsg) {
						t.Errorf("エラーに %q が含まれていない: %v", wantMsg, err)
					}
				}
			}
		})
	}
}

// --- TestLoadDotEnv_* ---

func TestLoadDotEnv_CurrentDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeEnvFile(t, dir, "TFC_API_TOKEN=envtoken\nTFC_ORGANIZATION=envorg\nTFC_WORKSPACE_NAME=envws\n")

	unsetenv(t, "TFC_API_TOKEN")
	unsetenv(t, "TFC_ORGANIZATION")
	unsetenv(t, "TFC_WORKSPACE_NAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if cfg.APIToken != "envtoken" {
		t.Errorf("APIToken: got %q, want %q", cfg.APIToken, "envtoken")
	}
	if cfg.Organization != "envorg" {
		t.Errorf("Organization: got %q, want %q", cfg.Organization, "envorg")
	}
	if cfg.WorkspaceName != "envws" {
		t.Errorf("WorkspaceName: got %q, want %q", cfg.WorkspaceName, "envws")
	}
}

func TestLoadDotEnv_ParentDir(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Chdir(child)
	writeEnvFile(t, parent, "TFC_API_TOKEN=parenttoken\nTFC_ORGANIZATION=parentorg\nTFC_WORKSPACE_NAME=parentws\n")

	unsetenv(t, "TFC_API_TOKEN")
	unsetenv(t, "TFC_ORGANIZATION")
	unsetenv(t, "TFC_WORKSPACE_NAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if cfg.APIToken != "parenttoken" {
		t.Errorf("APIToken: got %q, want %q", cfg.APIToken, "parenttoken")
	}
}

func TestLoadDotEnv_EnvVarTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeEnvFile(t, dir, "TFC_API_TOKEN=fromfile\nTFC_ORGANIZATION=fileorg\nTFC_WORKSPACE_NAME=filews\n")

	// 環境変数を先にセット → .env より優先されるはず
	t.Setenv("TFC_API_TOKEN", "fromenv")
	unsetenv(t, "TFC_ORGANIZATION")
	unsetenv(t, "TFC_WORKSPACE_NAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if cfg.APIToken != "fromenv" {
		t.Errorf("APIToken: got %q, want %q (環境変数が優先されるべき)", cfg.APIToken, "fromenv")
	}
	if cfg.Organization != "fileorg" {
		t.Errorf("Organization: got %q, want %q (.env から読み込まれるべき)", cfg.Organization, "fileorg")
	}
}

func TestLoadDotEnv_Quotes(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeEnvFile(t, dir, `TFC_API_TOKEN="double-quoted"
TFC_ORGANIZATION='single-quoted'
TFC_WORKSPACE_NAME=noquote
`)

	unsetenv(t, "TFC_API_TOKEN")
	unsetenv(t, "TFC_ORGANIZATION")
	unsetenv(t, "TFC_WORKSPACE_NAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if cfg.APIToken != "double-quoted" {
		t.Errorf("APIToken: got %q, want %q", cfg.APIToken, "double-quoted")
	}
	if cfg.Organization != "single-quoted" {
		t.Errorf("Organization: got %q, want %q", cfg.Organization, "single-quoted")
	}
	if cfg.WorkspaceName != "noquote" {
		t.Errorf("WorkspaceName: got %q, want %q", cfg.WorkspaceName, "noquote")
	}
}

func TestLoadDotEnv_ExportKeyword(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeEnvFile(t, dir, "export TFC_API_TOKEN=exporttoken\nexport TFC_ORGANIZATION=exportorg\nexport TFC_WORKSPACE_NAME=exportws\n")

	unsetenv(t, "TFC_API_TOKEN")
	unsetenv(t, "TFC_ORGANIZATION")
	unsetenv(t, "TFC_WORKSPACE_NAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if cfg.APIToken != "exporttoken" {
		t.Errorf("APIToken: got %q, want %q", cfg.APIToken, "exporttoken")
	}
}

func TestLoadDotEnv_Comments(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeEnvFile(t, dir, `# このファイルは設定用です
TFC_API_TOKEN=commenttest

# 空行の後のコメント
TFC_ORGANIZATION=commentorg
TFC_WORKSPACE_NAME=commentws
`)

	unsetenv(t, "TFC_API_TOKEN")
	unsetenv(t, "TFC_ORGANIZATION")
	unsetenv(t, "TFC_WORKSPACE_NAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if cfg.APIToken != "commenttest" {
		t.Errorf("APIToken: got %q, want %q", cfg.APIToken, "commenttest")
	}
}

func TestLoadDotEnv_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeEnvFile(t, dir, "INVALID_LINE_WITHOUT_EQUALS\n")

	unsetenv(t, "TFC_API_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("エラーが期待されたが発生しなかった")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("エラーに 'invalid format' が含まれていない: %v", err)
	}
}

func TestLoadDotEnv_NoFile(t *testing.T) {
	// .env が存在しない一時ディレクトリ → エラーにならない
	dir := t.TempDir()
	t.Chdir(dir)

	unsetenv(t, "TFC_API_TOKEN")
	unsetenv(t, "TFC_ORGANIZATION")
	unsetenv(t, "TFC_WORKSPACE_NAME")

	_, err := Load()
	// 環境変数未設定なのでエラーになるが、.env 読み込みに起因しないこと
	if err != nil && strings.Contains(err.Error(), ".env") {
		t.Errorf(".env に関するエラーが発生した（.env が存在しないのに）: %v", err)
	}
}
