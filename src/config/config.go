package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFileName = "config.yaml"

// DefaultConfigPath returns the path to config.yaml next to the executable.
func DefaultConfigPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), configFileName), nil
}

// WriteTemplate writes a starter config.yaml to path if it does not already exist.
func WriteTemplate(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	return os.WriteFile(path, []byte(configTemplate), 0o644)
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if len(cfg.Environments) == 0 {
		return fmt.Errorf("no environments defined")
	}
	if cfg.IndexPattern == "" {
		cfg.IndexPattern = "kestrel-*"
	}
	if cfg.QueryTimeoutSeconds <= 0 {
		cfg.QueryTimeoutSeconds = 10
	}
	return nil
}

// DataCenters returns the list of data centers for the given environment.
func (c *Config) DataCenters(env string) ([]string, error) {
	e, ok := c.Environments[env]
	if !ok {
		return nil, fmt.Errorf("unknown environment %q", env)
	}
	return e.DataCenters, nil
}

// KibanaURL builds the Kibana base URL for a given dc/env pair.
func (c *Config) KibanaURL(dc, env string) string {
	url := "https://kibana.{dc}.{env}.crto.in"
	url = replaceAll(url, "{dc}", dc)
	url = replaceAll(url, "{env}", env)
	return url
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

const configTemplate = `environments:
  prod:
    data_centers:
      - da1
      - us5
      - fr3
      - fr4
      - nl3
      - jp2
      - sg1
  preprod:
    data_centers:
      - da1
      - fr4

index_pattern: "kestrel-*"
query_timeout_seconds: 10

applications:
  - my-app

timeframes:
  - label: "15 minutes"
    value: "15m"
  - label: "30 minutes"
    value: "30m"
  - label: "1 hour"
    value: "1h"
  - label: "3 hours"
    value: "3h"
  - label: "6 hours"
    value: "6h"
  - label: "24 hours"
    value: "24h"
  - label: "2 days"
    value: "2d"
  - label: "7 days"
    value: "7d"
`
