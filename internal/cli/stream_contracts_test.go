package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
)

func TestCSVStreamWritesHeaderOnce(t *testing.T) {
	var out bytes.Buffer
	if err := writePointValueCSVHeader(&out); err != nil {
		t.Fatal(err)
	}
	values := []iec104.PointValue{{CommonAddress: 1, IOA: 1001, Name: "active_power", Value: 42.0}}
	if err := writeStreamPointValues(&out, "csv", values); err != nil {
		t.Fatal(err)
	}
	if err := writeStreamPointValues(&out, "csv", values); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(out.String(), "time,common_address,ioa,name,type,value,unit,cause,quality"); count != 1 {
		t.Fatalf("CSV header count = %d, output: %q", count, out.String())
	}
}
