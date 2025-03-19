package main

import (
	"os"
	"strconv"
)

func getEnvInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultVal
}
