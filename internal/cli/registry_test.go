package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DishanRajapaksha/industrial-cli-kit/command"
	"github.com/DishanRajapaksha/industrial-cli-kit/completion"
)

func TestRegistryMatchesDispatcher(t *testing.T) {
	dispatched := []string{
		"validate-config", "generate-configs", "init-config", "test-connection", "status",
		"listen", "monitor", "interrogate", "watch", "read", "command", "setpoint",
		"clock-sync", "completions", "help", "version",
	}
	registered := map[string]bool{}
	for _, registeredCommand := range cliRegistry.Commands {
		if registered[registeredCommand.Name] {
			t.Fatalf("duplicate registry command %q", registeredCommand.Name)
		}
		registered[registeredCommand.Name] = true
	}
	for _, name := range dispatched {
		if !registered[name] {
			t.Errorf("dispatcher command %q is not registered", name)
		}
	}
	for name := range registered {
		found := false
		for _, candidate := range dispatched {
			if candidate == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("registered command %q is not dispatched", name)
		}
	}
}

func TestRegistryGlobalFlagsMatchParser(t *testing.T) {
	values := map[string]string{
		"config":  "config.yaml",
		"profile": "local",
		"format":  "json",
		"timeout": "10s",
	}
	for _, global := range cliRegistry.GlobalFlags {
		args := []string{"--" + global.Name}
		if global.TakesValue {
			args = append(args, values[global.Name])
		}
		args = append(args, "listen")
		_, rest, err := parseGlobalOptions(args)
		if err != nil {
			t.Errorf("registered global flag --%s is rejected: %v", global.Name, err)
			continue
		}
		if len(rest) == 0 || rest[0] != "listen" {
			t.Errorf("parsing --%s produced rest %v", global.Name, rest)
		}
	}
}

func TestRegistryAppliesPrefixGlobalPolicies(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantRest   []string
		wantConfig string
		wantFormat string
		wantTime   time.Duration
		verbose    bool
	}{
		{
			name:       "diagnostics drops output format",
			args:       []string{"--config", "site.yaml", "--format", "json", "--timeout", "5s", "--verbose", "test-connection", "--host", "192.0.2.10"},
			wantRest:   []string{"test-connection", "--host", "192.0.2.10"},
			wantConfig: "site.yaml",
			wantFormat: defaultFormat,
			wantTime:   5 * time.Second,
			verbose:    true,
		},
		{
			name:       "validation keeps output format override",
			args:       []string{"--config", "site.yaml", "--profile", "local", "--format", "csv", "--debug", "validate-config"},
			wantRest:   []string{"validate-config"},
			wantConfig: "site.yaml",
			wantFormat: "csv",
		},
		{
			name:       "local init drops every global",
			args:       []string{"--config", "site.yaml", "--verbose", "init-config", "--output", "new.yaml"},
			wantRest:   []string{"init-config", "--output", "new.yaml"},
			wantConfig: defaultConfigPath,
			wantFormat: defaultFormat,
		},
		{
			name:       "control keeps diagnostics but drops format",
			args:       []string{"--config", "site.yaml", "--format", "json", "--timeout", "3s", "--debug", "command", "single", "--ioa", "100", "--value", "on"},
			wantRest:   []string{"command", "single", "--ioa", "100", "--value", "on"},
			wantConfig: "site.yaml",
			wantFormat: defaultFormat,
			wantTime:   3 * time.Second,
		},
		{
			name:       "completion shell remains positional",
			args:       []string{"--profile", "local", "completions", "bash"},
			wantRest:   []string{"completions", "bash"},
			wantConfig: defaultConfigPath,
			wantFormat: defaultFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts, rest, err := parseGlobalOptions(test.args)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rest, test.wantRest) {
				t.Fatalf("rest = %#v, want %#v", rest, test.wantRest)
			}
			if opts.ConfigPath != test.wantConfig {
				t.Errorf("ConfigPath=%q, want %q", opts.ConfigPath, test.wantConfig)
			}
			if opts.Format != test.wantFormat {
				t.Errorf("Format=%q, want %q", opts.Format, test.wantFormat)
			}
			if opts.Timeout != test.wantTime {
				t.Errorf("Timeout=%s, want %s", opts.Timeout, test.wantTime)
			}
			if opts.Verbose != test.verbose {
				t.Errorf("Verbose=%v, want %v", opts.Verbose, test.verbose)
			}
		})
	}
}

