package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
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
	case "listen":
		return runListen(opts, rest[1:])
	case "interrogate":
		return runInterrogate(opts, rest[1:])
	case "watch":
		return runWatch(opts, rest[1:])
	case "read":
		return runRead(opts, rest[1:])
	case "command":
		return runCommand(opts, rest[1:])
	case "setpoint":
		return runSetpoint(opts, rest[1:])
	case "clock-sync":
		return runClockSync(opts, rest[1:])
	case "completions":
		return runCompletions(rest[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", rest[0])
		printHelp(os.Stderr)
		return exitcode.GeneralError
	}
}

func runCompletions(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: iec-104-cli completions bash|zsh")
		return exitcode.ConfigError
	}
	if err := writeCompletion(os.Stdout, args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	return exitcode.Success
}

func runClockSync(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("clock-sync", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	commonAddress := uint(0)
	timeOverride := ""
	safety := controlSafety{}
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "clock-sync timeout")
	fs.UintVar(&commonAddress, "common-address", 0, "common address")
	fs.StringVar(&timeOverride, "time", "", "RFC3339 timestamp to send")
	fs.BoolVar(&safety.DryRun, "dry-run", false, "print clock-sync without sending")
	fs.BoolVar(&safety.Yes, "yes", false, "send the clock-sync command")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	_ = profile
	if err := safety.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	syncTime := time.Now()
	if timeOverride != "" {
		parsed, err := time.Parse(time.RFC3339, timeOverride)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --time %q: %v\n", timeOverride, err)
			return exitcode.ConfigError
		}
		syncTime = parsed
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	applyConnectionFlagOverrides(cfg, visited, host, port, timeout)
	if visited["common-address"] {
		cfg.IEC104.CommonAddress = uint16(commonAddress)
	}

	fmt.Fprintln(os.Stdout, "Clock sync")
	fmt.Fprintf(os.Stdout, "Common address: %d\n", cfg.IEC104.CommonAddress)
	fmt.Fprintf(os.Stdout, "Time: %s\n", syncTime.Format(time.RFC3339))
	if !safety.AllowsExecution() {
		fmt.Fprintln(os.Stdout, "Mode: dry-run")
		fmt.Fprintln(os.Stdout, "Result: not sent")
		return exitcode.Success
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout.Duration())
	defer cancel()
	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	logVerbose(opts, "sending clock sync to common address %d", cfg.IEC104.CommonAddress)
	logDebug(opts, "clock-sync time=%s", syncTime.Format(time.RFC3339))
	if err := client.SyncClock(ctx, cfg.IEC104.CommonAddress, syncTime); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}
	fmt.Fprintln(os.Stdout, "Mode: execute")
	fmt.Fprintln(os.Stdout, "Result: sent")
	return exitcode.Success
}

func runSetpoint(opts globalOptions, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "setpoint type is required")
		return exitcode.ConfigError
	}
	switch args[0] {
	case "normalized", "scaled", "float":
		return runSetpointKind(opts, args[0], args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown setpoint type %q\n", args[0])
		return exitcode.ConfigError
	}
}

