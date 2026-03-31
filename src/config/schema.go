package config

type Config struct {
	Environments map[string]EnvironmentConfig `yaml:"environments"`
	IndexPattern string                       `yaml:"index_pattern"`
	QueryTimeoutSeconds   int                          `yaml:"query_timeout_seconds"`
	Applications []string          `yaml:"applications"`
	Timeframes   []TimeframeOption `yaml:"timeframes"`
}

type EnvironmentConfig struct {
	DataCenters []string `yaml:"data_centers"`
}

type TimeframeOption struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}
