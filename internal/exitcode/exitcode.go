package exitcode

const (
	Success                  = 0
	GeneralError             = 1
	ConfigError              = 2
	TransportConnectionError = 3
	ProtocolRequestError     = 4
	WriteControlRejected     = 7
	OperationTimeout         = 8
	OutputError              = 9

	// Protocol-specific aliases retain descriptive names without changing the
	// shared cross-CLI meanings of the numeric values.
	TCPConnectionError   = TransportConnectionError
	IEC104SessionError   = ProtocolRequestError
	UnsupportedASDU      = ProtocolRequestError
	InterrogationTimeout = OperationTimeout
	CommandRejected      = WriteControlRejected
	CommandTimeout       = OperationTimeout
)
