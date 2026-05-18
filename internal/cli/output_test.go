package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
)

func samplePointValue() iec104.PointValue {
	return iec104.PointValue{
		Timestamp:     time.Date(2026, 5, 18, 12, 34, 56, 0, time.UTC),
		CommonAddress: 1,
		IOA:           1001,
		Name:          "active_power",
		Type:          "float",
		Cause:         "spontaneous",
		Value:         12.34,
		Unit:          "MW",
		Quality:       iec104.Quality{},
		RawTypeID:     13,
	}
}

func TestWritePointValuesTable(t *testing.T) {
	var out bytes.Buffer
	if err := writePointValues(&out, "table", []iec104.PointValue{samplePointValue()}); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"TIME", "active_power", "12.34", "good"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output %q does not contain %q", got, want)
		}
	}
}

func TestWritePointValuesText(t *testing.T) {
	var out bytes.Buffer
	if err := writePointValues(&out, "text", []iec104.PointValue{samplePointValue()}); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}
	if got, want := out.String(), "active_power=12.34 MW\n"; got != want {
		t.Fatalf("text output = %q, want %q", got, want)
	}
}

func TestWritePointValuesJSON(t *testing.T) {
	var out bytes.Buffer
	if err := writePointValues(&out, "json", []iec104.PointValue{samplePointValue()}); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}
	var decoded []iec104.PointValue
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Name != "active_power" {
		t.Fatalf("decoded JSON = %#v", decoded)
	}
}

func TestWritePointValuesJSONL(t *testing.T) {
	var out bytes.Buffer
	if err := writePointValues(&out, "jsonl", []iec104.PointValue{samplePointValue(), samplePointValue()}); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("JSONL line count = %d, want 2; output %q", len(lines), out.String())
	}
	for _, line := range lines {
		var decoded iec104.PointValue
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("JSONL line is invalid JSON: %v", err)
		}
	}
}
