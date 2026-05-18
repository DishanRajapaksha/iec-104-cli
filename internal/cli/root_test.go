package cli

import (
	"fmt"
	"testing"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
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

func TestRunWatchRejectsInvalidInterval(t *testing.T) {
	if got := Run([]string{"watch", "--interval", "0s"}); got != exitcode.ConfigError {
		t.Fatalf("Run(watch invalid interval) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunReadRequiresIOA(t *testing.T) {
	if got := Run([]string{"read", "--config", "config.example.yaml"}); got != exitcode.ConfigError {
		t.Fatalf("Run(read missing ioa) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunCommandSingleDefaultsToDryRun(t *testing.T) {
	if got := Run([]string{"command", "single", "--ioa", "1000", "--value", "on"}); got != exitcode.Success {
		t.Fatalf("Run(command single dry run) = %d, want %d", got, exitcode.Success)
	}
}

func TestCommandDryRunDoesNotRequireServer(t *testing.T) {
	if got := Run([]string{"command", "single", "--config", "missing.yaml", "--ioa", "1000", "--value", "on", "--dry-run"}); got != exitcode.Success {
		t.Fatalf("Run(command dry run without server) = %d, want %d", got, exitcode.Success)
	}
}

func TestParseSingleCommandValue(t *testing.T) {
	for _, value := range []string{"on", "true", "1"} {
		got, err := parseSingleCommandValue(value)
		if err != nil || !got {
			t.Fatalf("parseSingleCommandValue(%q) = %t, %v; want true, nil", value, got, err)
		}
	}
	for _, value := range []string{"off", "false", "0"} {
		got, err := parseSingleCommandValue(value)
		if err != nil || got {
			t.Fatalf("parseSingleCommandValue(%q) = %t, %v; want false, nil", value, got, err)
		}
	}
	if _, err := parseSingleCommandValue("open"); err == nil {
		t.Fatal("expected invalid value error")
	}
}

func TestRunCommandDoubleDryRun(t *testing.T) {
	if got := Run([]string{"command", "double", "--ioa", "1001", "--value", "close", "--dry-run"}); got != exitcode.Success {
		t.Fatalf("Run(command double dry run) = %d, want %d", got, exitcode.Success)
	}
}

func TestParseDoubleCommandValue(t *testing.T) {
	tests := map[string]uint8{
		"intermediate":  0,
		"open":          1,
		"off":           1,
		"close":         2,
		"on":            2,
		"indeterminate": 3,
	}
	for value, want := range tests {
		got, _, err := parseDoubleCommandValue(value)
		if err != nil || got != want {
			t.Fatalf("parseDoubleCommandValue(%q) = %d, %v; want %d, nil", value, got, err, want)
		}
	}
	if _, _, err := parseDoubleCommandValue("bad"); err == nil {
		t.Fatal("expected invalid value error")
	}
}

func TestRunSetpointFloatDryRun(t *testing.T) {
	if got := Run([]string{"setpoint", "float", "--ioa", "2002", "--value", "12.5", "--dry-run"}); got != exitcode.Success {
		t.Fatalf("Run(setpoint float dry run) = %d, want %d", got, exitcode.Success)
	}
}

func TestParseSetpointValue(t *testing.T) {
	normalized, _, err := parseSetpointValue("normalized", "0.5")
	if err != nil {
		t.Fatalf("parse normalized returned error: %v", err)
	}
	if normalized.(int16) == 0 {
		t.Fatal("normalized setpoint was not converted")
	}
	scaled, _, err := parseSetpointValue("scaled", "42")
	if err != nil || scaled.(int16) != 42 {
		t.Fatalf("parse scaled = %#v, %v; want 42, nil", scaled, err)
	}
	floatValue, _, err := parseSetpointValue("float", "12.5")
	if err != nil || floatValue.(float32) != 12.5 {
		t.Fatalf("parse float = %#v, %v; want 12.5, nil", floatValue, err)
	}
	if _, _, err := parseSetpointValue("normalized", "2"); err == nil {
		t.Fatal("expected normalized range error")
	}
}

func TestRunClockSyncDryRunWithTime(t *testing.T) {
	if got := Run([]string{"clock-sync", "--time", "2026-05-18T12:00:00Z", "--dry-run"}); got != exitcode.Success {
		t.Fatalf("Run(clock-sync dry run) = %d, want %d", got, exitcode.Success)
	}
}

func TestRunVerboseDoesNotChangeExitCode(t *testing.T) {
	if got := Run([]string{"test-connection", "--verbose", "--config", "missing-test-config.yaml"}); got != exitcode.ConfigError {
		t.Fatalf("Run(verbose missing config) = %d, want %d", got, exitcode.ConfigError)
	}
}

func TestRunCompletionsBash(t *testing.T) {
	if got := Run([]string{"completions", "bash"}); got != exitcode.Success {
		t.Fatalf("Run(completions bash) = %d, want %d", got, exitcode.Success)
	}
}

func TestMapRunError(t *testing.T) {
	tests := map[error]int{
		fmt.Errorf("wrap: %w", iec104.ErrTCPConnection):        exitcode.TCPConnectionError,
		fmt.Errorf("wrap: %w", iec104.ErrSession):              exitcode.IEC104SessionError,
		fmt.Errorf("wrap: %w", iec104.ErrUnsupportedType):      exitcode.UnsupportedASDU,
		fmt.Errorf("wrap: %w", iec104.ErrCommandRejected):      exitcode.CommandRejected,
		fmt.Errorf("wrap: %w", iec104.ErrCommandTimeout):       exitcode.CommandTimeout,
		fmt.Errorf("wrap: %w", iec104.ErrInterrogationTimeout): exitcode.InterrogationTimeout,
	}
	for err, want := range tests {
		if got := mapRunError(err); got != want {
			t.Fatalf("mapRunError(%v) = %d, want %d", err, got, want)
		}
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
		"--dump-frames",
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
	if !opts.DumpFrames {
		t.Fatalf("DumpFrames = false, want true")
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
