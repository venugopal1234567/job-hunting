package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Nvidia   NvidiaConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type NvidiaConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "remotehunter"),
			User:     getEnv("DB_USER", "hunter"),
			Password: getEnv("DB_PASSWORD", "hunterpass"),
		},
		Nvidia: NvidiaConfig{
			APIKey:  getEnv("NVIDIA_API_KEY", ""),
			BaseURL: getEnv("NVIDIA_BASE_URL", "https://integrate.api.nvidia.com/v1"),
			Model:   getEnv("NVIDIA_MODEL", "z-ai/glm-5.2"),
		},
	}
}

// DSN returns the PostgreSQL data source name
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		d.Host, d.Port, d.User, d.Password, d.Name,
	)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
