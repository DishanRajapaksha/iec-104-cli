package config

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	PointFiles []string         `yaml:"point_files"`
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
	if err := loadPointFiles(&cfg, filepath.Dir(path)); err != nil {
		return nil, true, err
	}

	applyDefaults(&cfg)
	applyOverrides(&cfg, overrides)
	return &cfg, true, nil
}

func loadPointFiles(cfg *Config, baseDir string) error {
	for _, pointFile := range cfg.PointFiles {
		path := strings.TrimSpace(pointFile)
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		points, err := loadPointFile(path)
		if err != nil {
			return err
		}
		cfg.Points = append(cfg.Points, points...)
	}
	return nil
}

func loadPointFile(path string) ([]PointConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read point file %q: %w", path, err)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		return parsePointCSV(path, data)
	case ".yaml", ".yml":
		return parsePointYAML(path, data)
	default:
		return nil, fmt.Errorf("unsupported point file %q: expected .csv, .yaml, or .yml", path)
	}
}

func parsePointYAML(path string, data []byte) ([]PointConfig, error) {
	var list []PointConfig
	if err := yaml.Unmarshal(data, &list); err == nil && list != nil {
		return list, nil
	}
	var doc struct {
		Points []PointConfig `yaml:"points"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse point file %q: %w", path, err)
	}
	return doc.Points, nil
}

func parsePointCSV(path string, data []byte) ([]PointConfig, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse point file %q: %w", path, err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	header := map[string]int{}
	for i, name := range records[0] {
		header[strings.ToLower(strings.TrimSpace(name))] = i
	}
	required := []string{"name", "ioa", "type"}
	for _, name := range required {
		if _, ok := header[name]; !ok {
			return nil, fmt.Errorf("failed to parse point file %q: missing %q column", path, name)
		}
	}
	points := make([]PointConfig, 0, len(records)-1)
	for row, record := range records[1:] {
		ioa, err := strconv.ParseUint(csvField(record, header["ioa"]), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse point file %q row %d: invalid ioa: %w", path, row+2, err)
		}
		points = append(points, PointConfig{
			Name: csvField(record, header["name"]),
			IOA:  uint32(ioa),
			Type: csvField(record, header["type"]),
			Unit: csvOptionalField(record, header, "unit"),
		})
	}
	return points, nil
}

func csvField(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func csvOptionalField(record []string, header map[string]int, name string) string {
	index, ok := header[name]
	if !ok {
		return ""
	}
	return csvField(record, index)
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
