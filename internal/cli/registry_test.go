package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/industrial-cli-kit/completion"
)

func TestRegistryMatchesDispatcher(t *testing.T) {
	dispatched := []string{
		"validate-config", "generate-configs", "init-config", "test-connection", "status",
		"listen", "monitor", "interrogate", "watch", "read", "command", "setpoint",
		"clock-sync", "completions", "help", "version",
	}
	registered := map[string]bool{}
	for _, command := range cliRegistry.Commands {
		if registered[command.Name] {
			t.Fatalf("duplicate registry command %q", command.Name)
		}
		registered[command.Name] = true
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
		"config": "config.yaml",
		"profile": "local",
		"format": "json",
		"timeout": "10s",
	}
	for _, global := range cliRegistry.GlobalFlags {
		args := []string{"--" + global.Name}
		if global.TakesValue {
			args = append(args, values[global.Name])
		}
		args = append(args, "status")
		_, rest, err := parseGlobalOptions(args)
		if err != nil {
			t.Errorf("registered global flag --%s is rejected: %v", global.Name, err)
			continue
		}
		if len(rest) == 0 || rest[0] != "status" {
			t.Errorf("parsing --%s produced rest %v", global.Name, rest)
		}
	}
}

func TestRegistryIncludesNestedControlFamilies(t *testing.T) {
	want := map[string]map[string]bool{
		"command": {"single": false, "double": false},
		"setpoint": {"normalized": false, "scaled": false, "float": false},
	}
	for _, command := range cliRegistry.Commands {
		targets, ok := want[command.Name]
		if !ok {
			continue
		}
		for _, subcommand := range command.Subcommands {
			if _, exists := targets[subcommand.Name]; exists {
				targets[subcommand.Name] = true
			}
		}
	}
	for command, targets := range want {
		for target, found := range targets {
			if !found {
				t.Errorf("%s registry missing %q", command, target)
			}
		}
	}
}

func TestGeneratedCompletionsContainNestedSafetyFlags(t *testing.T) {
	var out bytes.Buffer
	if err := completion.Write(&out, "bash", cliRegistry); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"command:single", "setpoint:float", "--yes", "--dry-run", "--dump-frames"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("completion output missing %q", want)
		}
	}
}
