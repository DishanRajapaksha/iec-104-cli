package config

import (
	"fmt"
	"strings"
)

var allowedOutputFormats = map[string]bool{
	"table": true,
	"text":  true,
	"json":  true,
	"jsonl": true,
	"csv":   true,
}

var allowedPointTypes = map[string]bool{
	"single_point":     true,
	"double_point":     true,
	"normalized":       true,
	"scaled":           true,
	"float":            true,
	"integrated_total": true,
}

func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.Connection.Host) == "" {
		return fmt.Errorf("%w: connection.host is required", ErrConfig)
	}
	if cfg.Connection.Port < 1 || cfg.Connection.Port > 65535 {
		return fmt.Errorf("%w: connection.port must be between 1 and 65535", ErrConfig)
	}
	if cfg.Connection.Timeout.Duration() <= 0 {
		return fmt.Errorf("%w: connection.timeout must be positive", ErrConfig)
	}
	if cfg.Connection.Reconnect && cfg.Connection.ReconnectInterval.Duration() <= 0 {
		return fmt.Errorf("%w: connection.reconnect_interval must be positive when reconnect is enabled", ErrConfig)
	}
	if cfg.IEC104.CommonAddress == 0 {
		return fmt.Errorf("%w: iec104.common_address must be between 1 and 65535", ErrConfig)
	}
	if cfg.IEC104.InterrogationQualifier < 20 || cfg.IEC104.InterrogationQualifier > 36 {
		return fmt.Errorf("%w: iec104.interrogation_qualifier must be between 20 and 36", ErrConfig)
	}
	if !allowedOutputFormats[strings.ToLower(strings.TrimSpace(cfg.Output.Format))] {
		return fmt.Errorf("%w: unsupported output format %q", ErrConfig, cfg.Output.Format)
	}
	if cfg.Cache.Enabled && strings.TrimSpace(cfg.Cache.Path) == "" {
		return fmt.Errorf("%w: cache.path is required when cache is enabled", ErrConfig)
	}

	names := map[string]struct{}{}
	for i, point := range cfg.Points {
		if strings.TrimSpace(point.Name) == "" {
			return fmt.Errorf("%w: points[%d].name is required", ErrConfig, i)
		}
		if _, ok := names[point.Name]; ok {
			return fmt.Errorf("%w: duplicate point name %q", ErrConfig, point.Name)
		}
		names[point.Name] = struct{}{}
		if point.IOA == 0 || point.IOA > 16777215 {
			return fmt.Errorf("%w: points[%d].ioa must be between 1 and 16777215", ErrConfig, i)
		}
		if !allowedPointTypes[strings.ToLower(strings.TrimSpace(point.Type))] {
			return fmt.Errorf("%w: unsupported point type %q for point %q", ErrConfig, point.Type, point.Name)
		}
	}

	return nil
}
