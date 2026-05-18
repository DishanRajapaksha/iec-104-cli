package cli

import (
	"testing"

	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
)

func TestRunHelp(t *testing.T) {
	if got := Run([]string{"help"}); got != exitcode.Success {
		t.Fatalf("Run(help) = %d, want %d", got, exitcode.Success)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if got := Run([]string{"bogus"}); got != exitcode.GeneralError {
		t.Fatalf("Run(bogus) = %d, want %d", got, exitcode.GeneralError)
	}
}
