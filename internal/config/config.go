// Package config loads configuration from the environment using only the
// standard library.
package config

import "os"

// Config holds the environment-derived configuration.
type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
}

// Load reads configuration from environment variables, with PORT defaulting
// to 8080.
func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        port,
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}, nil
}
