package config

type Config struct {
	Environments map[string]EnvironmentConfig `yaml:"environments"`
	IndexPattern string                       `yaml:"index_pattern"`
	QueryTimeoutSeconds   int                          `yaml:"query_timeout_seconds"`
	Applications          []string                     `yaml:"applications"`
	SeverityLevels        []string                     `yaml:"severity_levels"`
	Timeframes            []TimeframeOption            `yaml:"timeframes"`
	FieldMapping          FieldMapping                 `yaml:"field_mapping"`
}

type EnvironmentConfig struct {
	DataCenters []string `yaml:"data_centers"`
}

type TimeframeOption struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}

type FieldMapping struct {
	Timestamp   string `yaml:"timestamp"`
	Severity    string `yaml:"severity"`
	Application string `yaml:"application"`
	TraceID     string `yaml:"trace_id"`
	Message     string `yaml:"message"`
}
