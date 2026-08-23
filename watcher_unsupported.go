//go:build legacywatcher && !linux && !windows

package main

import (
	"context"
	"fmt"
)

func vigilarDirectorio(ctx context.Context, ruta string, cola *Cola) error {
	return fmt.Errorf("vigilancia de directorios no soportada en este sistema")
}
