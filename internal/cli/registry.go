package cli

import "github.com/DishanRajapaksha/industrial-cli-kit/command"

var controlFlags = registryFlags("config", "profile", "host", "port", "timeout", "common-address", "ioa", "value", "qualifier", "select", "dry-run", "yes")

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
		{Name: "validate-config", Summary: "Validate local config"},
		{Name: "init-config", Summary: "Write a starter YAML config", Flags: registryFlags("output", "force")},
		{Name: "generate-configs", Summary: "Generate example config files", Flags: registryFlags("output-dir", "force")},
		{Name: "status", Summary: "Run TCP and STARTDT diagnostics", Flags: connectionFlags()},
		{Name: "test-connection", Summary: "Run TCP and STARTDT diagnostics", Flags: connectionFlags()},
		{Name: "listen", Summary: "Print incoming point values", Flags: append(connectionFlags(), registryFlags("duration")...)},
		{Name: "monitor", Summary: "Print incoming point values", Flags: append(connectionFlags(), registryFlags("duration")...)},
		{Name: "interrogate", Summary: "Send general interrogation", Flags: append(connectionFlags(), registryFlags("common-address", "qualifier", "duration")...)},
		{Name: "watch", Summary: "Print latest cached values", Flags: registryFlags("config", "profile", "interval", "count", "duration", "name", "ioa")},
		{Name: "read", Summary: "Read a specific IOA", Flags: append(connectionFlags(), registryFlags("common-address", "ioa")...)},
		{Name: "command", Summary: "Run control commands", Subcommands: []command.Command{
			{Name: "single", Summary: "Send a single command", Flags: controlFlags},
			{Name: "double", Summary: "Send a double command", Flags: controlFlags},
		}},
		{Name: "setpoint", Summary: "Run setpoint commands", Subcommands: []command.Command{
			{Name: "normalized", Summary: "Send a normalized setpoint", Flags: controlFlags},
			{Name: "scaled", Summary: "Send a scaled setpoint", Flags: controlFlags},
			{Name: "float", Summary: "Send a floating-point setpoint", Flags: controlFlags},
		}},
		{Name: "clock-sync", Summary: "Synchronize the remote clock", Flags: registryFlags("config", "profile", "host", "port", "timeout", "common-address", "time", "dry-run", "yes")},
		{Name: "completions", Summary: "Generate shell completion scripts"},
		{Name: "help", Summary: "Print help"},
		{Name: "version", Summary: "Print version information"},
	},
}

func connectionFlags() []command.Flag {
	return registryFlags("config", "profile", "host", "port", "timeout")
}

func registryFlags(names ...string) []command.Flag {
	flags := make([]command.Flag, 0, len(names))
	for _, name := range names {
		takesValue := name != "force" && name != "select" && name != "dry-run" && name != "yes"
		flags = append(flags, command.Flag{Name: name, TakesValue: takesValue})
	}
	return flags
}