func runSetpointKind(opts globalOptions, kind string, args []string) int {
	fs := flag.NewFlagSet("setpoint "+kind, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	commonAddress := uint(0)
	ioa := uint(0)
	rawValue := ""
	safety := controlSafety{}
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "setpoint timeout")
	fs.UintVar(&commonAddress, "common-address", 0, "common address")
	fs.UintVar(&ioa, "ioa", 0, "information object address")
	fs.StringVar(&rawValue, "value", "", "setpoint value")
	fs.BoolVar(&safety.DryRun, "dry-run", false, "print setpoint without sending")
	fs.BoolVar(&safety.Yes, "yes", false, "send the setpoint")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	_ = profile
	if err := safety.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	if ioa == 0 {
		fmt.Fprintln(os.Stderr, "--ioa is required")
		return exitcode.ConfigError
	}
	value, display, err := parseSetpointValue(kind, rawValue)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	applyConnectionFlagOverrides(cfg, visited, host, port, timeout)
	if visited["common-address"] {
		cfg.IEC104.CommonAddress = uint16(commonAddress)
	}

	fmt.Fprintf(os.Stdout, "Setpoint command: %s\n", kind)
	fmt.Fprintf(os.Stdout, "Common address: %d\n", cfg.IEC104.CommonAddress)
	fmt.Fprintf(os.Stdout, "IOA: %d\n", ioa)
	fmt.Fprintf(os.Stdout, "Type: %s\n", kind)
	fmt.Fprintf(os.Stdout, "Value: %s\n", display)
	fmt.Fprintf(os.Stdout, "Qualifier: no_additional_definition\n")
	if !safety.AllowsExecution() {
		fmt.Fprintln(os.Stdout, "Mode: dry-run")
		fmt.Fprintln(os.Stdout, "Result: not sent")
		return exitcode.Success
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout.Duration())
	defer cancel()
	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	logVerbose(opts, "sending %s setpoint to IOA %d", kind, ioa)
	logDebug(opts, "setpoint common_address=%d value=%v", cfg.IEC104.CommonAddress, value)
	if err := client.SendSetpoint(ctx, cfg.IEC104.CommonAddress, uint32(ioa), kind, value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}
	fmt.Fprintln(os.Stdout, "Mode: execute")
	fmt.Fprintln(os.Stdout, "Result: sent")
	return exitcode.Success
}

func parseSetpointValue(kind string, raw string) (any, string, error) {
	if raw == "" {
		return nil, "", fmt.Errorf("--value is required")
	}
	switch kind {
	case "normalized":
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, "", fmt.Errorf("invalid normalized setpoint %q: %w", raw, err)
		}
		if value < -1 || value > 1 {
			return nil, "", fmt.Errorf("normalized setpoint must be between -1 and 1")
		}
		return int16(value * 32767), fmt.Sprintf("%g", value), nil
	case "scaled":
		value, err := strconv.ParseInt(raw, 10, 16)
		if err != nil {
			return nil, "", fmt.Errorf("scaled setpoint must be a 16-bit integer: %w", err)
		}
		return int16(value), fmt.Sprintf("%d", value), nil
	case "float":
		value, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return nil, "", fmt.Errorf("invalid float setpoint %q: %w", raw, err)
		}
		return float32(value), fmt.Sprintf("%g", value), nil
	default:
		return nil, "", fmt.Errorf("unknown setpoint type %q", kind)
	}
}

func runCommand(opts globalOptions, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "command type is required")
		return exitcode.ConfigError
	}
	switch args[0] {
	case "single":
		return runCommandSingle(opts, args[1:])
	case "double":
		return runCommandDouble(opts, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command type %q\n", args[0])
		return exitcode.ConfigError
	}
}

func runCommandSingle(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("command single", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	commonAddress := uint(0)
	ioa := uint(0)
	rawValue := ""
	safety := controlSafety{}
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "command timeout")
	fs.UintVar(&commonAddress, "common-address", 0, "common address")
	fs.UintVar(&ioa, "ioa", 0, "information object address")
	fs.StringVar(&rawValue, "value", "", "single command value")
	fs.BoolVar(&safety.DryRun, "dry-run", false, "print command without sending")
	fs.BoolVar(&safety.Yes, "yes", false, "send the command")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	_ = profile
	if err := safety.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	if ioa == 0 {
		fmt.Fprintln(os.Stderr, "--ioa is required")
		return exitcode.ConfigError
	}
	if rawValue == "" {
		fmt.Fprintln(os.Stderr, "--value is required")
		return exitcode.ConfigError
	}
	value, err := parseSingleCommandValue(rawValue)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	applyConnectionFlagOverrides(cfg, visited, host, port, timeout)
	if visited["common-address"] {
		cfg.IEC104.CommonAddress = uint16(commonAddress)
	}

	fmt.Fprintln(os.Stdout, "Control command: single")
	fmt.Fprintf(os.Stdout, "Common address: %d\n", cfg.IEC104.CommonAddress)
	fmt.Fprintf(os.Stdout, "IOA: %d\n", ioa)
	fmt.Fprintf(os.Stdout, "Type: single\n")
	fmt.Fprintf(os.Stdout, "Value: %t\n", value)
	fmt.Fprintf(os.Stdout, "Qualifier: no_additional_definition\n")
	if !safety.AllowsExecution() {
		fmt.Fprintln(os.Stdout, "Mode: dry-run")
		fmt.Fprintln(os.Stdout, "Result: not sent")
		return exitcode.Success
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout.Duration())
	defer cancel()
	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	if err := client.SendSingleCommand(ctx, cfg.IEC104.CommonAddress, uint32(ioa), value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}
	fmt.Fprintln(os.Stdout, "Mode: execute")
	fmt.Fprintln(os.Stdout, "Result: sent")
	return exitcode.Success
}

