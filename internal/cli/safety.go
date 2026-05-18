package cli

import "fmt"

type controlSafety struct {
	DryRun bool
	Yes    bool
}

func (s controlSafety) AllowsExecution() bool {
	return s.Yes && !s.DryRun
}

func (s controlSafety) Validate() error {
	if s.DryRun && s.Yes {
		return fmt.Errorf("--dry-run and --yes cannot be used together")
	}
	return nil
}
