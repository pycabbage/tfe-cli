package config

import (
	"errors"
	"os"
)

type Config struct {
	APIToken      string
	Organization  string
	WorkspaceName string
}

func Load() (*Config, error) {
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
