package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "config.yaml"

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
	if cfg.OpenSearchURLTemplate == "" {
		return fmt.Errorf("opensearch_url_template is required")
	}
	if cfg.IndexPattern == "" {
		return fmt.Errorf("index_pattern is required")
	}
	if cfg.FieldMapping.Timestamp == "" {
		return fmt.Errorf("field_mapping.timestamp is required")
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

// OpenSearchURL builds the OpenSearch base URL for a given dc/env pair.
func (c *Config) OpenSearchURL(dc, env string) string {
	url := c.OpenSearchURLTemplate
	url = replaceAll(url, "{dc}", dc)
	url = replaceAll(url, "{env}", env)
	return url
}

// KibanaURL builds the Kibana base URL for a given dc/env pair.
func (c *Config) KibanaURL(dc, env string) string {
	url := c.KibanaURLTemplate
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
