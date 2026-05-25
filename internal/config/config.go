package config

import (
	"os"
	"time"
)

// Config stores application configuration.
type Config struct {
	AppPort     string
	DatabaseURL string
	AppEnv      string
	// jwt options
	JWTSecret       string
	JWTIssuer       string
	JWTAudience     string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func Load() Config {
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5433/datn?sslmode=disable"
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-only-change-this-jwt-secret"
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "datn-backend"
	}

	jwtAudience := os.Getenv("JWT_AUDIENCE")
	if jwtAudience == "" {
		jwtAudience = "datn-api"
	}

	return Config{
		AppPort:         appPort,
		DatabaseURL:     databaseURL,
		AppEnv:          appEnv,
		JWTSecret:       jwtSecret,
		JWTIssuer:       jwtIssuer,
		JWTAudience:     jwtAudience,
		AccessTokenTTL:  durationFromEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: durationFromEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour),
	}
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}
