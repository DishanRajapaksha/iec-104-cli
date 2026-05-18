package iec104

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/iec104/server"
)

func TestIntegrationClientConnectsToLocalServer(t *testing.T) {
	srv, cfg, _ := startIntegrationServer(t)
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := NewWendyClient(cfg).TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection returned error: %v", err)
	}
}

func TestIntegrationInterrogationReturnsValues(t *testing.T) {
	srv, cfg, _ := startIntegrationServer(t)
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	values, err := NewWendyClient(cfg).Interrogate(ctx, 1)
	if len(values) == 0 {
		t.Fatalf("Interrogate returned no values, err=%v", err)
	}
}

func TestIntegrationListenReceivesSpontaneousValue(t *testing.T) {
	srv, cfg, _ := startIntegrationServer(t)
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got := make(chan PointValue, 1)
	err := NewWendyClient(cfg).Listen(ctx, func(value PointValue) {
		got <- value
		cancel()
	})
	if err != nil && err != context.Canceled {
		t.Fatalf("Listen returned error: %v", err)
	}
	select {
	case <-got:
	default:
		t.Fatal("listen did not receive value")
	}
}

func TestIntegrationSingleCommandReachesServerHandler(t *testing.T) {
	srv, cfg, handler := startIntegrationServer(t)
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := NewWendyClient(cfg).SendSingleCommand(ctx, 1, 1000, true); err != nil {
		t.Fatalf("SendSingleCommand returned error: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for atomic.LoadInt32(&handler.singleCommands) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&handler.singleCommands); got == 0 {
		t.Fatal("server handler did not receive single command")
	}
}

func startIntegrationServer(t *testing.T) (*server.Server, ClientConfig, *integrationHandler) {
	t.Helper()
	port := freeTCPPort(t)
	settings := server.NewSettings()
	settings.Host = "127.0.0.1"
	settings.Port = port
	settings.Params = asdu.ParamsWide
	handler := &integrationHandler{}
	srv := server.New(settings, handler)
	srv.SetOnConnectionHandler(func(conn asdu.Connect) {
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = asdu.MeasuredValueFloat(conn, false, asdu.CauseOfTransmission{Cause: asdu.Spontaneous}, 1, asdu.MeasuredValueFloatInfo{
				Ioa:   1001,
				Value: 12.5,
				Qds:   asdu.QDSGood,
				Time:  time.Now(),
			})
		}()
	})
	srv.Start()
	time.Sleep(100 * time.Millisecond)
	return srv, ClientConfig{
		Host:              "127.0.0.1",
		Port:              port,
		Timeout:           time.Second,
		Reconnect:         false,
		ReconnectInterval: 100 * time.Millisecond,
	}, handler
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

type integrationHandler struct {
	singleCommands int32
}

func (h *integrationHandler) OnInterrogation(conn asdu.Connect, _ *asdu.ASDU, _ asdu.QualifierOfInterrogation) error {
	return asdu.MeasuredValueFloat(conn, false, asdu.CauseOfTransmission{Cause: asdu.InterrogatedByStation}, 1, asdu.MeasuredValueFloatInfo{
		Ioa:   1001,
		Value: 12.5,
		Qds:   asdu.QDSGood,
		Time:  time.Now(),
	})
}

func (h *integrationHandler) OnCounterInterrogation(asdu.Connect, *asdu.ASDU, asdu.QualifierCountCall) error {
	return nil
}

func (h *integrationHandler) OnRead(conn asdu.Connect, _ *asdu.ASDU, ioa asdu.InfoObjAddr) error {
	return asdu.MeasuredValueFloat(conn, false, asdu.CauseOfTransmission{Cause: asdu.Request}, 1, asdu.MeasuredValueFloatInfo{
		Ioa:   ioa,
		Value: 12.5,
		Qds:   asdu.QDSGood,
		Time:  time.Now(),
	})
}

func (h *integrationHandler) OnClockSync(asdu.Connect, *asdu.ASDU, time.Time) error {
	return nil
}

func (h *integrationHandler) OnResetProcess(asdu.Connect, *asdu.ASDU, asdu.QualifierOfResetProcessCmd) error {
	return nil
}

func (h *integrationHandler) OnDelayAcquisition(asdu.Connect, *asdu.ASDU, uint16) error {
	return nil
}

func (h *integrationHandler) OnTestCommand(asdu.Connect, *asdu.ASDU) error {
	return nil
}

func (h *integrationHandler) OnASDU(_ asdu.Connect, packet *asdu.ASDU) error {
	if packet.Type == asdu.C_SC_NA_1 {
		atomic.AddInt32(&h.singleCommands, 1)
	}
	return nil
}
