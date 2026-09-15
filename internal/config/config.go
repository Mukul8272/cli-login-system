package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBHost           string
	DBPort           string
	DBName           string
	DBUser           string
	DBPassword       string
	SessionTimeout   time.Duration
	MaxLoginAttempts int
	LockoutDuration  time.Duration
}

func Load() (Config, error) {
	sessionMinutes, err := intEnv("SESSION_TIMEOUT_MINUTES", 30)
	if err != nil {
		return Config{}, err
	}
	maxAttempts, err := intEnv("MAX_LOGIN_ATTEMPTS", 5)
	if err != nil {
		return Config{}, err
	}
	lockoutMinutes, err := intEnv("LOCKOUT_MINUTES", 15)
	if err != nil {
		return Config{}, err
	}
	if sessionMinutes <= 0 || maxAttempts <= 0 || lockoutMinutes <= 0 {
		return Config{}, fmt.Errorf("SESSION_TIMEOUT_MINUTES, MAX_LOGIN_ATTEMPTS and LOCKOUT_MINUTES must be greater than zero")
	}
	return Config{
		DBHost:           os.Getenv("DB_HOST"),
		DBPort:           os.Getenv("DB_PORT"),
		DBName:           os.Getenv("DB_NAME"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		SessionTimeout:   time.Duration(sessionMinutes) * time.Minute,
		MaxLoginAttempts: maxAttempts,
		LockoutDuration:  time.Duration(lockoutMinutes) * time.Minute,
	}, nil
}

func intEnv(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}
