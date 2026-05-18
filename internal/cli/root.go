package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/config"
	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
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
	Timeout    time.Duration
	Verbose    bool
	Debug      bool
}

func defaultGlobalOptions() globalOptions {
	return globalOptions{
		ConfigPath: defaultConfigPath,
		Format:     defaultFormat,
	}
}

func Main() {
	os.Exit(Run(os.Args[1:]))
}

func Run(args []string) int {
	opts, rest, err := parseGlobalOptions(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	_ = opts

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
		return runValidateConfig(opts, rest[1:])
	case "test-connection":
		return runTestConnection(opts, rest[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", rest[0])
		printHelp(os.Stderr)
		return exitcode.GeneralError
	}
}

func runTestConnection(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("test-connection", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "connection timeout")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	_ = profile

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	if visited["host"] {
		cfg.Connection.Host = host
	}
	if visited["port"] {
		cfg.Connection.Port = port
	}
	if visited["timeout"] || opts.Timeout > 0 {
		cfg.Connection.Timeout = config.NewDuration(timeout)
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	fmt.Fprintf(os.Stdout, "Host: %s\n", cfg.Connection.Host)
	fmt.Fprintf(os.Stdout, "Port: %d\n", cfg.Connection.Port)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout.Duration())
	defer cancel()
	client := iec104.NewWendyClient(iec104.ClientConfig{
		Host:              cfg.Connection.Host,
		Port:              cfg.Connection.Port,
		Timeout:           cfg.Connection.Timeout.Duration(),
		Reconnect:         false,
		ReconnectInterval: cfg.Connection.ReconnectInterval.Duration(),
		OriginatorAddress: cfg.IEC104.OriginatorAddress,
		Debug:             opts.Debug,
	})
	if err := client.TestConnection(ctx); err != nil {
		fmt.Fprintf(os.Stdout, "TCP: failed\n")
		fmt.Fprintf(os.Stdout, "IEC104 STARTDT: not started\n")
		fmt.Fprintf(os.Stdout, "Result: failed\n")
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}

	fmt.Fprintf(os.Stdout, "TCP: ok\n")
	fmt.Fprintf(os.Stdout, "IEC104 STARTDT: ok\n")
	fmt.Fprintf(os.Stdout, "Result: connected\n")
	return exitcode.Success
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func mapRunError(err error) int {
	switch {
	case errors.Is(err, iec104.ErrTCPConnection), errors.Is(err, context.DeadlineExceeded):
		return exitcode.TCPConnectionError
	case errors.Is(err, iec104.ErrSession):
		return exitcode.IEC104SessionError
	default:
		return exitcode.GeneralError
	}
}

func runValidateConfig(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("validate-config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	_ = profile

	cfg, err := config.LoadRequired(configPath, config.Overrides{OutputFormat: &opts.Format})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	fmt.Fprintln(os.Stdout, "config validation: PASS")
	return exitcode.Success
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
			rest = append(rest, args[i:]...)
			break
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
			i = next
		case "--verbose":
			opts.Verbose = true
		case "--debug":
			opts.Debug = true
		case "--help", "-h", "--version", "-v":
			rest = append(rest, arg)
			rest = append(rest, args[i+1:]...)
			return opts, rest, nil
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
  validate-config  Validate local config without server connection
  test-connection  Run TCP and IEC 104 STARTDT diagnostics

Planned commands:
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
