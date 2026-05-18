package config

import "testing"

func TestValidateExampleShape(t *testing.T) {
	cfg := Default()
	cfg.Connection.Host = "127.0.0.1"
	cfg.Points = []PointConfig{{
		Name: "active_power",
		IOA:  1001,
		Type: "float",
		Unit: "MW",
	}}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRejectsDuplicatePointName(t *testing.T) {
	cfg := Default()
	cfg.Connection.Host = "127.0.0.1"
	cfg.Points = []PointConfig{
		{Name: "active_power", IOA: 1001, Type: "float"},
		{Name: "active_power", IOA: 1002, Type: "float"},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected duplicate point name error")
	}
}

func TestValidateRejectsUnsupportedOutputFormat(t *testing.T) {
	cfg := Default()
	cfg.Connection.Host = "127.0.0.1"
	cfg.Output.Format = "yaml"

	if err := Validate(cfg); err == nil {
		t.Fatal("expected output format error")
	}
}
