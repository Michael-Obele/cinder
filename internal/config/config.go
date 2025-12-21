package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	App    AppConfig    `mapstructure:"app"`
	Redis  RedisConfig  `mapstructure:"redis"`
	Brave  BraveConfig  `mapstructure:"brave"`
}

type BraveConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release, test
}

type AppConfig struct {
	LogLevel string `mapstructure:"loglevel"` // debug, info, warn, error
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
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
	v.SetDefault("redis.host", "")
	v.SetDefault("redis.port", "")
	v.SetDefault("redis.port", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("brave.api_key", "")

	// Custom bindings
	v.BindEnv("brave.api_key", "BRAVE_SEARCH_API_KEY")

	// No need for ReadInConfig since we use env vars

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Construct Redis URL if not set but individual fields are present
	if cfg.Redis.URL == "" && cfg.Redis.Host != "" {
		port := cfg.Redis.Port
		if port == "" {
			port = "6379"
		}

		addr := fmt.Sprintf("%s:%s", cfg.Redis.Host, port)
		if cfg.Redis.Password != "" {
			cfg.Redis.URL = fmt.Sprintf("redis://:%s@%s", cfg.Redis.Password, addr)
		} else {
			cfg.Redis.URL = fmt.Sprintf("redis://%s", addr)
		}
	}

	return &cfg, nil
}
