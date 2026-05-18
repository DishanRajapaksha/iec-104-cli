package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
)

func TestRunHelp(t *testing.T) {
	if got := Run([]string{"help"}); got != exitcode.Success {
		t.Fatalf("Run(help) = %d, want %d", got, exitcode.Success)
	}
}

func TestRunHelpWithGlobalFormat(t *testing.T) {
	if got := Run([]string{"--format", "json", "help"}); got != exitcode.Success {
		t.Fatalf("Run(--format json help) = %d, want %d", got, exitcode.Success)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if got := Run([]string{"bogus"}); got != exitcode.GeneralError {
		t.Fatalf("Run(bogus) = %d, want %d", got, exitcode.GeneralError)
	}
}

func TestRunInvalidFormat(t *testing.T) {
	if got := Run([]string{"--format", "nonsense", "help"}); got != exitcode.ConfigError {
		t.Fatalf("Run(--format nonsense help) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunValidateConfig(t *testing.T) {
	path := writeCLIConfig(t, `connection:
  host: 127.0.0.1
points:
  - name: active_power
    ioa: 1001
    type: float
`)

	if got := Run([]string{"validate-config", "--config", path}); got != exitcode.Success {
		t.Fatalf("Run(validate-config --config path) = %d, want %d", got, exitcode.Success)
	}
}

func TestRunValidateConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	if got := Run([]string{"validate-config", "--config", path}); got != exitcode.ConfigError {
		t.Fatalf("Run(validate-config --config missing) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunValidateConfigInvalidFile(t *testing.T) {
	path := writeCLIConfig(t, `connection:
  host: ""
  port: 70000
points:
  - name: bad
    ioa: 0
    type: unknown
`)

	if got := Run([]string{"--config", path, "validate-config"}); got != exitcode.ConfigError {
		t.Fatalf("Run(--config path validate-config) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestParseGlobalOptionsDefaults(t *testing.T) {
	opts, rest, err := parseGlobalOptions([]string{"help"})
	if err != nil {
		t.Fatalf("parseGlobalOptions returned error: %v", err)
	}
	if opts.ConfigPath != defaultConfigPath {
		t.Fatalf("ConfigPath = %q, want %q", opts.ConfigPath, defaultConfigPath)
	}
	if opts.Format != defaultFormat {
		t.Fatalf("Format = %q, want %q", opts.Format, defaultFormat)
	}
	if opts.FormatSet {
		t.Fatalf("FormatSet = true, want false")
	}
	if opts.TimeoutSet {
		t.Fatalf("TimeoutSet = true, want false")
	}
	if len(rest) != 1 || rest[0] != "help" {
		t.Fatalf("rest = %#v, want [help]", rest)
	}
}

func TestParseGlobalOptionsValues(t *testing.T) {
	opts, rest, err := parseGlobalOptions([]string{
		"--config", "site.yaml",
		"--profile=plant-a",
		"--format", "jsonl",
		"--timeout", "15s",
		"--verbose",
		"--debug",
		"listen",
	})
	if err != nil {
		t.Fatalf("parseGlobalOptions returned error: %v", err)
	}
	if opts.ConfigPath != "site.yaml" {
		t.Fatalf("ConfigPath = %q, want site.yaml", opts.ConfigPath)
	}
	if opts.Profile != "plant-a" {
		t.Fatalf("Profile = %q, want plant-a", opts.Profile)
	}
	if opts.Format != "jsonl" {
		t.Fatalf("Format = %q, want jsonl", opts.Format)
	}
	if !opts.FormatSet {
		t.Fatalf("FormatSet = false, want true")
	}
	if opts.Timeout != 15*time.Second {
		t.Fatalf("Timeout = %s, want 15s", opts.Timeout)
	}
	if !opts.TimeoutSet {
		t.Fatalf("TimeoutSet = false, want true")
	}
	if !opts.Verbose {
		t.Fatalf("Verbose = false, want true")
	}
	if !opts.Debug {
		t.Fatalf("Debug = false, want true")
	}
	if len(rest) != 1 || rest[0] != "listen" {
		t.Fatalf("rest = %#v, want [listen]", rest)
	}
}

func TestParseGlobalOptionsAfterCommand(t *testing.T) {
	opts, rest, err := parseGlobalOptions([]string{"validate-config", "--config", "site.yaml", "--format", "json"})
	if err != nil {
		t.Fatalf("parseGlobalOptions returned error: %v", err)
	}
	if opts.ConfigPath != "site.yaml" {
		t.Fatalf("ConfigPath = %q, want site.yaml", opts.ConfigPath)
	}
	if opts.Format != "json" {
		t.Fatalf("Format = %q, want json", opts.Format)
	}
	if len(rest) != 1 || rest[0] != "validate-config" {
		t.Fatalf("rest = %#v, want [validate-config]", rest)
	}
}

func TestParseGlobalOptionsInvalidTimeout(t *testing.T) {
	_, _, err := parseGlobalOptions([]string{"--timeout", "nope", "help"})
	if err == nil {
		t.Fatal("parseGlobalOptions returned nil error for invalid timeout")
	}
}

func TestParseGlobalOptionsMissingValue(t *testing.T) {
	_, _, err := parseGlobalOptions([]string{"--config"})
	if err == nil {
		t.Fatal("parseGlobalOptions returned nil error for missing config value")
	}
}

func writeCLIConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write CLI config: %v", err)
	}
	return path
}
