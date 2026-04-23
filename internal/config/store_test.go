package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	origDir := configDirForTest
	configDirForTest = dir
	t.Cleanup(func() {
		configDirForTest = origDir
	})
	return dir
}

func TestLoadStore_ReturnsEmptyStoreWhenFileNotExists(t *testing.T) {
	setupTestStore(t)

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if store.CurrentProfile != "" {
		t.Errorf("CurrentProfile = %q, want empty", store.CurrentProfile)
	}
	if len(store.Profiles) != 0 {
		t.Errorf("Profiles = %v, want empty map", store.Profiles)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := setupTestStore(t)

	store := &Store{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {
				APIToken:     "test-token",
				Organization: "test-org",
				Workspace:    "test-ws",
			},
		},
	}

	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path := filepath.Join(dir, "config.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permission = %o, want 0600", info.Mode().Perm())
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if loaded.CurrentProfile != "default" {
		t.Errorf("CurrentProfile = %q, want %q", loaded.CurrentProfile, "default")
	}
	p := loaded.Profiles["default"]
	if p == nil {
		t.Fatal("default profile is nil")
	}
	if p.APIToken != "test-token" {
		t.Errorf("APIToken = %q, want %q", p.APIToken, "test-token")
	}
	if p.Organization != "test-org" {
		t.Errorf("Organization = %q, want %q", p.Organization, "test-org")
	}
	if p.Workspace != "test-ws" {
		t.Errorf("Workspace = %q, want %q", p.Workspace, "test-ws")
	}
}

func TestActiveProfile_ReturnsNilWhenNoneSelected(t *testing.T) {
	store := &Store{Profiles: map[string]*Profile{}}
	if p := store.ActiveProfile(); p != nil {
		t.Errorf("ActiveProfile() = %v, want nil", p)
	}
}

func TestActiveProfile_ReturnsProfileWhenSelected(t *testing.T) {
	expected := &Profile{APIToken: "tok"}
	store := &Store{
		CurrentProfile: "staging",
		Profiles: map[string]*Profile{
			"staging": expected,
		},
	}
	if p := store.ActiveProfile(); p != expected {
		t.Errorf("ActiveProfile() = %v, want %v", p, expected)
	}
}

func TestSetProfile_AddsProfileAndUpdatesCurrent(t *testing.T) {
	store := &Store{Profiles: map[string]*Profile{}}
	p := &Profile{APIToken: "tok", Organization: "org", Workspace: "ws"}
	store.SetProfile("myprofile", p)

	if store.CurrentProfile != "myprofile" {
		t.Errorf("CurrentProfile = %q, want %q", store.CurrentProfile, "myprofile")
	}
	if store.Profiles["myprofile"] != p {
		t.Error("profile not stored correctly")
	}
}

func TestDeleteProfile_RemovesExistingProfile(t *testing.T) {
	store := &Store{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {APIToken: "a"},
			"staging": {APIToken: "b"},
		},
	}

	if err := store.DeleteProfile("default"); err != nil {
		t.Fatalf("DeleteProfile() error: %v", err)
	}
	if _, ok := store.Profiles["default"]; ok {
		t.Error("default profile should be deleted")
	}
	if store.CurrentProfile != "staging" {
		t.Errorf("CurrentProfile = %q, want %q", store.CurrentProfile, "staging")
	}
}

func TestDeleteProfile_ClearsCurrentWhenLastProfile(t *testing.T) {
	store := &Store{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {APIToken: "a"},
		},
	}

	if err := store.DeleteProfile("default"); err != nil {
		t.Fatalf("DeleteProfile() error: %v", err)
	}
	if store.CurrentProfile != "" {
		t.Errorf("CurrentProfile = %q, want empty", store.CurrentProfile)
	}
}

func TestDeleteProfile_ErrorForNonExistentProfile(t *testing.T) {
	store := &Store{Profiles: map[string]*Profile{}}
	if err := store.DeleteProfile("missing"); err == nil {
		t.Error("expected error for missing profile")
	}
}

func TestToConfig_ConvertsProfileToConfig(t *testing.T) {
	p := &Profile{APIToken: "tok", Organization: "org", Workspace: "ws"}
	cfg := p.ToConfig()
	if cfg.APIToken != "tok" {
		t.Errorf("APIToken = %q, want %q", cfg.APIToken, "tok")
	}
	if cfg.Organization != "org" {
		t.Errorf("Organization = %q, want %q", cfg.Organization, "org")
	}
	if cfg.WorkspaceName != "ws" {
		t.Errorf("WorkspaceName = %q, want %q", cfg.WorkspaceName, "ws")
	}
}
