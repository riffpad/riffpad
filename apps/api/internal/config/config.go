package config

import (
	"log"
	"os"
	"strconv"
)

type LLMConfig struct {
	Provider       string
	BaseURL        string
	APIKey         string
	Model          string
	EnableThinking *bool
	ThinkingBudget *int
	ContextWindow  int
	ReserveTokens  int
	KeepRecentTokens int
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
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/riffpad?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		SupabaseURL: getEnv("SUPABASE_URL", ""),
		SupabaseKey: getEnv("SUPABASE_ANON_KEY", ""),
		LLM: LLMConfig{
			Provider:              getEnv("LLM_PROVIDER", ""), // optional, informational only
			BaseURL:               getEnv("LLM_BASE_URL", "https://ark.cn-beijing.volces.com/api/v3"),
			APIKey:                getEnv("LLM_API_KEY", ""),
			Model:                 getEnv("LLM_MODEL", "doubao-pro-32k"),
			ContextWindow:         128000,
			ReserveTokens:    16384,
			KeepRecentTokens: 20480,
		},
	}
	if v := os.Getenv("LLM_ENABLE_THINKING"); v != "" {
		b := v == "true" || v == "1"
		cfg.LLM.EnableThinking = &b
	}
	if v := os.Getenv("LLM_THINKING_BUDGET"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LLM.ThinkingBudget = &n
		}
	}
	if v := os.Getenv("LLM_CONTEXT_WINDOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LLM.ContextWindow = n
		}
	}
	if v := os.Getenv("LLM_RESERVE_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LLM.ReserveTokens = n
		}
	}
	if v := os.Getenv("LLM_KEEP_RECENT_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LLM.KeepRecentTokens = n
		}
	}
	log.Printf("config loaded: baseURL=%s model=%s apiKeyLen=%d", cfg.LLM.BaseURL, cfg.LLM.Model, len(cfg.LLM.APIKey))
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
