package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
)

const (
	appName           = "iec-104-cli"
	defaultConfigPath = "config.yaml"
	defaultFormat     = "table"
)

var allowedFormats = map[string]struct{}{
	"table": {},
	"text":  {},
	"json":  {},
	"jsonl": {},
}

type globalOptions struct {
	ConfigPath string
	Profile    string
	Format     string
	FormatSet  bool
	Timeout    time.Duration
	TimeoutSet bool
	Verbose    bool
	Debug      bool
}

func defaultGlobalOptions() globalOptions {
	return globalOptions{
		ConfigPath: defaultConfigPath,
		Format:     defaultFormat,
	}
}

// Main is the process entrypoint for the CLI package.
func Main() {
	os.Exit(Run(os.Args[1:]))
}

// Run executes the CLI with the provided arguments and returns a stable exit code.
func Run(args []string) int {
	opts, rest, err := parseGlobalOptions(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	if len(rest) == 0 {
		printHelp(os.Stdout)
		return exitcode.Success
	}

	switch rest[0] {
	case "help", "--help", "-h":
		printHelp(os.Stdout)
		return exitcode.Success
	case "version", "--version", "-v":
		fmt.Fprintf(os.Stdout, "%s development\n", appName)
		return exitcode.Success
	case "validate-config":
		return runValidateConfig(opts)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", rest[0])
		printHelp(os.Stderr)
		return exitcode.GeneralError
	}
}

func parseGlobalOptions(args []string) (globalOptions, []string, error) {
	opts := defaultGlobalOptions()
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}

		if !strings.HasPrefix(arg, "-") {
			rest = append(rest, arg)
			continue
		}

		name, value, hasInlineValue := strings.Cut(arg, "=")

		switch name {
		case "--config":
			v, next, err := flagValue("--config", value, hasInlineValue, args, i)
			if err != nil {
				return opts, nil, err
			}
			opts.ConfigPath = v
			i = next
		case "--profile":
			v, next, err := flagValue("--profile", value, hasInlineValue, args, i)
			if err != nil {
				return opts, nil, err
			}
			opts.Profile = v
			i = next
		case "--format":
			v, next, err := flagValue("--format", value, hasInlineValue, args, i)
			if err != nil {
				return opts, nil, err
			}
			if _, ok := allowedFormats[v]; !ok {
				return opts, nil, fmt.Errorf("invalid output format %q; expected one of table, text, json, jsonl", v)
			}
			opts.Format = v
			opts.FormatSet = true
			i = next
		case "--timeout":
			v, next, err := flagValue("--timeout", value, hasInlineValue, args, i)
			if err != nil {
				return opts, nil, err
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return opts, nil, fmt.Errorf("invalid timeout %q: %w", v, err)
			}
			if d <= 0 {
				return opts, nil, fmt.Errorf("timeout must be positive")
			}
			opts.Timeout = d
			opts.TimeoutSet = true
			i = next
		case "--verbose":
			opts.Verbose = true
		case "--debug":
			opts.Debug = true
		case "--help", "-h", "--version", "-v":
			rest = append(rest, arg)
		default:
			return opts, nil, fmt.Errorf("unknown global flag %q", name)
		}
	}

	return opts, rest, nil
}

func flagValue(name, inlineValue string, hasInlineValue bool, args []string, index int) (string, int, error) {
	if hasInlineValue {
		if inlineValue == "" {
			return "", index, fmt.Errorf("%s requires a value", name)
		}
		return inlineValue, index, nil
	}

	next := index + 1
	if next >= len(args) || strings.HasPrefix(args[next], "-") {
		return "", index, fmt.Errorf("%s requires a value", name)
	}

	return args[next], next, nil
}

func printHelp(out *os.File) {
	fmt.Fprintf(out, `%s is a script-friendly IEC 60870-5-104 command-line client.

Usage:
  %s [global flags] <command> [flags]

Global flags:
  --config string     Config file path (default "config.yaml")
  --profile string    Config profile name
  --format string     Output format: table, text, json, jsonl (default "table")
  --timeout duration  Operation timeout, for example 10s or 1m
  --verbose           Print high-level diagnostics to stderr
  --debug             Print protocol-level diagnostics to stderr

Available commands:
  help             Show this help message
  version          Show version information
  validate-config  Validate local configuration without connecting to a server

Planned commands:
  test-connection
  listen
  interrogate
  watch
  read
  command
  setpoint
  clock-sync
  completions

`, appName, appName)
}
