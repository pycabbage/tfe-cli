package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var configDirForTest string

type Profile struct {
	APIToken     string `yaml:"api_token"`
	Organization string `yaml:"organization"`
	Workspace    string `yaml:"workspace"`
}

type Store struct {
	CurrentProfile string              `yaml:"current-profile"`
	Profiles       map[string]*Profile `yaml:"profiles"`
}

func ConfigDir() (string, error) {
	if configDirForTest != "" {
		return configDirForTest, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config dir: %w", err)
	}
	return filepath.Join(dir, "tfe"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func LoadStore() (*Store, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{Profiles: make(map[string]*Profile)}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var s Store
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if s.Profiles == nil {
		s.Profiles = make(map[string]*Profile)
	}
	return &s, nil
}

func (s *Store) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

func (s *Store) ActiveProfile() *Profile {
	if s.CurrentProfile == "" {
		return nil
	}
	return s.Profiles[s.CurrentProfile]
}

func (s *Store) SetProfile(name string, p *Profile) {
	s.Profiles[name] = p
	s.CurrentProfile = name
}

func (s *Store) DeleteProfile(name string) error {
	if _, ok := s.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(s.Profiles, name)
	if s.CurrentProfile == name {
		for k := range s.Profiles {
			s.CurrentProfile = k
			return nil
		}
		s.CurrentProfile = ""
	}
	return nil
}

func (p *Profile) ToConfig() *Config {
	return &Config{
		APIToken:      p.APIToken,
		Organization:  p.Organization,
		WorkspaceName: p.Workspace,
	}
}
