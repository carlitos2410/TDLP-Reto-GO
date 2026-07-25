package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	name := envOrDefault("WORKER_NAME", "ticker")
	interval := envIntOrDefault("INTERVAL_SECONDS", 2)

	for i := 1; ; i++ {
		fmt.Printf("[%s] tick #%d\n", name, i)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
