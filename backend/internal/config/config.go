// Package config loads application settings from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	AllowedOrigins []string
	MaxUploadMB    int64
	AskPerMinute   int // per-IP rate limit for POST /api/ask

	DatabaseURL string

	GeminiAPIKey string
	GeminiModel  string
	// GeminiFallbackModels are tried in order when GeminiModel is overloaded/unavailable.
	GeminiFallbackModels []string

	// StorageProvider: "cloudinary" or "supabase"
	StorageProvider string

	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	CloudinaryFolder    string

	SupabaseURL        string
	SupabaseServiceKey string
	SupabaseBucket     string
}

func Load() (*Config, error) {
	c := &Config{
		Port:           env("PORT", "8080"),
		AllowedOrigins: splitList(env("ALLOWED_ORIGINS", "http://localhost:5173")),
		MaxUploadMB:    int64(envInt("MAX_UPLOAD_MB", 5)),
		AskPerMinute:   envInt("ASK_PER_MINUTE", 5),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          env("GEMINI_MODEL", "gemini-3.8-flash"),
		GeminiFallbackModels: splitList(env("GEMINI_FALLBACK_MODELS", "gemini-3.7-flash,gemini-3.5-flash,gemini-flash-latest")),

		StorageProvider: env("STORAGE_PROVIDER", "cloudinary"),

		CloudinaryCloudName: os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey:    os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret: os.Getenv("CLOUDINARY_API_SECRET"),
		CloudinaryFolder:    env("CLOUDINARY_FOLDER", "plantdoc"),

		SupabaseURL:        strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		SupabaseServiceKey: os.Getenv("SUPABASE_SERVICE_KEY"),
		SupabaseBucket:     env("SUPABASE_BUCKET", "plantdoc"),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func splitList(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
