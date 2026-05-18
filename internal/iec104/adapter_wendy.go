package iec104

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wendy512/go-iecp5/asdu"
	wendyclient "github.com/wendy512/iec104/client"
)

type WendyClient struct {
	cfg    ClientConfig
	client *wendyclient.Client
	events chan PointValue
}

func NewWendyClient(cfg ClientConfig) *WendyClient {
	return &WendyClient{
		cfg:    cfg,
		events: make(chan PointValue, 256),
	}
}

func (c *WendyClient) Connect(ctx context.Context) error {
	settings := wendyclient.NewSettings()
	settings.Host = c.cfg.Host
	settings.Port = c.cfg.Port
	settings.AutoConnect = c.cfg.Reconnect
	settings.ReconnectInterval = c.cfg.ReconnectInterval
	if c.cfg.Timeout > 0 {
		settings.Cfg104.ConnectTimeout0 = c.cfg.Timeout
	}
	settings.Params = asdu.ParamsWide
	settings.Params.OrigAddress = asdu.OriginAddr(c.cfg.OriginatorAddress)

	c.client = wendyclient.New(settings, &wendyCallback{events: c.events})
	return runWithContext(ctx, c.client.Connect, mapWendyError)
}

func (c *WendyClient) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *WendyClient) TestConnection(ctx context.Context) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	return c.Close()
}

func (c *WendyClient) Interrogate(ctx context.Context, commonAddress uint16) ([]PointValue, error) {
	if c.client == nil || !c.client.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}
	if err := c.client.SendInterrogationCmd(commonAddress); err != nil {
		return nil, mapWendyError(err)
	}

	var values []PointValue
	for {
		select {
		case <-ctx.Done():
			return values, ctx.Err()
		case value := <-c.events:
			values = append(values, value)
		}
	}
}

func (c *WendyClient) Listen(ctx context.Context, handler func(PointValue)) error {
	if c.client == nil || !c.client.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case value := <-c.events:
			handler(value)
		}
	}
}

func (c *WendyClient) Read(ctx context.Context, commonAddress uint16, ioa uint32) (PointValue, error) {
	if c.client == nil || !c.client.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return PointValue{}, err
		}
	}
	if err := c.client.SendReadCmd(commonAddress, uint(ioa)); err != nil {
		return PointValue{}, mapWendyError(err)
	}
	for {
		select {
		case <-ctx.Done():
			return PointValue{}, ctx.Err()
		case value := <-c.events:
			if value.CommonAddress == commonAddress && value.IOA == ioa {
				return value, nil
			}
		}
	}
}

func (c *WendyClient) SendSingleCommand(ctx context.Context, commonAddress uint16, ioa uint32, value bool) error {
	if c.client == nil || !c.client.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return err
		}
	}
	return mapWendyError(c.client.SendCmd(commonAddress, asdu.C_SC_NA_1, asdu.InfoObjAddr(ioa), value))
}

func (c *WendyClient) SendDoubleCommand(ctx context.Context, commonAddress uint16, ioa uint32, value uint8) error {
	if c.client == nil || !c.client.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return err
		}
	}
	return mapWendyError(c.client.SendCmd(commonAddress, asdu.C_DC_NA_1, asdu.InfoObjAddr(ioa), value))
}

func (c *WendyClient) SendSetpoint(ctx context.Context, commonAddress uint16, ioa uint32, kind string, value any) error {
	if c.client == nil || !c.client.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return err
		}
	}
	typeID := asdu.C_SE_NC_1
	switch kind {
	case "normalized":
		typeID = asdu.C_SE_NA_1
	case "scaled":
		typeID = asdu.C_SE_NB_1
	case "float":
		typeID = asdu.C_SE_NC_1
	default:
		return fmt.Errorf("%w: unsupported setpoint kind %q", ErrUnsupportedType, kind)
	}
	return mapWendyError(c.client.SendCmd(commonAddress, typeID, asdu.InfoObjAddr(ioa), value))
}

type wendyCallback struct {
	events chan<- PointValue
}

func (c *wendyCallback) OnInterrogation(packet *asdu.ASDU) error {
	return c.OnASDU(packet)
}

func (c *wendyCallback) OnCounterInterrogation(packet *asdu.ASDU) error {
	return c.OnASDU(packet)
}

func (c *wendyCallback) OnRead(packet *asdu.ASDU) error {
	return c.OnASDU(packet)
}

