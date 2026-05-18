package exitcode

const (
	Success = 0
	GeneralError = 1
	ConfigError = 2
	TCPConnectionError = 3
	IEC104SessionError = 4
	InterrogationTimeout = 5
	UnsupportedASDU = 6
	CommandRejected = 7
	CommandTimeout = 8
	OutputError = 9
)
