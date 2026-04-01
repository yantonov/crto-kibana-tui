package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFileName = "config.yaml"

// IndexPattern is the OpenSearch index pattern to query.
const IndexPattern = "kestrel-*"

// QueryTimeoutSeconds is the per-datacenter search timeout.
const QueryTimeoutSeconds = 10

// Environments defines available environments and their data centers.
var Environments = map[string]EnvironmentConfig{
	"prod": {DataCenters: []string{"da1", "us5", "fr3", "fr4", "nl3", "jp2", "sg1"}},
	"preprod": {DataCenters: []string{"da1", "fr4"}},
}

// Timeframes defines the selectable time range options.
var Timeframes = []TimeframeOption{
	{Label: "15 minutes", Value: "15m"},
	{Label: "30 minutes", Value: "30m"},
	{Label: "1 hour", Value: "1h"},
	{Label: "3 hours", Value: "3h"},
	{Label: "6 hours", Value: "6h"},
	{Label: "24 hours", Value: "24h"},
	{Label: "2 days", Value: "2d"},
	{Label: "7 days", Value: "7d"},
}

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

	return &cfg, nil
}

// DataCenters returns the list of data centers for the given environment.
func (c *Config) DataCenters(env string) ([]string, error) {
	e, ok := Environments[env]
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

const configTemplate = `applications:
  - my-app
`
