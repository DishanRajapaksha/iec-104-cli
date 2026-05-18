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
