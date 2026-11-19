package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

const (
	StoragePostgres = "postgres"
	StorageMemory   = "memory"
)

type Config struct {
	HTTPAddr    string
	StorageType string
	DataBaseUrl string
}

func Load() (
	*Config,
	error,
) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg := Config{
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
		StorageType: os.Getenv("STORAGE_TYPE"),
		DataBaseUrl: os.Getenv("DB_URL"),
	}

	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.StorageType == "" {
		cfg.StorageType = StorageMemory
	}

	switch cfg.StorageType {
	case StorageMemory:
	case StoragePostgres:
		if cfg.DataBaseUrl == "" {
			return nil, errors.New("DB_URL is required when using PostgreSQL storage")
		}
	default:
		return nil, fmt.Errorf("unsupported STORAGE_TYPE: %q", cfg.StorageType)
	}

	return &cfg, nil
}
