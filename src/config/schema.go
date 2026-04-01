package config

type Config struct {
	AppNames []string `yaml:"applications"`
}

type EnvironmentConfig struct {
	DataCenters []string
}

type TimeframeOption struct {
	Label string
	Value string
}
