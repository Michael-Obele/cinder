package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: error loading .env file: %v", err)
	}

	redisURL := os.Getenv("REDIS_URL")
	fmt.Printf("Testing connection to: %s\n", redisURL)

	opts, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	inspector := asynq.NewInspector(opts)
	defer inspector.Close()

	// Try to get queues - this requires a connection
	queues, err := inspector.Queues()
	if err != nil {
		log.Fatalf("FAIL: Could not connect to Redis: %v", err)
	}

	fmt.Printf("SUCCESS: Connected to Redis! Found queues: %v\n", queues)
}
