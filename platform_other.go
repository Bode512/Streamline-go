//go:build !windows

package main

// configurarConsola no hace nada en sistemas Unix.
// En Linux/macOS el locale se hereda del entorno; la salida estándar
// de Go ya es UTF-8.
// Equivale a configurar_consola() del C para Unix.
func configurarConsola() {
}