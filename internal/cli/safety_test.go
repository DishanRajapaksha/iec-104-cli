package cli

import "testing"

func TestControlSafetyDryRunByDefault(t *testing.T) {
	if (controlSafety{}).AllowsExecution() {
		t.Fatal("default safety allowed execution")
	}
}

func TestControlSafetyRequiresYesWithoutDryRun(t *testing.T) {
	if !(controlSafety{Yes: true}).AllowsExecution() {
		t.Fatal("--yes without --dry-run should allow execution")
	}
}

func TestControlSafetyRejectsConflictingFlags(t *testing.T) {
	if err := (controlSafety{DryRun: true, Yes: true}).Validate(); err == nil {
		t.Fatal("expected conflict error")
	}
}