func parseSingleCommandValue(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1":
		return true, nil
	case "off", "false", "0":
		return false, nil
	default:
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed, nil
		}
		return false, fmt.Errorf("invalid single command value %q; expected on, off, true, false, 1, or 0", value)
	}
}

func runCommandDouble(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("command double", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	commonAddress := uint(0)
	ioa := uint(0)
	rawValue := ""
	safety := controlSafety{}
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "command timeout")
	fs.UintVar(&commonAddress, "common-address", 0, "common address")
	fs.UintVar(&ioa, "ioa", 0, "information object address")
	fs.StringVar(&rawValue, "value", "", "double command value")
	fs.BoolVar(&safety.DryRun, "dry-run", false, "print command without sending")
	fs.BoolVar(&safety.Yes, "yes", false, "send the command")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	_ = profile
	if err := safety.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	if ioa == 0 {
		fmt.Fprintln(os.Stderr, "--ioa is required")
		return exitcode.ConfigError
	}
	value, label, err := parseDoubleCommandValue(rawValue)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	applyConnectionFlagOverrides(cfg, visited, host, port, timeout)
	if visited["common-address"] {
		cfg.IEC104.CommonAddress = uint16(commonAddress)
	}

	fmt.Fprintln(os.Stdout, "Control command: double")
	fmt.Fprintf(os.Stdout, "Common address: %d\n", cfg.IEC104.CommonAddress)
	fmt.Fprintf(os.Stdout, "IOA: %d\n", ioa)
	fmt.Fprintf(os.Stdout, "Type: double\n")
	fmt.Fprintf(os.Stdout, "Value: %s\n", label)
	fmt.Fprintf(os.Stdout, "Qualifier: no_additional_definition\n")
	if !safety.AllowsExecution() {
		fmt.Fprintln(os.Stdout, "Mode: dry-run")
		fmt.Fprintln(os.Stdout, "Result: not sent")
		return exitcode.Success
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout.Duration())
	defer cancel()
	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	if err := client.SendDoubleCommand(ctx, cfg.IEC104.CommonAddress, uint32(ioa), value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}
	fmt.Fprintln(os.Stdout, "Mode: execute")
	fmt.Fprintln(os.Stdout, "Result: sent")
	return exitcode.Success
}

func parseDoubleCommandValue(value string) (uint8, string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "intermediate":
		return 0, "intermediate", nil
	case "off", "open":
		return 1, "off", nil
	case "on", "close":
		return 2, "on", nil
	case "indeterminate":
		return 3, "indeterminate", nil
	default:
		return 0, "", fmt.Errorf("invalid double command value %q; expected on, off, open, close, intermediate, or indeterminate", value)
	}
}

