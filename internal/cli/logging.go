package cli

import (
	"fmt"
	"os"
)

func logVerbose(opts globalOptions, format string, args ...any) {
	if opts.Verbose || opts.Debug {
		fmt.Fprintf(os.Stderr, "verbose: "+format+"\n", args...)
	}
}

func logDebug(opts globalOptions, format string, args ...any) {
	if opts.Debug {
		fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
	}
}
