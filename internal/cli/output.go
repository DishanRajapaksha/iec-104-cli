package cli

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
	shared "github.com/DishanRajapaksha/industrial-cli-kit/output"
)

func writePointValues(w io.Writer, format string, values []iec104.PointValue) error {
	switch format {
	case shared.FormatTable:
		return writePointValueTable(w, values)
	case shared.FormatText:
		return writePointValueText(w, values)
	case shared.FormatJSON:
		return shared.WriteJSON(w, values)
	case shared.FormatJSONL:
		for _, value := range values {
			if err := shared.WriteJSONLine(w, value); err != nil {
				return err
			}
		}
		return nil
	case shared.FormatCSV:
		return writePointValueCSV(w, values)
	default:
		return fmt.Errorf("%w: unsupported output format %q", shared.ErrOutput, format)
	}
}

func writeStreamPointValues(w io.Writer, format string, values []iec104.PointValue) error {
	if format == shared.FormatCSV {
		return writePointValueCSVRows(w, values)
	}
	return writePointValues(w, format, values)
}

func writePointValueTable(w io.Writer, values []iec104.PointValue) error {
	rows := make([][]string, 0, len(values))
	for _, value := range values {
		rows = append(rows, []string{
			formatPointTime(value.Timestamp),
			strconv.FormatUint(uint64(value.CommonAddress), 10),
			strconv.FormatUint(uint64(value.IOA), 10),
			value.Name,
			value.Type,
			fmt.Sprint(value.Value),
			value.Unit,
			value.Cause,
			value.Quality.Display(),
		})
	}
	return shared.WriteTable(w, []string{"TIME", "CA", "IOA", "NAME", "TYPE", "VALUE", "UNIT", "CAUSE", "QUALITY"}, rows)
}

func writePointValueText(w io.Writer, values []iec104.PointValue) error {
	for _, value := range values {
		name := value.Name
		if name == "" {
			name = fmt.Sprintf("%d", value.IOA)
		}
		line := fmt.Sprintf("%s=%v", name, value.Value)
		if value.Unit != "" {
			line += " " + value.Unit
		}
		if err := shared.WriteText(w, line); err != nil {
			return err
		}
	}
	return nil
}

func writePointValueCSV(w io.Writer, values []iec104.PointValue) error {
	return shared.WriteCSV(w, pointValueCSVHeader(), pointValueCSVRows(values))
}

func writePointValueCSVHeader(w io.Writer) error {
	return shared.WriteCSV(w, pointValueCSVHeader(), nil)
}

func writePointValueCSVRows(w io.Writer, values []iec104.PointValue) error {
	return shared.WriteCSVRows(w, pointValueCSVRows(values))
}

func pointValueCSVRows(values []iec104.PointValue) [][]string {
	rows := make([][]string, 0, len(values))
	for _, value := range values {
		rows = append(rows, []string{
			formatPointTime(value.Timestamp),
			strconv.FormatUint(uint64(value.CommonAddress), 10),
			strconv.FormatUint(uint64(value.IOA), 10),
			value.Name,
			value.Type,
			fmt.Sprint(value.Value),
			value.Unit,
			value.Cause,
			value.Quality.Display(),
		})
	}
	return rows
}

func pointValueCSVHeader() []string {
	return []string{"time", "common_address", "ioa", "name", "type", "value", "unit", "cause", "quality"}
}

func formatPointTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