func (c *wendyCallback) OnTestCommand(_ *asdu.ASDU) error {
	return nil
}

func (c *wendyCallback) OnClockSync(_ *asdu.ASDU) error {
	return nil
}

func (c *wendyCallback) OnResetProcess(_ *asdu.ASDU) error {
	return nil
}

func (c *wendyCallback) OnDelayAcquisition(_ *asdu.ASDU) error {
	return nil
}

func (c *wendyCallback) OnASDU(packet *asdu.ASDU) error {
	// The backend invokes callbacks from its protocol goroutines. Convert to
	// local immutable values here so the rest of the CLI never sees backend ASDU
	// pointers or has to care which goroutine delivered the packet.
	for _, value := range pointValuesFromASDU(packet) {
		select {
		case c.events <- value:
		default:
		}
	}
	return nil
}

func pointValuesFromASDU(packet *asdu.ASDU) []PointValue {
	switch wendyclient.GetDataType(packet.Type) {
	case wendyclient.SinglePoint:
		points := packet.GetSinglePoint()
		values := make([]PointValue, 0, len(points))
		for _, point := range points {
			values = append(values, pointValue(packet, uint32(point.Ioa), "single_point", point.Value, point.Qds, point.Time))
		}
		return values
	case wendyclient.DoublePoint:
		points := packet.GetDoublePoint()
		values := make([]PointValue, 0, len(points))
		for _, point := range points {
			values = append(values, pointValue(packet, uint32(point.Ioa), "double_point", point.Value, point.Qds, point.Time))
		}
		return values
	case wendyclient.MeasuredValueScaled:
		points := packet.GetMeasuredValueScaled()
		values := make([]PointValue, 0, len(points))
		for _, point := range points {
			values = append(values, pointValue(packet, uint32(point.Ioa), "scaled", point.Value, point.Qds, point.Time))
		}
		return values
	case wendyclient.MeasuredValueNormal:
		points := packet.GetMeasuredValueNormal()
		values := make([]PointValue, 0, len(points))
		for _, point := range points {
			values = append(values, pointValue(packet, uint32(point.Ioa), "normalized", point.Value, point.Qds, point.Time))
		}
		return values
	case wendyclient.MeasuredValueFloat:
		points := packet.GetMeasuredValueFloat()
		values := make([]PointValue, 0, len(points))
		for _, point := range points {
			values = append(values, pointValue(packet, uint32(point.Ioa), "float", point.Value, point.Qds, point.Time))
		}
		return values
	case wendyclient.IntegratedTotals:
		points := packet.GetIntegratedTotals()
		values := make([]PointValue, 0, len(points))
		for _, point := range points {
			values = append(values, pointValue(packet, uint32(point.Ioa), "integrated_total", point.Value.CounterReading, qualityFromBinaryCounter(point.Value), point.Time))
		}
		return values
	default:
		return nil
	}
}

func pointValue(packet *asdu.ASDU, ioa uint32, typ string, value any, quality any, timestamp time.Time) PointValue {
	return PointValue{
		Timestamp:     timestamp,
		CommonAddress: uint16(packet.CommonAddr),
		IOA:           ioa,
		Type:          typ,
		Cause:         packet.Coa.String(),
		Value:         value,
		Quality:       qualityFromAny(quality),
		RawTypeID:     uint8(packet.Type),
	}
}

func qualityFromAny(value any) Quality {
	qds, ok := value.(asdu.QualityDescriptor)
	if !ok {
		return Quality{}
	}
	return Quality{
		Invalid:     qds&asdu.QDSInvalid != 0,
		NotTopical:  qds&asdu.QDSNotTopical != 0,
		Substituted: qds&asdu.QDSSubstituted != 0,
		Blocked:     qds&asdu.QDSBlocked != 0,
	}
}

func qualityFromBinaryCounter(value asdu.BinaryCounterReading) Quality {
	return Quality{
		Invalid:     value.IsInvalid,
		Substituted: value.IsAdjusted,
	}
}

func runWithContext(ctx context.Context, fn func() error, mapErr func(error) error) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			return mapErr(err)
		}
		return nil
	}
}

func mapWendyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, wendyclient.NotConnected) {
		return fmt.Errorf("%w: %v", ErrSession, err)
	}
	return fmt.Errorf("%w: %v", ErrTCPConnection, err)
}
