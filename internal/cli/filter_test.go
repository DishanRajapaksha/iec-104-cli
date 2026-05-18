package cli

import (
	"testing"

	"github.com/DishanRajapaksha/iec-104-cli/internal/config"
	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
)

func TestBuildPointFilterEnrichesConfiguredPoint(t *testing.T) {
	cfg := config.Default()
	cfg.Points = []config.PointConfig{{Name: "active_power", IOA: 1001, Type: "float", Unit: "MW"}}
	filter, err := buildPointFilter(cfg, 1, 0, "active_power")
	if err != nil {
		t.Fatalf("buildPointFilter returned error: %v", err)
	}

	value, ok := filter(iec104.PointValue{CommonAddress: 1, IOA: 1001, Value: 12.34})
	if !ok {
		t.Fatal("filter rejected matching point")
	}
	if value.Name != "active_power" || value.Unit != "MW" || value.Type != "float" {
		t.Fatalf("enriched value = %#v", value)
	}
}

func TestBuildPointFilterRejectsUnknownPoint(t *testing.T) {
	_, err := buildPointFilter(config.Default(), 0, 0, "missing")
	if err == nil {
		t.Fatal("expected unknown point error")
	}
}

func TestPointFilterRejectsNonMatchingCommonAddress(t *testing.T) {
	filter, err := buildPointFilter(config.Default(), 2, 0, "")
	if err != nil {
		t.Fatalf("buildPointFilter returned error: %v", err)
	}
	if _, ok := filter(iec104.PointValue{CommonAddress: 1, IOA: 1001}); ok {
		t.Fatal("filter accepted non-matching common address")
	}
}
