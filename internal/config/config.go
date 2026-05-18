package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultPath   = "config.yaml"
	DefaultFormat = "table"
)

type Duration struct {
	d time.Duration
}

func NewDuration(d time.Duration) Duration {
	return Duration{d: d}
}

func (d Duration) Duration() time.Duration {
	return d.d
}

func (d Duration) IsZero() bool {
	return d.d == 0
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.d = parsed
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return d.d.String(), nil
}

type Config struct {
	Connection ConnectionConfig `yaml:"connection"`
	IEC104     IEC104Config     `yaml:"iec104"`
	Output     OutputConfig     `yaml:"output"`
	Points     []PointConfig    `yaml:"points"`
}

type ConnectionConfig struct {
	Host              string   `yaml:"host"`
	Port              int      `yaml:"port"`
	Timeout           Duration `yaml:"timeout"`
	Reconnect         bool     `yaml:"reconnect"`
	ReconnectInterval Duration `yaml:"reconnect_interval"`
}

type IEC104Config struct {
	CommonAddress          uint16 `yaml:"common_address"`
	OriginatorAddress      uint8  `yaml:"originator_address"`
	InterrogationQualifier uint8  `yaml:"interrogation_qualifier"`
}

type OutputConfig struct {
	Format     string `yaml:"format"`
	Timestamps string `yaml:"timestamps"`
}

type PointConfig struct {
	Name string `yaml:"name"`
	IOA  uint32 `yaml:"ioa"`
	Type string `yaml:"type"`
	Unit string `yaml:"unit"`
}

type Overrides struct {
	Host          *string
	Port          *int
	Timeout       *time.Duration
	OutputFormat  *string
	CommonAddress *uint16
}

func Default() Config {
	return Config{
		Connection: ConnectionConfig{
			Port:              2404,
			Timeout:           NewDuration(10 * time.Second),
			Reconnect:         true,
			ReconnectInterval: NewDuration(5 * time.Second),
		},
		IEC104: IEC104Config{
			CommonAddress:          1,
			OriginatorAddress:      0,
			InterrogationQualifier: 20,
		},
		Output: OutputConfig{
			Format:     DefaultFormat,
			Timestamps: "local",
		},
	}
}

func Load(path string, overrides Overrides) (*Config, bool, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, false, err
		}
		applyOverrides(&cfg, overrides)
		return &cfg, false, nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, true, fmt.Errorf("failed to parse config: %w", err)
	}

	applyDefaults(&cfg)
	applyOverrides(&cfg, overrides)
	return &cfg, true, nil
}

func LoadRequired(path string, overrides Overrides) (*Config, error) {
	cfg, found, err := Load(path, overrides)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("config file %q not found", path)
	}
	return cfg, nil
}

func ParsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", value, err)
	}
	return port, nil
}

func applyDefaults(cfg *Config) {
	defaults := Default()
	if cfg.Connection.Port == 0 {
		cfg.Connection.Port = defaults.Connection.Port
	}
	if cfg.Connection.Timeout.IsZero() {
		cfg.Connection.Timeout = defaults.Connection.Timeout
	}
	if cfg.Connection.ReconnectInterval.IsZero() {
		cfg.Connection.ReconnectInterval = defaults.Connection.ReconnectInterval
	}
	if cfg.IEC104.CommonAddress == 0 {
		cfg.IEC104.CommonAddress = defaults.IEC104.CommonAddress
	}
	if cfg.IEC104.InterrogationQualifier == 0 {
		cfg.IEC104.InterrogationQualifier = defaults.IEC104.InterrogationQualifier
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = defaults.Output.Format
	}
	if cfg.Output.Timestamps == "" {
		cfg.Output.Timestamps = defaults.Output.Timestamps
	}
}

func applyOverrides(cfg *Config, overrides Overrides) {
	if overrides.Host != nil {
		cfg.Connection.Host = *overrides.Host
	}
	if overrides.Port != nil {
		cfg.Connection.Port = *overrides.Port
	}
	if overrides.Timeout != nil {
		cfg.Connection.Timeout = NewDuration(*overrides.Timeout)
	}
	if overrides.OutputFormat != nil {
		cfg.Output.Format = *overrides.OutputFormat
	}
	if overrides.CommonAddress != nil {
		cfg.IEC104.CommonAddress = *overrides.CommonAddress
	}
}
