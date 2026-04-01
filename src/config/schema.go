package config

type Config struct {
	Applications []string `yaml:"applications"`
}

type EnvironmentConfig struct {
	DataCenters []string
}

type TimeframeOption struct {
	Label string
	Value string
}