func TestRegistryNestedControlFamiliesAndAccurateFlags(t *testing.T) {
	commandFamily := iecRegistryCommand(t, "command")
	setpointFamily := iecRegistryCommand(t, "setpoint")
	if commandFamily.LeadingArgs != 1 || setpointFamily.LeadingArgs != 1 {
		t.Fatalf("nested control families must preserve their subcommand positional")
	}
	assertSubcommands(t, commandFamily, "single", "double")
	assertSubcommands(t, setpointFamily, "normalized", "scaled", "float")

	for _, family := range []command.Command{commandFamily, setpointFamily} {
		for _, subcommand := range family.Subcommands {
			for _, name := range []string{"host", "port", "common-address", "ioa", "value"} {
				assertIECFlag(t, subcommand.Flags, name, true)
			}
			for _, name := range []string{"yes", "dry-run"} {
				assertIECFlag(t, subcommand.Flags, name, false)
			}
			for _, stale := range []string{"qualifier", "select"} {
				if hasIECFlag(subcommand.Flags, stale) {
					t.Errorf("%s %s still exposes stale --%s", family.Name, subcommand.Name, stale)
				}
			}
		}
	}

	interrogate := iecRegistryCommand(t, "interrogate")
	for _, stale := range []string{"qualifier", "duration"} {
		if hasIECFlag(interrogate.Flags, stale) {
			t.Errorf("interrogate still exposes stale --%s", stale)
		}
	}
	for _, name := range []string{"common-address", "ioa", "point"} {
		assertIECFlag(t, interrogate.Flags, name, true)
	}

	watch := iecRegistryCommand(t, "watch")
	for _, name := range []string{"interval", "stale-after", "ioa", "point", "cache"} {
		assertIECFlag(t, watch.Flags, name, true)
	}
	assertIECFlag(t, watch.Flags, "no-cache", false)

	clockSync := iecRegistryCommand(t, "clock-sync")
	assertIECFlag(t, clockSync.Flags, "time", true)
	assertIECFlag(t, clockSync.Flags, "yes", false)
	assertIECFlag(t, clockSync.Flags, "dry-run", false)
	for _, stale := range []string{"value", "qualifier", "select"} {
		if hasIECFlag(clockSync.Flags, stale) {
			t.Errorf("clock-sync still exposes stale --%s", stale)
		}
	}
}

func TestGeneratedCompletionsContainNestedSafetyAndWatchFlags(t *testing.T) {
	var out bytes.Buffer
	if err := completion.Write(&out, "bash", cliRegistry); err != nil {
		t.Fatal(err)
	}
	script := out.String()
	for _, want := range []string{
		"command:single", "setpoint:float", "--yes", "--dry-run", "--dump-frames",
		"--stale-after", "--no-cache", "--point", "--ioas",
		"complete -F _iec_104_cli_completion iec-104-cli",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("completion output missing %q", want)
		}
	}
	assertIECCaseOmits(t, script, "test-connection", "--format")
	assertIECCaseContains(t, script, "validate-config", "--config", "--profile", "--format")
	assertIECCaseOmits(t, script, "validate-config", "--verbose", "--debug", "--dump-frames")
	assertIECCaseContains(t, script, "init-config", "--output", "--force")
	assertIECCaseOmits(t, script, "init-config", "--config", "--format", "--verbose")
}

func iecRegistryCommand(t *testing.T, name string) command.Command {
	t.Helper()
	for _, registered := range cliRegistry.Commands {
		if registered.Name == name {
			return registered
		}
	}
	t.Fatalf("registry command %q not found", name)
	return command.Command{}
}

func assertSubcommands(t *testing.T, registered command.Command, names ...string) {
	t.Helper()
	found := map[string]bool{}
	for _, subcommand := range registered.Subcommands {
		found[subcommand.Name] = true
	}
	for _, name := range names {
		if !found[name] {
			t.Errorf("%s registry missing subcommand %q", registered.Name, name)
		}
	}
}

func assertIECFlag(t *testing.T, flags []command.Flag, name string, takesValue bool) {
	t.Helper()
	for _, flag := range flags {
		if flag.Name == name {
			if flag.TakesValue != takesValue {
				t.Fatalf("flag --%s TakesValue=%v, want %v", name, flag.TakesValue, takesValue)
			}
			return
		}
	}
	t.Fatalf("flag --%s not found", name)
}

func hasIECFlag(flags []command.Flag, name string) bool {
	for _, flag := range flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}

func assertIECCaseContains(t *testing.T, script, name string, values ...string) {
	t.Helper()
	line := iecBashCaseLine(t, script, name)
	for _, value := range values {
		if !strings.Contains(line, value) {
			t.Errorf("%s completion is missing %q: %s", name, value, line)
		}
	}
}

func assertIECCaseOmits(t *testing.T, script, name string, values ...string) {
	t.Helper()
	line := iecBashCaseLine(t, script, name)
	for _, value := range values {
		if strings.Contains(line, value) {
			t.Errorf("%s completion unexpectedly includes %q: %s", name, value, line)
		}
	}
}

func iecBashCaseLine(t *testing.T, script, name string) string {
	t.Helper()
	prefix := "    " + name + ") words="
	for _, line := range strings.Split(script, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("completion case for %q not found", name)
	return ""
}
