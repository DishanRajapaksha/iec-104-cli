from pathlib import Path


def update_function(text: str, name: str, transform) -> str:
    start = text.find(f"func {name}(")
    if start < 0:
        raise SystemExit(f"function {name} not found")
    end = text.find("\nfunc ", start + 1)
    if end < 0:
        end = len(text)
    segment = transform(text[start:end])
    return text[:start] + segment + text[end:]


output = Path("internal/cli/output.go")
text = output.read_text()
marker = '''func writePointValues(w io.Writer, format string, values []iec104.PointValue) error {
\tswitch format {'''
replacement = '''func writePointValues(w io.Writer, format string, values []iec104.PointValue) error {
\tswitch format {'''
if marker not in text:
    raise SystemExit("writePointValues marker not found")

insert_after = '''\tdefault:
\t\treturn fmt.Errorf("unsupported output format %q", format)
\t}
}
'''
stream_helper = insert_after + '''
func writeStreamPointValues(w io.Writer, format string, values []iec104.PointValue) error {
\tif format == "csv" {
\t\treturn writePointValueCSVRows(w, values)
\t}
\treturn writePointValues(w, format, values)
}
'''
if text.count(insert_after) != 1:
    raise SystemExit("writePointValues end marker not unique")
text = text.replace(insert_after, stream_helper, 1)

start = text.find("func writePointValueCSV(")
if start < 0:
    raise SystemExit("writePointValueCSV not found")
end = text.find("\nfunc ", start + 1)
if end < 0:
    end = len(text)
new_csv = r'''func writePointValueCSV(w io.Writer, values []iec104.PointValue) error {
	if err := writePointValueCSVHeader(w); err != nil {
		return err
	}
	return writePointValueCSVRows(w, values)
}

func writePointValueCSVHeader(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(pointValueCSVHeader()); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func writePointValueCSVRows(w io.Writer, values []iec104.PointValue) error {
	cw := csv.NewWriter(w)
	for _, value := range values {
		if err := cw.Write([]string{
			formatPointTime(value.Timestamp),
			strconv.FormatUint(uint64(value.CommonAddress), 10),
			strconv.FormatUint(uint64(value.IOA), 10),
			value.Name,
			value.Type,
			fmt.Sprint(value.Value),
			value.Unit,
			value.Cause,
			value.Quality.Display(),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func pointValueCSVHeader() []string {
	return []string{"time", "common_address", "ioa", "name", "type", "value", "unit", "cause", "quality"}
}
'''
text = text[:start] + new_csv.rstrip() + "\n" + text[end:]
output.write_text(text)

root = Path("internal/cli/root.go")
text = root.read_text()

def fix_watch(segment: str) -> str:
    marker = '''\tticker := time.NewTicker(interval)
\tdefer ticker.Stop()
\tfor {'''
    replacement = '''\tif format == "csv" {
\t\tif err := writePointValueCSVHeader(os.Stdout); err != nil {
\t\t\tfmt.Fprintln(os.Stderr, err)
\t\t\treturn exitcode.OutputError
\t\t}
\t}
\tticker := time.NewTicker(interval)
\tdefer ticker.Stop()
\tfor {'''
    if segment.count(marker) != 1:
        raise SystemExit("watch ticker marker not found")
    segment = segment.replace(marker, replacement, 1)
    old = "writePointValues(os.Stdout, format, filtered)"
    if segment.count(old) != 1:
        raise SystemExit("watch writer call not found")
    return segment.replace(old, "writeStreamPointValues(os.Stdout, format, filtered)", 1)


def fix_listen(segment: str) -> str:
    marker = '''\tlogVerbose(opts, "listening on %s:%d", cfg.Connection.Host, cfg.Connection.Port)
\tlogDebug(opts, "listen duration=%s common_address=%d ioa=%d point=%q format=%s", duration, commonAddress, ioa, pointName, format)
\terr = runListenWithReconnect'''
    replacement = '''\tlogVerbose(opts, "listening on %s:%d", cfg.Connection.Host, cfg.Connection.Port)
\tlogDebug(opts, "listen duration=%s common_address=%d ioa=%d point=%q format=%s", duration, commonAddress, ioa, pointName, format)
\tif format == "csv" {
\t\tif err := writePointValueCSVHeader(os.Stdout); err != nil {
\t\t\tfmt.Fprintln(os.Stderr, err)
\t\t\treturn exitcode.OutputError
\t\t}
\t}
\terr = runListenWithReconnect'''
    if segment.count(marker) != 1:
        raise SystemExit("listen header marker not found")
    segment = segment.replace(marker, replacement, 1)
    old = "writePointValues(os.Stdout, format, []iec104.PointValue{enriched})"
    if segment.count(old) != 1:
        raise SystemExit("listen writer call not found")
    return segment.replace(old, "writeStreamPointValues(os.Stdout, format, []iec104.PointValue{enriched})", 1)

text = update_function(text, "runWatch", fix_watch)
text = update_function(text, "runListen", fix_listen)
root.write_text(text)

Path("internal/cli/stream_contracts_test.go").write_text(r'''package cli

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
''')
