package cli

import (
	"io"

	sharedhelp "github.com/DishanRajapaksha/industrial-cli-kit/help"
)

func writeRegistryHelp(w io.Writer) {
	_ = sharedhelp.Write(w, cliRegistry, sharedhelp.Options{
		Description: "iec-104-cli is a script-friendly IEC 60870-5-104 command-line client.",
		Usage: []string{"iec-104-cli [global flags] <command> [flags]"},
		Examples: []string{
			"iec-104-cli init-config",
			"iec-104-cli validate-config --profile local",
			"iec-104-cli test-connection --host 127.0.0.1 --port 2404",
			"iec-104-cli interrogate --profile local --duration 5s",
			"iec-104-cli monitor --profile local --format jsonl",
			"iec-104-cli read --profile local --common-address 1 --ioa 1001",
			"iec-104-cli command single --profile local --common-address 1 --ioa 1002 --value on --yes",
			"iec-104-cli setpoint float --profile local --common-address 1 --ioa 1003 --value 12.5 --yes",
			"iec-104-cli clock-sync --profile local --yes",
			"iec-104-cli completions zsh",
		},
	})
}
