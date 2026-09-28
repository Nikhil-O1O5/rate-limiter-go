package config

import "os"

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	RedisAddr  string
	AppPort    string
}

func Load() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "rl_user"),
		DBPassword: getEnv("DB_PASSWORD", "rl_pass"),
		DBName:     getEnv("DB_NAME", "rl_db"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		AppPort:    getEnv("APP_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