func runRead(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	commonAddress := uint(0)
	ioa := uint(0)
	format := opts.Format
	verbose := opts.Verbose
	debug := opts.Debug
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "read timeout")
	fs.UintVar(&commonAddress, "common-address", 0, "common address")
	fs.UintVar(&ioa, "ioa", 0, "information object address to read")
	fs.StringVar(&format, "format", format, "output format: table, text, json, jsonl")
	fs.BoolVar(&verbose, "verbose", verbose, "print high-level connection decisions")
	fs.BoolVar(&debug, "debug", debug, "print protocol-level summaries")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	opts.Verbose = verbose
	opts.Debug = debug
	_ = profile
	if ioa == 0 {
		fmt.Fprintln(os.Stderr, "--ioa is required")
		return exitcode.ConfigError
	}
	if _, ok := allowedFormats[format]; !ok {
		fmt.Fprintf(os.Stderr, "invalid output format %q; expected one of table, text, json, jsonl\n", format)
		return exitcode.ConfigError
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	applyConnectionFlagOverrides(cfg, visited, host, port, timeout)
	if visited["common-address"] {
		cfg.IEC104.CommonAddress = uint16(commonAddress)
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	filter, err := buildPointFilter(*cfg, 0, uint32(ioa), "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout.Duration())
	defer cancel()
	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	value, err := client.Read(ctx, cfg.IEC104.CommonAddress, uint32(ioa))
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		fmt.Fprintf(os.Stderr, "read timed out waiting for IOA %d; some IEC 104 devices do not support read and expect interrogation or spontaneous updates instead\n", ioa)
		return exitcode.InterrogationTimeout
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}
	enriched, ok := filter(value)
	if !ok {
		fmt.Fprintf(os.Stderr, "read response did not match IOA %d\n", ioa)
		return exitcode.GeneralError
	}
	if err := writePointValues(os.Stdout, format, []iec104.PointValue{enriched}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.OutputError
	}
	return exitcode.Success
}

func runWatch(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	interval := time.Second
	staleAfter := 30 * time.Second
	ioa := uint(0)
	pointName := ""
	format := opts.Format
	verbose := opts.Verbose
	debug := opts.Debug
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "connection timeout")
	fs.DurationVar(&interval, "interval", interval, "print interval")
	fs.DurationVar(&staleAfter, "stale-after", staleAfter, "mark values stale after this age")
	fs.UintVar(&ioa, "ioa", 0, "filter by information object address")
	fs.StringVar(&pointName, "point", "", "filter by configured point name")
	fs.StringVar(&format, "format", format, "output format: table, text, json, jsonl")
	fs.BoolVar(&verbose, "verbose", verbose, "print high-level connection decisions")
	fs.BoolVar(&debug, "debug", debug, "print protocol-level summaries")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	opts.Verbose = verbose
	opts.Debug = debug
	_ = profile
	if interval <= 0 {
		fmt.Fprintln(os.Stderr, "--interval must be positive")
		return exitcode.ConfigError
	}
	if staleAfter <= 0 {
		fmt.Fprintln(os.Stderr, "--stale-after must be positive")
		return exitcode.ConfigError
	}
	if _, ok := allowedFormats[format]; !ok {
		fmt.Fprintf(os.Stderr, "invalid output format %q; expected one of table, text, json, jsonl\n", format)
		return exitcode.ConfigError
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	applyConnectionFlagOverrides(cfg, visitedFlags(fs), host, port, timeout)
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	filter, err := buildPointFilter(*cfg, 0, uint32(ioa), pointName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := iec104.NewLatestCache()
	errCh := make(chan error, 1)
	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	logVerbose(opts, "starting watch on %s:%d", cfg.Connection.Host, cfg.Connection.Port)
	logDebug(opts, "watch interval=%s stale_after=%s ioa=%d point=%q format=%s", interval, staleAfter, ioa, pointName, format)
	go func() {
		errCh <- client.Listen(ctx, func(value iec104.PointValue) {
			cache.Update(value)
		})
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				fmt.Fprintln(os.Stderr, err)
				return mapRunError(err)
			}
			return exitcode.Success
		case now := <-ticker.C:
			values := cache.Snapshot(now, staleAfter)
			filtered := make([]iec104.PointValue, 0, len(values))
			for _, value := range values {
				if enriched, ok := filter(value); ok {
					filtered = append(filtered, enriched)
				}
			}
			if err := writePointValues(os.Stdout, format, filtered); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return exitcode.OutputError
			}
		}
	}
}

