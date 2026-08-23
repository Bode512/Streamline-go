package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

var (
	// miCola es la cola global de trabajos. Definida en queue.go.
	// Equivale a la variable global del C.
	miCola = NuevaCola(8)

	// directorioSalida es el nombre del directorio que no debe
	// ser escaneado para evitar un bucle infinito.
	// En el C original estaba hardcodeado a
	// "C:\\Users\\User\\Videos\\convertidos".
	// En Go usamos un nombre relativo portable.
	directorioSalida = "convertidos"
)

// RecorrerDirectorios explora recursivamente rutaActual,
// encolando los archivos de vídeo que encuentre en miCola.
//
// Equivale a recorrer_directorios() del C.
// Devuelve error en lugar de imprimir perror y salir.
func RecorrerDirectorios(rutaActual string) error {
	// Verificar que la ruta inicial existe y es un directorio.
	info, err := os.Stat(rutaActual)
	if err != nil {
		return fmt.Errorf("stat %s: %w", rutaActual, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("no es un directorio: %s", rutaActual)
	}

	err = filepath.WalkDir(rutaActual, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// El C imprimía perror y continuaba con otras entradas.
			log.Printf("error accediendo a %s: %v", path, err)
			return nil
		}

		fmt.Printf("Entrada: %s\n", path)

		if d.IsDir() {
			fmt.Println(" -> es un directorio. Recursividad...")

			// Evitar entrar en el directorio de salida.
			// En el C original se comparaba con la ruta completa;
			// aquí comparamos con el nombre base para que sea portable.
			if filepath.Base(path) == directorioSalida {
				fmt.Println(" -> es la carpeta de salida, no entro para evitar bucle infinito")
				return filepath.SkipDir
			}
			return nil
		}

		fmt.Println(" -> es un archivo")

		if EsVideo(d.Name()) {
			fmt.Printf("Archivo valido: %s\n", path)
			miCola.Enqueue(path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("recorriendo %s: %w", rutaActual, err)
	}
	return nil
}
