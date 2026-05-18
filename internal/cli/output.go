package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
)

func writePointValues(w io.Writer, format string, values []iec104.PointValue) error {
	switch format {
	case "table":
		return writePointValueTable(w, values)
	case "text":
		return writePointValueText(w, values)
	case "json":
		return json.NewEncoder(w).Encode(values)
	case "jsonl":
		enc := json.NewEncoder(w)
		for _, value := range values {
			if err := enc.Encode(value); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writePointValueTable(w io.Writer, values []iec104.PointValue) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TIME\tCA\tIOA\tNAME\tTYPE\tVALUE\tUNIT\tCAUSE\tQUALITY"); err != nil {
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintf(tw, "%s\t%d\t%d\t%s\t%s\t%v\t%s\t%s\t%s\n",
			formatPointTime(value.Timestamp),
			value.CommonAddress,
			value.IOA,
			value.Name,
			value.Type,
			value.Value,
			value.Unit,
			value.Cause,
			value.Quality.Display(),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writePointValueText(w io.Writer, values []iec104.PointValue) error {
	for _, value := range values {
		name := value.Name
		if name == "" {
			name = fmt.Sprintf("%d", value.IOA)
		}
		if _, err := fmt.Fprintf(w, "%s=%v", name, value.Value); err != nil {
			return err
		}
		if value.Unit != "" {
			if _, err := fmt.Fprintf(w, " %s", value.Unit); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func formatPointTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