func runInterrogate(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("interrogate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	commonAddress := uint(0)
	ioa := uint(0)
	pointName := ""
	format := opts.Format
	verbose := opts.Verbose
	debug := opts.Debug
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "interrogation timeout")
	fs.UintVar(&commonAddress, "common-address", 0, "common address to interrogate")
	fs.UintVar(&ioa, "ioa", 0, "filter by information object address")
	fs.StringVar(&pointName, "point", "", "filter by configured point name")
	fs.StringVar(&format, "format", format, "output format: table, text, json, jsonl")
	fs.BoolVar(&verbose, "verbose", verbose, "print high-level connection decisions")
	fs.BoolVar(&debug, "debug", debug, "print protocol-level summaries")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	opts.Verbose = verbose
	opts.Debug = debug
	_ = profile
	if _, ok := allowedFormats[format]; !ok {
		fmt.Fprintf(os.Stderr, "invalid output format %q; expected one of table, text, json, jsonl\n", format)
		return exitcode.ConfigError
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	applyConnectionFlagOverrides(cfg, visited, host, port, timeout)
	if visited["common-address"] {
		cfg.IEC104.CommonAddress = uint16(commonAddress)
	}
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	filter, err := buildPointFilter(*cfg, 0, uint32(ioa), pointName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.Timeout.Duration())
	defer cancel()
	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	logVerbose(opts, "interrogating common address %d on %s:%d", cfg.IEC104.CommonAddress, cfg.Connection.Host, cfg.Connection.Port)
	logDebug(opts, "interrogate ioa=%d point=%q format=%s", ioa, pointName, format)
	values, err := client.Interrogate(ctx, cfg.IEC104.CommonAddress)
	filtered := make([]iec104.PointValue, 0, len(values))
	for _, value := range values {
		if enriched, ok := filter(value); ok {
			filtered = append(filtered, enriched)
		}
	}
	if len(filtered) == 0 && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
		fmt.Fprintln(os.Stderr, "interrogation timed out before receiving matching values")
		return exitcode.InterrogationTimeout
	}
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}
	if err := writePointValues(os.Stdout, format, filtered); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.OutputError
	}
	return exitcode.Success
}

