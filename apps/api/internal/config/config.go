package config

import "os"

type LLMConfig struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	SupabaseURL string
	SupabaseKey string
	LLM         LLMConfig
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/riffpad?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		SupabaseURL: getEnv("SUPABASE_URL", ""),
		SupabaseKey: getEnv("SUPABASE_ANON_KEY", ""),
		LLM: LLMConfig{
			Provider: getEnv("LLM_PROVIDER", "volcengine"),
			BaseURL:  getEnv("LLM_BASE_URL", "https://ark.cn-beijing.volces.com/api/v3"),
			APIKey:   getEnv("LLM_API_KEY", ""),
			Model:    getEnv("LLM_MODEL", "doubao-pro-32k"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
