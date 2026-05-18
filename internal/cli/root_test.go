package cli

import (
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

func TestRunValidateConfigMissingFile(t *testing.T) {
	if got := Run([]string{"validate-config", "--config", "missing-test-config.yaml"}); got != exitcode.ConfigError {
		t.Fatalf("Run(validate-config missing) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunTestConnectionRequiresHost(t *testing.T) {
	if got := Run([]string{"test-connection", "--config", "missing-test-config.yaml"}); got != exitcode.ConfigError {
		t.Fatalf("Run(test-connection missing host) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunListenRejectsUnknownFormat(t *testing.T) {
	if got := Run([]string{"listen", "--format", "yaml"}); got != exitcode.ConfigError {
		t.Fatalf("Run(listen invalid format) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunInterrogateRejectsUnknownPoint(t *testing.T) {
	if got := Run([]string{"interrogate", "--config", "config.example.yaml", "--point", "missing"}); got != exitcode.ConfigError {
		t.Fatalf("Run(interrogate unknown point) = %d, want %d", got, exitcode.ConfigError)
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
	if opts.Timeout != 15*time.Second {
		t.Fatalf("Timeout = %s, want 15s", opts.Timeout)
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
