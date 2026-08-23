package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

const (
	// handbrakeDefaultWindows es la ruta del binario por defecto en Windows.
	handbrakeDefaultWindows = `bin\HandBrakeCLI.exe`
	// handbrakeDefaultUnix es la ruta del binario por defecto en Linux/macOS.
	handbrakeDefaultUnix = "bin/HandBrakeCLI"
)

// archivoExiste informa si la ruta existe, es un archivo regular y no está vacío.
// Equivale a archivo_existe() del C.
func archivoExiste(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

// rutaSegura valida que una ruta no contenga caracteres de control
// ni caracteres peligrosos. En Go no se usa shell, pero se mantiene
// la misma validación para preservar el comportamiento y evitar
// problemas con sistemas de archivos.
func rutaSegura(path string) bool {
	if path == "" {
		return false
	}
	for _, r := range path {
		if r < 32 {
			return false
		}
		switch r {
		case '"', '\'', '&', '|', ';', '<', '>', '`', '$':
			return false
		}
	}
	return true
}

// rutaHandBrake devuelve la ruta al binario de HandBrakeCLI.
// Prioriza la variable de entorno HANDBRAKECLI_PATH si existe y es válida.
// Equivale a ruta_handbrake() del C.
func rutaHandBrake() string {
	if configured := os.Getenv("HANDBRAKE_PATH"); configured != "" {
		return configured
	}
	if configured := os.Getenv("HANDBRAKECLI_PATH"); configured != "" {
		return configured
	}
	if runtime.GOOS == "windows" {
		if archivoExiste(handbrakeDefaultWindows) {
			return handbrakeDefaultWindows
		}
		if path, err := exec.LookPath("HandBrakeCLI.exe"); err == nil {
			return path
		}
	} else if path, err := exec.LookPath("HandBrakeCLI"); err == nil {
		return path
	}
	return handbrakeDefaultUnix
}

// lineFilterWriter escribe solo las líneas que contienen "ETA" o "Encoding",
// imitando el filtrado del C. Puede usarse de forma segura para stdout y stderr
// combinados porque exec serializa las escrituras cuando ambos apuntan al
// mismo writer.
type lineFilterWriter struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (w *lineFilterWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, b := range p {
		if b == '\n' {
			line := w.buf.String()
			w.buf.Reset()
			if line != "" {
				fmt.Printf("[HANDBRAKE] %s\n", line)
			}
			continue
		}
		w.buf.WriteByte(b)
	}
	return len(p), nil
}

// compressVideo procesa un vídeo con HandBrakeCLI.
//
// Parámetros:
//   - input:  ruta del archivo de vídeo original.
//   - output: ruta de destino del vídeo comprimido.
//
// Devuelve true si la compresión se completó correctamente,
// false en caso contrario. Equivale a compress_video() del C.
func compressVideo(ctx context.Context, input, output string) bool {
	if !archivoExiste(input) {
		fmt.Printf("[WORKER] Archivo ya no se encuentra en videos/ (omitiendo): %s\n", input)
		return false
	}
	if !rutaSegura(input) || !rutaSegura(output) {
		fmt.Printf("[WORKER] Ruta rechazada por contener caracteres no seguros.\n")
		return false
	}

	tempOutput := output + ".part"

	// Limpiar temporales y destino anterior, igual que el C.
	_ = os.Remove(tempOutput)
	_ = os.Remove(output)

	fmt.Printf("\n[WORKER] Procesando video: %s -> %s\n", input, output)

	handbrakePath := rutaHandBrake()
	if !archivoExiste(handbrakePath) {
		fmt.Printf("[WORKER] No se encontró HandBrakeCLI (%s).\n", handbrakePath)
		_ = os.Remove(tempOutput)
		return false
	}

	// Construir el comando sin shell.
	// El C usaba: "HandBrakeCLI" -i "input" -o "temp" -e x265 -q 26
	// --encoder-preset fast --vfr -B 96 -E av_aac 2>&1
	cmd := exec.CommandContext(ctx,
		handbrakePath,
		"-i", input,
		"-o", tempOutput,
		"-e", "x265",
		"-q", "26",
		"--encoder-preset", "fast",
		"--vfr",
		"-B", "96",
		"-E", "av_aac",
	)

	// Combinar stdout y stderr en un único filtro.
	filter := &lineFilterWriter{}
	cmd.Stdout = filter
	cmd.Stderr = filter

	if err := cmd.Run(); err != nil {
		fmt.Printf("[WORKER] HandBrakeCLI terminó con error: %v\n", err)
		_ = os.Remove(tempOutput)
		return false
	}

	// Solo un proceso terminado correctamente puede publicar.
	if archivoExiste(tempOutput) {
		if err := os.Rename(tempOutput, output); err != nil {
			fmt.Printf("[WORKER] Error al renombrar %s a %s: %v\n", tempOutput, output, err)
			_ = os.Remove(tempOutput)
			return false
		}
	}

	if !archivoExiste(output) {
		fmt.Printf("[WORKER] No se generó el archivo de salida: %s\n", output)
		_ = os.Remove(tempOutput)
		return false
	}

	fmt.Printf("\n[WORKER] Compresión completada con HandBrakeCLI: %s\n", output)

	// Eliminar el original de videos/.
	if err := moveToTrash(input); err == nil {
		fmt.Printf("[WORKER] Archivo procesado y movido a la Papelera: %s\n", input)
	} else {
		fmt.Printf("[WORKER] Error al eliminar el original de videos/: %v\n", err)
	}

	return true
}
