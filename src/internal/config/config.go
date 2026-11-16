package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTP  HTTPConfig
	Redis RedisConfig
}

type HTTPConfig struct {
	Address      string        `env:"HTTP_ADDRESS,notEmpty" envDefault:":3000"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT,notEmpty" envDefault:"5s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT,notEmpty" envDefault:"5s"`
}

type RedisConfig struct {
	Address  string `env:"REDIS_ADDRESS,notEmpty" envDefault:"localhost:6379"`
	Username string `env:"REDIS_USERNAME"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB,notEmpty" envDefault:"0"`
}

func Load() (Config, error) {

	if err := godotenv.Load(); err != nil {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	if cfg.HTTP.ReadTimeout <= 0 {
		return Config{}, fmt.Errorf("HTTP_READ_TIMEOUT must be positive")
	}
	if cfg.HTTP.WriteTimeout <= 0 {
		return Config{}, fmt.Errorf("HTTP_WRITE_TIMEOUT must be positive")
	}
	if cfg.Redis.DB < 0 {
		return Config{}, fmt.Errorf("REDIS_DB must be nonnegative")
	}

	return cfg, nil
}
