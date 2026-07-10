package cli

import (
	"testing"

	"github.com/DishanRajapaksha/iec-104-cli/internal/exitcode"
)

func TestSnapshotFormatContract(t *testing.T) {
	for _, format := range []string{"table", "text", "json", "csv"} {
		if err := validateSnapshotFormat(format); err != nil {
			t.Fatalf("snapshot format %q rejected: %v", format, err)
		}
	}
	if err := validateSnapshotFormat("jsonl"); err == nil {
		t.Fatal("snapshot commands must reject jsonl")
	}
}

func TestStreamFormatContract(t *testing.T) {
	for _, format := range []string{"text", "jsonl", "csv"} {
		if err := validateStreamFormat(format); err != nil {
			t.Fatalf("stream format %q rejected: %v", format, err)
		}
	}
	for _, format := range []string{"table", "json"} {
		if err := validateStreamFormat(format); err == nil {
			t.Fatalf("stream format %q must be rejected", format)
		}
	}
}

func TestSharedExitCodeContract(t *testing.T) {
	if exitcode.TransportConnectionError != 3 || exitcode.ProtocolRequestError != 4 {
		t.Fatal("transport and protocol exit codes changed")
	}
	if exitcode.WriteControlRejected != 7 || exitcode.OperationTimeout != 8 || exitcode.OutputError != 9 {
		t.Fatal("write, timeout, or output exit codes changed")
	}
	if exitcode.IEC104SessionError != exitcode.ProtocolRequestError || exitcode.UnsupportedASDU != exitcode.ProtocolRequestError {
		t.Fatal("IEC 104 protocol aliases do not use the shared protocol error code")
	}
	if exitcode.InterrogationTimeout != exitcode.OperationTimeout || exitcode.CommandTimeout != exitcode.OperationTimeout {
		t.Fatal("IEC 104 timeout aliases do not use the shared timeout code")
	}
}
