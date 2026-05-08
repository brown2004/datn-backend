package config

// Config stores application configuration.
type Config struct {
	AppPort     string
	DatabaseURL string
}

// Load returns placeholder configuration with defaults.
func Load() Config {
	return Config{
		AppPort:     "8080",
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/datn?sslmode=disable",
	}
}
