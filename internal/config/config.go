package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIToken      string
	Organization  string
	WorkspaceName string
}

func Load() (*Config, error) {
	if path, ok := findDotEnv(); ok {
		if err := loadDotEnv(path); err != nil {
			return nil, err
		}
	}
	cfg := &Config{
		APIToken:      os.Getenv("TFC_API_TOKEN"),
		Organization:  os.Getenv("TFC_ORGANIZATION"),
		WorkspaceName: os.Getenv("TFC_WORKSPACE_NAME"),
	}
	var errs []error
	if cfg.APIToken == "" {
		errs = append(errs, errors.New("TFC_API_TOKEN is not set"))
	}
	if cfg.Organization == "" {
		errs = append(errs, errors.New("TFC_ORGANIZATION is not set"))
	}
	if cfg.WorkspaceName == "" {
		errs = append(errs, errors.New("TFC_WORKSPACE_NAME is not set"))
	}
	return cfg, errors.Join(errs...)
}

// findDotEnv はカレントディレクトリから上位に向かって .env ファイルを探す。
func findDotEnv() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// loadDotEnv は .env ファイルをパースし、未設定の環境変数をセットする。
// 既存の環境変数（空文字含む）は上書きしない。
//
// サポート書式:
//   - KEY=VALUE
//   - KEY="VALUE" / KEY='VALUE'
//   - export KEY=VALUE
//   - # コメント行 / 空行
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading .env file: %w", err)
	}
	for i, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf(".env:%d: invalid format", i+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if q := value[0]; (q == '"' || q == '\'') && value[len(value)-1] == q {
				value = value[1 : len(value)-1]
			}
		}
		if key == "" {
			return fmt.Errorf(".env:%d: empty key", i+1)
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("setting %s: %w", key, err)
			}
		}
	}
	return nil
}
