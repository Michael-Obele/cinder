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

	// ChromeRecycleAfter restarts the Chrome allocator after this many
	// scrapes to bound browser memory growth (0 = default 100).
	ChromeRecycleAfter int `mapstructure:"chrome_recycle_after"`

	// APIKeys, when non-empty, enables X-API-Key auth on /v1/*.
	APIKeys []string `mapstructure:"api_keys"`

	// RateLimitRPM caps requests per client per minute (0 = unlimited).
	RateLimitRPM int `mapstructure:"rate_limit_rpm"`
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`

	// Upstash REST API credentials.
	// When set (and REDIS_URL is empty), the standard Redis URL is
	// derived automatically as:
	//   rediss://default:<RestToken>@<host-from-RestURL>:6379
	RestURL   string `mapstructure:"rest_url"`
	RestToken string `mapstructure:"rest_token"`
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
	v.SetDefault("app.chrome_recycle_after", 100)
	v.SetDefault("app.api_keys", "")
	v.SetDefault("app.rate_limit_rpm", 0)
	v.SetDefault("redis.url", "")
	v.SetDefault("redis.host", "")
	v.SetDefault("redis.port", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.rest_url", "")
	v.SetDefault("redis.rest_token", "")
	v.SetDefault("brave.api_key", "")

	// Custom bindings
	v.BindEnv("brave.api_key", "BRAVE_SEARCH_API_KEY")
	v.BindEnv("redis.rest_url", "UPSTASH_REDIS_REST_URL")
	v.BindEnv("redis.rest_token", "UPSTASH_REDIS_REST_TOKEN")

	// No need for ReadInConfig since we use env vars

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Parse comma-separated API keys from the environment.
	if raw := v.GetString("app.api_keys"); raw != "" {
		cfg.App.APIKeys = nil
		for _, k := range strings.Split(raw, ",") {
			if k = strings.TrimSpace(k); k != "" {
				cfg.App.APIKeys = append(cfg.App.APIKeys, k)
			}
		}
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

	// Derive Redis URL from Upstash REST credentials (standard env vars
	// set by Upstash console).  Only used when nothing else is set.
	if cfg.Redis.URL == "" && cfg.Redis.RestURL != "" && cfg.Redis.RestToken != "" {
		// UPSTASH_REDIS_REST_URL = https://<name>.upstash.io
		host := strings.TrimPrefix(cfg.Redis.RestURL, "https://")
		host = strings.TrimPrefix(host, "http://")
		// Strip trailing slash if present
		host = strings.TrimRight(host, "/")
		// Upstash Redis uses the same hostname and token, port 6379 with TLS
		cfg.Redis.URL = fmt.Sprintf("rediss://default:%s@%s:6379", cfg.Redis.RestToken, host)
	}

	return &cfg, nil
}
