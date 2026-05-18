package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
)

func writePointValues(out io.Writer, format string, values []iec104.PointValue) error {
	switch format {
	case "table":
		return writePointValuesTable(out, values)
	case "text":
		return writePointValuesText(out, values)
	case "json":
		return writePointValuesJSON(out, values)
	case "jsonl":
		return writePointValuesJSONL(out, values)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writePointValuesTable(out io.Writer, values []iec104.PointValue) error {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TIME\tCA\tIOA\tNAME\tTYPE\tVALUE\tUNIT\tCAUSE\tQUALITY"); err != nil {
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%d\t%d\t%s\t%s\t%v\t%s\t%s\t%s\n",
			formatTimestamp(value.Timestamp),
			value.CommonAddress,
			value.IOA,
			value.Name,
			value.Type,
			value.Value,
			value.Unit,
			value.Cause,
			value.Quality.String(),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writePointValuesText(out io.Writer, values []iec104.PointValue) error {
	for _, value := range values {
		name := value.Name
		if name == "" {
			name = fmt.Sprintf("ioa:%d", value.IOA)
		}
		unit := ""
		if value.Unit != "" {
			unit = " " + value.Unit
		}
		if _, err := fmt.Fprintf(out, "%s=%v%s\n", name, value.Value, unit); err != nil {
			return err
		}
	}
	return nil
}

func writePointValuesJSON(out io.Writer, values []iec104.PointValue) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(values)
}

func writePointValuesJSONL(out io.Writer, values []iec104.PointValue) error {
	encoder := json.NewEncoder(out)
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			return err
		}
	}
	return nil
}

func formatTimestamp(timestamp time.Time) string {
	if timestamp.IsZero() {
		return ""
	}
	return timestamp.Format("2006-01-02 15:04:05")
}
