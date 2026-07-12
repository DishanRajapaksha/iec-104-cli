package exitcode

import shared "github.com/DishanRajapaksha/industrial-cli-kit/exitcode"

const (
	Success                  = int(shared.Success)
	GeneralError             = int(shared.General)
	ConfigError              = int(shared.Config)
	TransportConnectionError = int(shared.Connection)
	ProtocolRequestError     = int(shared.Request)
	WriteControlRejected     = int(shared.Rejected)
	OperationTimeout         = int(shared.Timeout)
	OutputError              = int(shared.Output)

	// Protocol-specific aliases retain descriptive names without changing the
	// shared cross-CLI meanings of the numeric values.
	TCPConnectionError   = TransportConnectionError
	IEC104SessionError   = ProtocolRequestError
	UnsupportedASDU      = ProtocolRequestError
	InterrogationTimeout = OperationTimeout
	CommandRejected      = WriteControlRejected
	CommandTimeout       = OperationTimeout
)
