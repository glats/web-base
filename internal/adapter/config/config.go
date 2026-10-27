package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTP     HTTPConfig     `envPrefix:"HTTP_"`
	Database DatabaseConfig `envPrefix:"DATABASE_"`
	Log      LogConfig      `envPrefix:"LOG_"`
	Seed     SeedConfig     `envPrefix:"DB_SEED_"`
}

type HTTPConfig struct {
	Addr string `env:"ADDR" envDefault:":8080"`
}

type DatabaseConfig struct {
	Path string `env:"PATH" envDefault:"./data/app.db"`
}

type LogConfig struct {
	Level  string `env:"LEVEL" envDefault:"info"`
	Pretty bool   `env:"PRETTY" envDefault:"false"`
}

type SeedConfig struct {
	Enabled bool `env:"ENABLED" envDefault:"false"`
}

func Parse() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}

	return cfg, nil
}

func Get() Config {
	cfg, err := Parse()
	if err != nil {
		panic(err)
	}

	return cfg
}
