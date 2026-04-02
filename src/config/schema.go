package config

type Config struct {
	AppNames     []string                    `yaml:"applications"`
	environments map[string]EnvironmentConfig
}

type EnvironmentConfig struct {
	DataCenters []string
}

type TimeframeOption struct {
	Label string
	Value string
}
