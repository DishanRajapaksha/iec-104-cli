package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration to provide custom parsing for YAML.
type Duration struct {
	d time.Duration
}

// Duration returns the underlying time.Duration.
func (d Duration) Duration() time.Duration {
	return d.d
}

// UnmarshalYAML parses duration strings like "10s", "1m", etc.
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

// Config represents the IEC 104 connection configuration.
type Config struct {
	Endpoint     string   `yaml:"endpoint"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	CertFile     string   `yaml:"cert_file"`
	KeyFile      string   `yaml:"key_file"`
	CAFile       string   `yaml:"ca_file"`
	Timeout      Duration `yaml:"timeout"`
}

// Load reads a YAML configuration file and applies CLI-style overrides.
// If the file doesn't exist, it returns a config with defaults.
func Load(path string, overrides map[string]string) (*Config, error) {
	cfg := &Config{
		Timeout: Duration{d: 30 * time.Second}, // default timeout
	}

	// Try to read the file
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// File doesn't exist, use defaults
	} else {
		// Parse YAML
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Apply overrides
	if overrides != nil {
		if endpoint, ok := overrides["endpoint"]; ok {
			cfg.Endpoint = endpoint
		}
		if username, ok := overrides["username"]; ok {
			cfg.Username = username
		}
		if password, ok := overrides["password"]; ok {
			cfg.Password = password
		}
		if certFile, ok := overrides["cert_file"]; ok {
			cfg.CertFile = certFile
		}
		if keyFile, ok := overrides["key_file"]; ok {
			cfg.KeyFile = keyFile
		}
		if caFile, ok := overrides["ca_file"]; ok {
			cfg.CAFile = caFile
		}
		if timeout, ok := overrides["timeout"]; ok {
			duration, err := time.ParseDuration(timeout)
			if err != nil {
				return nil, fmt.Errorf("invalid timeout override %q: %w", timeout, err)
			}
			cfg.Timeout = Duration{d: duration}
		}
	}

	return cfg, nil
}
