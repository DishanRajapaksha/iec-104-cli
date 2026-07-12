package cli

import (
	"os"
	"os/exec"
	"testing"

	"github.com/DishanRajapaksha/industrial-cli-kit/contracttest"
)

const cliHelperEnvironment = "IEC104_CLI_TEST_HELPER"

func TestSharedCommandContract(t *testing.T) {
	contracttest.Baseline(t, func(args ...string) contracttest.Result {
		commandArgs := append([]string{"-test.run=TestCLIHelperProcess", "--"}, args...)
		cmd := exec.Command(os.Args[0], commandArgs...)
		cmd.Env = append(os.Environ(), cliHelperEnvironment+"=1")
		stdout, err := cmd.Output()
		result := contracttest.Result{Stdout: string(stdout)}
		if err == nil {
			return result
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Code = exitErr.ExitCode()
			result.Stderr = string(exitErr.Stderr)
			return result
		}
		t.Fatalf("run IEC 104 CLI helper: %v", err)
		return contracttest.Result{}
	})
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv(cliHelperEnvironment) != "1" {
		return
	}
	separator := 0
	for index, arg := range os.Args {
		if arg == "--" {
			separator = index + 1
			break
		}
	}
	if separator == 0 {
		os.Exit(1)
	}
	os.Exit(Run(os.Args[separator:]))
}
