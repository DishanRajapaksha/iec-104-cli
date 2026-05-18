package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath = "config.yaml"
	DefaultPort       = 2404
	DefaultFormat     = "table"
)

// Duration wraps time.Duration so YAML config can use values such as "10s" or "1m".
type Duration time.Duration

func NewDuration(d time.Duration) Duration {
	return Duration(d)
}

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == 0 || value.Value == "" {
		*d = 0
		return nil
	}

	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", value.Value, err)
	}

	*d = Duration(parsed)
	return nil
}

type Config struct {
	Connection ConnectionConfig    `yaml:"connection"`
	IEC104     IEC104Config        `yaml:"iec104"`
	Output     OutputConfig        `yaml:"output"`
	Points     []PointConfig       `yaml:"points"`
	Profiles   map[string]Profile  `yaml:"profiles,omitempty"`
}

type Profile struct {
	Connection *ConnectionConfig `yaml:"connection,omitempty"`
	IEC104     *IEC104Config     `yaml:"iec104,omitempty"`
	Output     *OutputConfig     `yaml:"output,omitempty"`
	Points     []PointConfig     `yaml:"points,omitempty"`
}

type ConnectionConfig struct {
	Host              string   `yaml:"host"`
	Port              int      `yaml:"port"`
	Timeout           Duration `yaml:"timeout"`
	Reconnect         bool     `yaml:"reconnect"`
	ReconnectInterval Duration `yaml:"reconnect_interval"`
}

type IEC104Config struct {
	CommonAddress         uint16 `yaml:"common_address"`
	OriginatorAddress     uint8  `yaml:"originator_address"`
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
	Unit string `yaml:"unit,omitempty"`
}

type Overrides struct {
	Host       string
	Port       int
	Timeout    time.Duration
	Format     string
	Profile    string
	ConfigPath string
}

func Defaults() Config {
	return Config{
		Connection: ConnectionConfig{
			Port:              DefaultPort,
			Timeout:           NewDuration(10 * time.Second),
			Reconnect:         true,
			ReconnectInterval: NewDuration(5 * time.Second),
		},
		IEC104: IEC104Config{
			CommonAddress:         1,
			OriginatorAddress:     0,
			InterrogationQualifier: 20,
		},
		Output: OutputConfig{
			Format:     DefaultFormat,
			Timestamps: "local",
		},
	}
}

func LoadFile(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %q: %w", path, err)
	}

	applyDefaults(&cfg)
	return cfg, nil
}

func LoadOptional(path string) (Config, bool, error) {
	cfg, err := LoadFile(path)
	if err == nil {
		return cfg, true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return Defaults(), false, nil
	}

	return cfg, false, err
}

func ApplyProfile(cfg Config, name string) (Config, error) {
	if name == "" {
		return cfg, nil
	}

	profile, ok := cfg.Profiles[name]
	if !ok {
		return cfg, fmt.Errorf("profile %q not found", name)
	}

	if profile.Connection != nil {
		cfg.Connection = mergeConnection(cfg.Connection, *profile.Connection)
	}
	if profile.IEC104 != nil {
		cfg.IEC104 = mergeIEC104(cfg.IEC104, *profile.IEC104)
	}
	if profile.Output != nil {
		cfg.Output = mergeOutput(cfg.Output, *profile.Output)
	}
	if len(profile.Points) > 0 {
		cfg.Points = profile.Points
	}

	applyDefaults(&cfg)
	return cfg, nil
}

func ApplyOverrides(cfg Config, overrides Overrides) Config {
	if overrides.Host != "" {
		cfg.Connection.Host = overrides.Host
	}
	if overrides.Port != 0 {
		cfg.Connection.Port = overrides.Port
	}
	if overrides.Timeout != 0 {
		cfg.Connection.Timeout = NewDuration(overrides.Timeout)
	}
	if overrides.Format != "" {
		cfg.Output.Format = overrides.Format
	}

	applyDefaults(&cfg)
	return cfg
}

func Load(path string, overrides Overrides) (Config, bool, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	cfg, loaded, err := LoadOptional(path)
	if err != nil {
		return cfg, loaded, err
	}

	cfg, err = ApplyProfile(cfg, overrides.Profile)
	if err != nil {
		return cfg, loaded, err
	}

	cfg = ApplyOverrides(cfg, overrides)
	return cfg, loaded, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Connection.Port == 0 {
		cfg.Connection.Port = DefaultPort
	}
	if cfg.Connection.Timeout == 0 {
		cfg.Connection.Timeout = NewDuration(10 * time.Second)
	}
	if cfg.Connection.ReconnectInterval == 0 {
		cfg.Connection.ReconnectInterval = NewDuration(5 * time.Second)
	}
	if cfg.IEC104.CommonAddress == 0 {
		cfg.IEC104.CommonAddress = 1
	}
	if cfg.IEC104.InterrogationQualifier == 0 {
		cfg.IEC104.InterrogationQualifier = 20
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = DefaultFormat
	}
	if cfg.Output.Timestamps == "" {
		cfg.Output.Timestamps = "local"
	}
}

func mergeConnection(base, override ConnectionConfig) ConnectionConfig {
	if override.Host != "" {
		base.Host = override.Host
	}
	if override.Port != 0 {
		base.Port = override.Port
	}
	if override.Timeout != 0 {
		base.Timeout = override.Timeout
	}
	if override.Reconnect {
		base.Reconnect = true
	}
	if override.ReconnectInterval != 0 {
		base.ReconnectInterval = override.ReconnectInterval
	}
	return base
}

func mergeIEC104(base, override IEC104Config) IEC104Config {
	if override.CommonAddress != 0 {
		base.CommonAddress = override.CommonAddress
	}
	if override.OriginatorAddress != 0 {
		base.OriginatorAddress = override.OriginatorAddress
	}
	if override.InterrogationQualifier != 0 {
		base.InterrogationQualifier = override.InterrogationQualifier
	}
	return base
}

func mergeOutput(base, override OutputConfig) OutputConfig {
	if override.Format != "" {
		base.Format = override.Format
	}
	if override.Timestamps != "" {
		base.Timestamps = override.Timestamps
	}
	return base
}
