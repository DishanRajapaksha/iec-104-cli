package cli

import (
	"io"
	"strings"

	"github.com/DishanRajapaksha/industrial-cli-kit/completion"
)

func writeCompletion(w io.Writer, shell string) error {
	return completion.Write(w, strings.ToLower(strings.TrimSpace(shell)), cliRegistry)
}