func runListen(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("listen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	duration := time.Duration(0)
	commonAddress := uint(0)
	ioa := uint(0)
	pointName := ""
	format := opts.Format
	verbose := opts.Verbose
	debug := opts.Debug
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "connection timeout")
	fs.DurationVar(&duration, "duration", 0, "listen duration; 0 means until interrupted")
	fs.UintVar(&commonAddress, "common-address", 0, "filter by common address")
	fs.UintVar(&ioa, "ioa", 0, "filter by information object address")
	fs.StringVar(&pointName, "point", "", "filter by configured point name")
	fs.StringVar(&format, "format", format, "output format: table, text, json, jsonl")
	fs.BoolVar(&verbose, "verbose", verbose, "print high-level connection decisions")
	fs.BoolVar(&debug, "debug", debug, "print protocol-level summaries")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	opts.Verbose = verbose
	opts.Debug = debug
	_ = profile
	if _, ok := allowedFormats[format]; !ok {
		fmt.Fprintf(os.Stderr, "invalid output format %q; expected one of table, text, json, jsonl\n", format)
		return exitcode.ConfigError
	}

	cfg, _, err := config.Load(configPath, config.Overrides{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	visited := visitedFlags(fs)
	applyConnectionFlagOverrides(cfg, visited, host, port, timeout)
	if err := config.Validate(*cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	filter, err := buildPointFilter(*cfg, uint16(commonAddress), uint32(ioa), pointName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	ctx := context.Background()
	cancel := func() {}
	if duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, duration)
	}
	defer cancel()

	client := iec104.NewWendyClient(clientConfigFromConfig(*cfg, opts.Debug))
	logVerbose(opts, "listening on %s:%d", cfg.Connection.Host, cfg.Connection.Port)
	logDebug(opts, "listen duration=%s common_address=%d ioa=%d point=%q format=%s", duration, commonAddress, ioa, pointName, format)
	err = client.Listen(ctx, func(value iec104.PointValue) {
		if enriched, ok := filter(value); ok {
			if writeErr := writePointValues(os.Stdout, format, []iec104.PointValue{enriched}); writeErr != nil {
				fmt.Fprintln(os.Stderr, writeErr)
			}
		}
	})
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return exitcode.Success
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}
	return exitcode.Success
}

func runTestConnection(opts globalOptions, args []string) int {
	fs := flag.NewFlagSet("test-connection", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := opts.ConfigPath
	profile := opts.Profile
	host := ""
	port := 0
	timeout := opts.Timeout
	verbose := opts.Verbose
	debug := opts.Debug
	fs.StringVar(&configPath, "config", configPath, "YAML config file")
	fs.StringVar(&profile, "profile", profile, "config profile name")
	fs.StringVar(&host, "host", "", "IEC 104 server host")
	fs.IntVar(&port, "port", 0, "IEC 104 server TCP port")
	fs.DurationVar(&timeout, "timeout", timeout, "connection timeout")
	fs.BoolVar(&verbose, "verbose", verbose, "print high-level connection decisions")
	fs.BoolVar(&debug, "debug", debug, "print protocol-level summaries")
	if err := fs.Parse(args); err != nil {
		return exitcode.ConfigError
	}
	opts.Verbose = verbose
	opts.Debug = debug
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
	logVerbose(opts, "testing connection to %s:%d", cfg.Connection.Host, cfg.Connection.Port)

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
		logDebug(opts, "test-connection failed: %v", err)
		fmt.Fprintf(os.Stdout, "TCP: failed\n")
		fmt.Fprintf(os.Stdout, "IEC104 STARTDT: not started\n")
		fmt.Fprintf(os.Stdout, "Result: failed\n")
		fmt.Fprintln(os.Stderr, err)
		return mapRunError(err)
	}

	fmt.Fprintf(os.Stdout, "TCP: ok\n")
	fmt.Fprintf(os.Stdout, "IEC104 STARTDT: ok\n")
	fmt.Fprintf(os.Stdout, "Result: connected\n")
	logVerbose(opts, "connection test succeeded")
	return exitcode.Success
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func applyConnectionFlagOverrides(cfg *config.Config, visited map[string]bool, host string, port int, timeout time.Duration) {
	if visited["host"] {
		cfg.Connection.Host = host
	}
	if visited["port"] {
		cfg.Connection.Port = port
	}
	if visited["timeout"] && timeout > 0 {
		cfg.Connection.Timeout = config.NewDuration(timeout)
	}
}

func clientConfigFromConfig(cfg config.Config, debug bool) iec104.ClientConfig {
	return iec104.ClientConfig{
		Host:              cfg.Connection.Host,
		Port:              cfg.Connection.Port,
		Timeout:           cfg.Connection.Timeout.Duration(),
		Reconnect:         cfg.Connection.Reconnect,
		ReconnectInterval: cfg.Connection.ReconnectInterval.Duration(),
		OriginatorAddress: cfg.IEC104.OriginatorAddress,
		Debug:             debug,
	}
}

func mapRunError(err error) int {
	switch {
	case errors.Is(err, iec104.ErrTCPConnection), errors.Is(err, context.DeadlineExceeded):
		return exitcode.TCPConnectionError
	case errors.Is(err, iec104.ErrSession):
		return exitcode.IEC104SessionError
	case errors.Is(err, iec104.ErrInterrogationTimeout):
		return exitcode.InterrogationTimeout
	case errors.Is(err, iec104.ErrUnsupportedType):
		return exitcode.UnsupportedASDU
	case errors.Is(err, iec104.ErrCommandRejected):
		return exitcode.CommandRejected
	case errors.Is(err, iec104.ErrCommandTimeout):
		return exitcode.CommandTimeout
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
  listen           Print incoming point values
  interrogate      Send general interrogation and print matching values
  watch            Print latest cached values on an interval
  read             Send IEC 104 read for a specific IOA
  command          Run control commands with dry-run safety
  setpoint         Run setpoint commands with dry-run safety
  clock-sync       Run clock synchronization with dry-run safety
  completions

`, appName, appName)
}
