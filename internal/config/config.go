package config

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig
	App    AppConfig
	Redis  RedisConfig
}

type ServerConfig struct {
	Port string
	Mode string // debug, release, test
}

type AppConfig struct {
	LogLevel string // debug, info, warn, error
}

type RedisConfig struct {
	URL string
}

func Load() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// .env file not found or error, continue with env vars
	}

	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("app.loglevel", "info")
	v.SetDefault("redis.url", "")

	// No need for ReadInConfig since we use env vars

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
