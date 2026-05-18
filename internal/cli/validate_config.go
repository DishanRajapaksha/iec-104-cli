package cli

import (
	"fmt"
	"os"

	"github.com/DishanRajapaksha/iec-104-cli/internal/config"
	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
)

func runValidateConfig(opts globalOptions) int {
	overrides := config.Overrides{
		Profile: opts.Profile,
	}
	if opts.FormatSet {
		overrides.Format = opts.Format
	}
	if opts.TimeoutSet {
		overrides.Timeout = opts.Timeout
	}

	cfg, loaded, err := config.Load(opts.ConfigPath, overrides)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}
	if !loaded {
		fmt.Fprintf(os.Stderr, "config file %q was not found\n", opts.ConfigPath)
		return exitcode.ConfigError
	}

	if err := config.Validate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitcode.ConfigError
	}

	fmt.Fprintf(os.Stdout, "config %q is valid\n", opts.ConfigPath)
	return exitcode.Success
}
