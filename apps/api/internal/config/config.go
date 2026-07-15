package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	SupabaseURL string
	SupabaseKey string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/riffpad?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		SupabaseURL: getEnv("SUPABASE_URL", ""),
		SupabaseKey: getEnv("SUPABASE_ANON_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
