package iec104

import (
	"context"
	"time"
)

type Client interface {
	Connect(ctx context.Context) error
	Close() error
	TestConnection(ctx context.Context) error
	Interrogate(ctx context.Context, commonAddress uint16) ([]PointValue, error)
	Listen(ctx context.Context, handler func(PointValue)) error
	Read(ctx context.Context, commonAddress uint16, ioa uint32) (PointValue, error)
	SendSingleCommand(ctx context.Context, commonAddress uint16, ioa uint32, value bool) error
	SendDoubleCommand(ctx context.Context, commonAddress uint16, ioa uint32, value uint8) error
	SendSetpoint(ctx context.Context, commonAddress uint16, ioa uint32, kind string, value any) error
}

type ClientConfig struct {
	Host              string
	Port              int
	Timeout           time.Duration
	Reconnect         bool
	ReconnectInterval time.Duration
	OriginatorAddress uint8
	Debug             bool
}
