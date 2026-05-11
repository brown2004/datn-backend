package config

import "os"

// Config stores application configuration.
type Config struct {
	AppPort     string
	DatabaseURL string
	AppEnv      string
}

func Load() Config {
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/datn?sslmode=disable"
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	return Config{
		AppPort:     appPort,
		DatabaseURL: databaseURL,
		AppEnv:      appEnv,
	}
}
