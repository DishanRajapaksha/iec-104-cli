package iec104

import "errors"

var (
	ErrTCPConnection        = errors.New("tcp connection error")
	ErrSession              = errors.New("iec 104 session error")
	ErrInterrogationTimeout = errors.New("interrogation timeout")
	ErrUnsupportedType      = errors.New("unsupported asdu or type")
	ErrCommandRejected      = errors.New("command rejected")
	ErrCommandTimeout       = errors.New("command timeout")
)
