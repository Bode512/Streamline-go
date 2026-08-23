package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port          string
	InputDir      string
	OutputDir     string
	DatabasePath  string
	HandBrakePath string
}

func Load() Config {
	result := Config{
		Port:          envOr("PORT", "8000"),
		InputDir:      envOr("STREAMLINE_INPUT_DIR", "videos"),
		OutputDir:     envOr("STREAMLINE_OUTPUT_DIR", "convertidos"),
		DatabasePath:  envOr("STREAMLINE_DATABASE", "streamline.db"),
		HandBrakePath: os.Getenv("HANDBRAKE_PATH"),
	}
	if result.HandBrakePath == "" {
		result.HandBrakePath = os.Getenv("HANDBRAKECLI_PATH")
	}
	return result
}

func (c Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("port is required")
	}
	if filepath.Clean(c.InputDir) == filepath.Clean(c.OutputDir) {
		return fmt.Errorf("input and output directories must differ")
	}
	if c.DatabasePath == "" {
		return fmt.Errorf("database path is required")
	}
	return nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
