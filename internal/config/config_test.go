package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFileAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	writeConfig(t, path, `connection:
  host: 127.0.0.1
points:
  - name: active_power
    ioa: 1001
    type: float
    unit: MW
`)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if cfg.Connection.Host != "127.0.0.1" {
		t.Fatalf("Host = %q, want 127.0.0.1", cfg.Connection.Host)
	}
	if cfg.Connection.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Connection.Port, DefaultPort)
	}
	if cfg.Connection.Timeout.Duration() != 10*time.Second {
		t.Fatalf("Timeout = %s, want 10s", cfg.Connection.Timeout.Duration())
	}
	if cfg.IEC104.CommonAddress != 1 {
		t.Fatalf("CommonAddress = %d, want 1", cfg.IEC104.CommonAddress)
	}
	if cfg.Output.Format != DefaultFormat {
		t.Fatalf("Format = %q, want %q", cfg.Output.Format, DefaultFormat)
	}
	if len(cfg.Points) != 1 {
		t.Fatalf("len(Points) = %d, want 1", len(cfg.Points))
	}
}

func TestLoadOptionalMissingFile(t *testing.T) {
	cfg, loaded, err := LoadOptional(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("LoadOptional returned error: %v", err)
	}
	if loaded {
		t.Fatalf("loaded = true, want false")
	}
	if cfg.Connection.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Connection.Port, DefaultPort)
	}
}

func TestApplyProfile(t *testing.T) {
	cfg := Defaults()
	cfg.Connection.Host = "base"
	cfg.Profiles = map[string]Profile{
		"site-a": {
			Connection: &ConnectionConfig{Host: "site-a", Port: 2405},
			IEC104:     &IEC104Config{CommonAddress: 12},
			Output:     &OutputConfig{Format: "json"},
			Points: []PointConfig{{
				Name: "breaker",
				IOA:  2001,
				Type: "single_point",
			}},
		},
	}

	got, err := ApplyProfile(cfg, "site-a")
	if err != nil {
		t.Fatalf("ApplyProfile returned error: %v", err)
	}
	if got.Connection.Host != "site-a" {
		t.Fatalf("Host = %q, want site-a", got.Connection.Host)
	}
	if got.Connection.Port != 2405 {
		t.Fatalf("Port = %d, want 2405", got.Connection.Port)
	}
	if got.IEC104.CommonAddress != 12 {
		t.Fatalf("CommonAddress = %d, want 12", got.IEC104.CommonAddress)
	}
	if got.Output.Format != "json" {
		t.Fatalf("Format = %q, want json", got.Output.Format)
	}
	if len(got.Points) != 1 || got.Points[0].Name != "breaker" {
		t.Fatalf("Points = %#v, want breaker point", got.Points)
	}
}

func TestApplyProfileMissing(t *testing.T) {
	_, err := ApplyProfile(Defaults(), "missing")
	if err == nil {
		t.Fatal("ApplyProfile returned nil error for missing profile")
	}
}

func TestApplyOverrides(t *testing.T) {
	cfg := Defaults()
	cfg = ApplyOverrides(cfg, Overrides{
		Host:    "override",
		Port:    2406,
		Timeout: 30 * time.Second,
		Format:  "jsonl",
	})

	if cfg.Connection.Host != "override" {
		t.Fatalf("Host = %q, want override", cfg.Connection.Host)
	}
	if cfg.Connection.Port != 2406 {
		t.Fatalf("Port = %d, want 2406", cfg.Connection.Port)
	}
	if cfg.Connection.Timeout.Duration() != 30*time.Second {
		t.Fatalf("Timeout = %s, want 30s", cfg.Connection.Timeout.Duration())
	}
	if cfg.Output.Format != "jsonl" {
		t.Fatalf("Format = %q, want jsonl", cfg.Output.Format)
	}
}

func TestLoadWithProfileAndOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	writeConfig(t, path, `connection:
  host: base
profiles:
  site-a:
    connection:
      host: site-a
      port: 2405
`)

	cfg, loaded, err := Load(path, Overrides{
		Profile: "site-a",
		Host:    "cli-host",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !loaded {
		t.Fatal("loaded = false, want true")
	}
	if cfg.Connection.Host != "cli-host" {
		t.Fatalf("Host = %q, want cli-host", cfg.Connection.Host)
	}
	if cfg.Connection.Port != 2405 {
		t.Fatalf("Port = %d, want 2405", cfg.Connection.Port)
	}
}

func TestDurationUnmarshalRejectsInvalidValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	writeConfig(t, path, `connection:
  host: 127.0.0.1
  timeout: bananas
`)

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile returned nil error for invalid duration")
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg := Defaults()
	cfg.Connection.Host = "127.0.0.1"
	cfg.Points = []PointConfig{{Name: "active_power", IOA: 1001, Type: "float"}}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRejectsInvalidConfig(t *testing.T) {
	cfg := Defaults()
	cfg.Connection.Host = ""
	cfg.Connection.Port = 70000
	cfg.Connection.Timeout = NewDuration(-time.Second)
	cfg.Output.Format = "xml"
	cfg.Output.Timestamps = "mars"
	cfg.Points = []PointConfig{
		{Name: "dup", IOA: 0, Type: "float"},
		{Name: "dup", IOA: MaxIOA + 1, Type: "bad"},
		{Name: "", IOA: 1, Type: "single_point"},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate returned nil error for invalid config")
	}

	message := err.Error()
	wantFragments := []string{
		"connection.host is required",
		"connection.port must be between 1 and 65535",
		"connection.timeout must be positive",
		"output.format must be one of table, text, json, jsonl",
		"output.timestamps must be local or utc",
		"point name \"dup\" is duplicated",
		"points[0].ioa must be between 1 and 16777215",
		"points[1].type must be one of",
		"points[2].name is required",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(message, fragment) {
			t.Fatalf("Validate error %q does not contain %q", message, fragment)
		}
	}
}

func writeConfig(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
