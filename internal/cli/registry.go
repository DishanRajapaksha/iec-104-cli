package cli

import "github.com/DishanRajapaksha/industrial-cli-kit/command"

var diagnosticGlobals = []string{"config", "profile", "timeout", "verbose", "debug", "dump-frames"}

var controlFlags = registryFlags("host", "port", "common-address", "ioa", "value", "dry-run", "yes")

var cliRegistry = command.Registry{
	Binary: appName,
	GlobalFlags: []command.Flag{
		{Name: "config", TakesValue: true, Summary: "config file path"},
		{Name: "profile", TakesValue: true, Summary: "config profile name"},
		{Name: "format", TakesValue: true, Summary: "output format"},
		{Name: "timeout", TakesValue: true, Summary: "operation timeout"},
		{Name: "verbose", Summary: "print high-level diagnostics"},
		{Name: "debug", Summary: "print protocol diagnostics"},
		{Name: "dump-frames", Summary: "dump protocol frames"},
	},
	Commands: []command.Command{
		{Name: "validate-config", Summary: "Validate local config", GlobalFlags: []string{"config", "profile", "format"}},
		{Name: "init-config", Summary: "Write a starter YAML config", Flags: registryFlags("output", "force"), GlobalFlags: []string{}},
		{Name: "generate-configs", Summary: "Generate example config files", Flags: registryFlags("dir"), GlobalFlags: []string{}},
		{Name: "status", Summary: "Run TCP and STARTDT diagnostics", Flags: registryFlags("host", "port"), GlobalFlags: diagnosticGlobals},
		{Name: "test-connection", Summary: "Run TCP and STARTDT diagnostics", Flags: registryFlags("host", "port"), GlobalFlags: diagnosticGlobals},
		{Name: "listen", Summary: "Print incoming point values", Flags: registryFlags("host", "port", "duration", "common-address", "ioa", "point")},
		{Name: "monitor", Summary: "Print incoming point values", Flags: registryFlags("host", "port", "duration", "common-address", "ioa", "point")},
		{Name: "interrogate", Summary: "Send general interrogation", Flags: registryFlags("host", "port", "common-address", "ioa", "point")},
		{Name: "watch", Summary: "Print latest cached values", Flags: registryFlags("host", "port", "interval", "stale-after", "ioa", "point", "cache", "no-cache")},
		{Name: "read", Summary: "Read one or more IOAs", Flags: registryFlags("host", "port", "common-address", "ioa", "ioas")},
		{
			Name:        "command",
			Summary:     "Run control commands",
			LeadingArgs: 1,
			GlobalFlags: diagnosticGlobals,
			Subcommands: []command.Command{
				{Name: "single", Summary: "Send a single command", Flags: controlFlags},
				{Name: "double", Summary: "Send a double command", Flags: controlFlags},
			},
		},
		{
			Name:        "setpoint",
			Summary:     "Run setpoint commands",
			LeadingArgs: 1,
			GlobalFlags: diagnosticGlobals,
			Subcommands: []command.Command{
				{Name: "normalized", Summary: "Send a normalized setpoint", Flags: controlFlags},
				{Name: "scaled", Summary: "Send a scaled setpoint", Flags: controlFlags},
				{Name: "float", Summary: "Send a floating-point setpoint", Flags: controlFlags},
			},
		},
		{
			Name:        "clock-sync",
			Summary:     "Synchronize the remote clock",
			Flags:       registryFlags("host", "port", "common-address", "time", "dry-run", "yes"),
			GlobalFlags: diagnosticGlobals,
		},
		{Name: "completions", Summary: "Generate shell completion scripts", LeadingArgs: 1, GlobalFlags: []string{}},
		{Name: "help", Summary: "Print help", GlobalFlags: []string{}},
		{Name: "version", Summary: "Print version information", GlobalFlags: []string{}},
	},
}

func registryFlags(names ...string) []command.Flag {
	boolean := map[string]bool{
		"dry-run":  true,
		"force":    true,
		"no-cache": true,
		"yes":      true,
	}
	flags := make([]command.Flag, 0, len(names))
	for _, name := range names {
		flags = append(flags, command.Flag{Name: name, TakesValue: !boolean[name]})
	}
	return flags
}
