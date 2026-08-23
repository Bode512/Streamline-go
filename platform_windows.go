//go:build windows

package main

import "syscall"

// configurarConsola establece la consola de Windows en UTF-8.
// Equivale a configurar_consola() del C para Windows.
func configurarConsola() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleOutputCP := kernel32.NewProc("SetConsoleOutputCP")
	setConsoleCP := kernel32.NewProc("SetConsoleCP")

	const CP_UTF8 = 65001

	// Los valores de retorno no son críticos; se ignoran
	// deliberadamente porque el programa puede seguir aunque falle.
	_, _, _ = setConsoleOutputCP.Call(CP_UTF8)
	_, _, _ = setConsoleCP.Call(CP_UTF8)
}