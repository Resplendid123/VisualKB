package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("configs path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve configs path failed: %w", err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read configs file failed: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml failed: %w", err)
	}
	envPath := filepath.Join(filepath.Dir(abs), "..", ".env")
	if err := loadEnv(&cfg, envPath); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadEnv(cfg *Config, envPath string) error {
	if _, err := os.Stat(envPath); err != nil {
		return fmt.Errorf(".env not found at %s: %w", envPath, err)
	}
	if err := godotenv.Overload(envPath); err != nil {
		return fmt.Errorf("load .env %s failed: %w", envPath, err)
	}
	if v := os.Getenv("KB_DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("KB_REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("KB_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("KB_JWT_ISSUER"); v != "" {
		cfg.Auth.Issuer = v
	}
	if v := os.Getenv("KB_LLM_API_KEY"); v != "" {
		cfg.LLM.ApiKey = v
	}
	if v := os.Getenv("KB_EMBEDDING_API_KEY"); v != "" {
		cfg.Embedding.ApiKey = v
	}
	if v := os.Getenv("KB_RERANK_API_KEY"); v != "" {
		cfg.Rerank.ApiKey = v
	}
	if v := os.Getenv("KB_S3_ACCESS_KEY"); v != "" {
		cfg.S3.AccessKey = v
	}
	if v := os.Getenv("KB_S3_SECRET_KEY"); v != "" {
		cfg.S3.SecretKey = v
	}

	return nil
}
