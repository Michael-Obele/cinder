package main

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Redis RedisConfig
}

type RedisConfig struct {
	URL string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("godotenv error: %v\n", err)
	}

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetDefault("redis.url", "")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		fmt.Printf("v.Unmarshal error: %v\n", err)
	}

	fmt.Printf("Redis URL from Viper v.Get: %q\n", v.Get("redis.url"))
	fmt.Printf("Redis URL from Unmarshal: %q\n", cfg.Redis.URL)
}
