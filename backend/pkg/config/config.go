package config

import (
	"os"
	"strings"

	"go.uber.org/fx"
)

// Config holds all application configuration.
type Config struct {
	ServerAddr     string
	DatabaseURL    string
	AllowedOrigins []string
	// NtfyURL is the full ntfy endpoint including topic, e.g.
	// https://ntfy.sh/my-topic or https://myserver.com/my-topic.
	// Leave empty to disable ntfy notifications.
	NtfyURL string
	// NtfyToken is an optional Bearer token for private ntfy topics.
	NtfyToken string
}

// New returns a Config populated from environment variables with sensible defaults.
func New() *Config {
	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")
	return &Config{
		ServerAddr:     getEnv("SERVER_ADDR", "0.0.0.0:8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "ratiodash.db"),
		AllowedOrigins: strings.Split(origins, ","),
		NtfyURL:        getEnv("NTFY_URL", ""),
		NtfyToken:      getEnv("NTFY_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

var Module = fx.Options(
	fx.Provide(New),
)
