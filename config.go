package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL     string
	Port            string
	BaseURL         string
	AuthToken       string
	CleanupInterval time.Duration
	CleanupMaxAge   time.Duration
}

func loadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		Port:            envOr("PORT", "8080"),
		BaseURL:         strings.TrimRight(os.Getenv("BASE_URL"), "/"),
		AuthToken:       os.Getenv("AUTH_TOKEN"),
		CleanupInterval: 0,
		CleanupMaxAge:   0,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.BaseURL == "" {
		return Config{}, fmt.Errorf("BASE_URL is required")
	}
	if cfg.AuthToken == "" {
		return Config{}, fmt.Errorf("AUTH_TOKEN is required")
	}

	var err error
	cfg.CleanupInterval, err = parseDurationEnv("CLEANUP_INTERVAL", time.Hour)
	if err != nil {
		return Config{}, fmt.Errorf("CLEANUP_INTERVAL: %w", err)
	}
	cfg.CleanupMaxAge, err = parseDurationEnv("CLEANUP_MAX_AGE", 24*time.Hour)
	if err != nil {
		return Config{}, fmt.Errorf("CLEANUP_MAX_AGE: %w", err)
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return time.ParseDuration(v)
}
