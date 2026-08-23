package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// VigilarDirectorio observa ruta y encola en cola todo archivo de vídeo
// nuevo o movido dentro de ese directorio.
//
// Se bloquea hasta que el contexto sea cancelado o falle la vigilancia.
// Equivale a vigilar_directorio() del C, pero con cancelación explícita.
func VigilarDirectorio(ctx context.Context, ruta string, cola *Cola) error {
	if cola == nil {
		return fmt.Errorf("cola nil")
	}
	if ruta == "" {
		return fmt.Errorf("ruta vacía")
	}

	info, err := os.Stat(ruta)
	if err != nil {
		return fmt.Errorf("stat %s: %w", ruta, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("no es un directorio: %s", ruta)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("crear watcher: %w", err)
	}
	defer watcher.Close()

	if err := filepath.WalkDir(ruta, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("registrar directorios: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher: %w", watchErr)
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create != 0 {
				if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
					if addErr := watcher.Add(event.Name); addErr != nil {
						return fmt.Errorf("registrar %s: %w", event.Name, addErr)
					}
				} else if EsVideo(event.Name) {
					cola.Enqueue(event.Name)
				}
			}
			if event.Op&fsnotify.Rename != 0 && EsVideo(event.Name) {
				cola.Enqueue(event.Name)
			}
		}
	}
}
