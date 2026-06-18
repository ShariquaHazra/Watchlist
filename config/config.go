package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	ServerPort  string
	JWTSecret   string
	CSVURL      string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		ServerPort:  getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "secret"),
		CSVURL:      getEnv("CSV_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}