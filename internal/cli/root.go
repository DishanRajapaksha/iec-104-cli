package cli

import (
	"fmt"
	"os"

	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
)

const appName = "iec-104-cli"

// Main is the process entrypoint for the CLI package.
func Main() {
	os.Exit(Run(os.Args[1:]))
}

// Run executes the CLI with the provided arguments and returns a stable exit code.
func Run(args []string) int {
	if len(args) == 0 {
		printHelp(os.Stdout)
		return exitcode.Success
	}

	switch args[0] {
	case "help", "--help", "-h":
		printHelp(os.Stdout)
		return exitcode.Success
	case "version", "--version", "-v":
		fmt.Fprintf(os.Stdout, "%s development\n", appName)
		return exitcode.Success
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printHelp(os.Stderr)
		return exitcode.GeneralError
	}
}

func printHelp(out *os.File) {
	fmt.Fprintf(out, `%s is a script-friendly IEC 60870-5-104 command-line client.

Usage:
  %s <command> [flags]

Available commands:
  help       Show this help message
  version    Show version information

Planned commands:
  validate-config
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
