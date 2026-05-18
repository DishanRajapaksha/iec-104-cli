package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "missing.yaml"), nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Endpoint != "" {
		t.Fatalf("Endpoint = %q, want empty", cfg.Endpoint)
	}
}

func TestLoadFileWithOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(`endpoint: http://host
username: user
password: pass
timeout: 10s
`), 0o600)

	cfg, err := Load(path, map[string]string{"endpoint": "override", "timeout": "20s"})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Endpoint != "override" {
		t.Fatalf("Endpoint = %q, want override", cfg.Endpoint)
	}
	if cfg.Username != "user" {
		t.Fatalf("Username = %q, want user", cfg.Username)
	}
	if cfg.Timeout.Duration() != 20*time.Second {
		t.Fatalf("Timeout = %s, want 20s", cfg.Timeout.Duration())
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(`timeout: bananas`), 0o600)

	_, err := Load(path, nil)
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}