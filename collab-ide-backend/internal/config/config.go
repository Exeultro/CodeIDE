package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	RedisHost      string
	RedisPwd       string
	JWTSecret      string
	OllamaURL      string
	OllamaModel    string
	AllowedOrigins []string
	TelegramToken  string `json:"telegram_token"`
	TelegramChatID string `json:"telegram_chat_id"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	return &Config{
		Port:           getEnv("PORT", "8080"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "root"),
		DBName:         getEnv("DB_NAME", "collabide"),
		RedisHost:      getEnv("REDIS_HOST", "localhost:6379"),
		RedisPwd:       getEnv("REDIS_PASSWORD", ""),
		JWTSecret:      getEnv("JWT_SECRET", "change-me"),
		OllamaURL:      getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel:    strings.TrimSpace(getEnv("OLLAMA_MODEL", "gemma3:1b")),
		AllowedOrigins: splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:8080,http://127.0.0.1:8080")),
		TelegramToken:  getEnv("TELEGRAM_TOKEN", ""),
		TelegramChatID: getEnv("TELEGRAM_CHAT_ID", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
