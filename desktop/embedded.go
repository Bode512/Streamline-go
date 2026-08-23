package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed embedded/streamline-core.exe
var embeddedCore []byte

//go:embed embedded/HandBrakeCLI.exe
var embeddedHandBrake []byte

func extractEmbeddedCore() (string, error) {
	directory := filepath.Join(os.TempDir(), "Streamline", "core")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(directory, "streamline-core.exe")
	if err := os.WriteFile(path, embeddedCore, 0o700); err != nil {
		return "", fmt.Errorf("extraer Core: %w", err)
	}
	return path, nil
}

func extractEmbeddedHandBrake() (string, error) {
	directory := filepath.Join(os.TempDir(), "Streamline", "bin")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(directory, "HandBrakeCLI.exe")
	if err := os.WriteFile(path, embeddedHandBrake, 0o700); err != nil {
		return "", fmt.Errorf("extraer HandBrakeCLI: %w", err)
	}
	return path, nil
}
