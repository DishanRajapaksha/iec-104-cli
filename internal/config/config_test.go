package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, found, err := Load(filepath.Join(dir, "missing.yaml"), Overrides{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if found {
		t.Fatal("found = true, want false")
	}
	if cfg.Connection.Port != 2404 {
		t.Fatalf("port = %d, want 2404", cfg.Connection.Port)
	}
	if cfg.Output.Format != DefaultFormat {
		t.Fatalf("format = %q, want %q", cfg.Output.Format, DefaultFormat)
	}
}

func TestLoadFileWithOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`connection:
  host: 192.0.2.10
  port: 2405
  timeout: 12s
  reconnect: true
  reconnect_interval: 3s
iec104:
  common_address: 7
  originator_address: 1
  interrogation_qualifier: 20
output:
  format: table
  timestamps: utc
points:
  - name: active_power
    ioa: 1001
    type: float
    unit: MW
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	host := "127.0.0.1"
	timeout := 20 * time.Second
	format := "json"
	cfg, found, err := Load(path, Overrides{
		Host:         &host,
		Timeout:      &timeout,
		OutputFormat: &format,
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if cfg.Connection.Host != "127.0.0.1" {
		t.Fatalf("host = %q, want override", cfg.Connection.Host)
	}
	if cfg.Connection.Port != 2405 {
		t.Fatalf("port = %d, want 2405", cfg.Connection.Port)
	}
	if cfg.Connection.Timeout.Duration() != 20*time.Second {
		t.Fatalf("timeout = %s, want 20s", cfg.Connection.Timeout.Duration())
	}
	if cfg.Output.Format != "json" {
		t.Fatalf("format = %q, want json", cfg.Output.Format)
	}
	if len(cfg.Points) != 1 || cfg.Points[0].Name != "active_power" {
		t.Fatalf("points = %#v", cfg.Points)
	}
}

func TestLoadRequiredMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadRequired(filepath.Join(dir, "missing.yaml"), Overrides{})
	if err == nil {
		t.Fatal("expected missing config error")
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`connection:
  timeout: bananas
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = Load(path, Overrides{})
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
