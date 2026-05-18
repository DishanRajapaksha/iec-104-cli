package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
)

func TestWritePointValuesTable(t *testing.T) {
	var out bytes.Buffer

	if err := writePointValues(&out, "table", samplePointValues()); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"TIME", "CA", "IOA", "active_power", "M_ME_NC_1", "12.34", "MW", "spontaneous", "good"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output %q does not contain %q", got, want)
		}
	}
}

func TestWritePointValuesText(t *testing.T) {
	var out bytes.Buffer

	if err := writePointValues(&out, "text", samplePointValues()); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "active_power=12.34 MW") {
		t.Fatalf("text output = %q, want active_power value", got)
	}
	if !strings.Contains(got, "ioa:2001=true") {
		t.Fatalf("text output = %q, want fallback IOA name", got)
	}
}

func TestWritePointValuesJSON(t *testing.T) {
	var out bytes.Buffer

	if err := writePointValues(&out, "json", samplePointValues()); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}

	var decoded []iec104.PointValue
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("json output is invalid: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("decoded len = %d, want 2", len(decoded))
	}
	if decoded[0].Name != "active_power" {
		t.Fatalf("decoded[0].Name = %q, want active_power", decoded[0].Name)
	}
}

func TestWritePointValuesJSONL(t *testing.T) {
	var out bytes.Buffer

	if err := writePointValues(&out, "jsonl", samplePointValues()); err != nil {
		t.Fatalf("writePointValues returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl line count = %d, want 2; output=%q", len(lines), out.String())
	}
	for _, line := range lines {
		var decoded iec104.PointValue
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("jsonl line %q is invalid JSON: %v", line, err)
		}
	}
}

func TestWritePointValuesUnsupportedFormat(t *testing.T) {
	var out bytes.Buffer

	if err := writePointValues(&out, "xml", samplePointValues()); err == nil {
		t.Fatal("writePointValues returned nil error for unsupported format")
	}
}

func samplePointValues() []iec104.PointValue {
	return []iec104.PointValue{
		{
			Timestamp:     time.Date(2026, 5, 18, 12, 34, 56, 0, time.UTC),
			CommonAddress: 1,
			IOA:           1001,
			Name:          "active_power",
			Type:          "M_ME_NC_1",
			Cause:         "spontaneous",
			Value:         12.34,
			Unit:          "MW",
			Quality:       iec104.Quality{},
			RawTypeID:     13,
		},
		{
			Timestamp:     time.Date(2026, 5, 18, 12, 35, 0, 0, time.UTC),
			CommonAddress: 1,
			IOA:           2001,
			Type:          "M_SP_NA_1",
			Cause:         "spontaneous",
			Value:         true,
			Quality:       iec104.Quality{Blocked: true},
			RawTypeID:     1,
		},
	}
}
