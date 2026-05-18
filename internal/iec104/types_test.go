package iec104

import "testing"

func TestQualityDisplayGood(t *testing.T) {
	if got := (Quality{}).Display(); got != "good" {
		t.Fatalf("Display() = %q, want good", got)
	}
}

func TestQualityDisplayCombinedFlags(t *testing.T) {
	quality := Quality{Invalid: true, Blocked: true}
	if got, want := quality.Display(), "invalid,blocked"; got != want {
		t.Fatalf("Display() = %q, want %q", got, want)
	}
}
