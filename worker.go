package main

import (
	"context"
	"log"
	"path/filepath"
)

// actualizarEstadoArchivo actualiza el estado de un archivo en el historial.
// Equivale a actualizar_estado_archivo() del C, que a su vez
// llamaba a hist_upsert(NULL, NULL, filename, status).
func actualizarEstadoArchivo(filename, status string) {
	histUpsert("", "", filename, 0, status)
}

// ProcesarCola consume elementos de miCola, comprime cada vídeo con
// HandBrakeCLI y actualiza el historial.
//
// Se bloquea hasta que la cola sea liberada mediante miCola.Liberar().
// Equivale a procesar_cola() del C.
func ProcesarCola(ctx context.Context, rutaSalida string) {
	// Asegurar que la carpeta de salida existe.
	// En el C esta creación estaba comentada; aquí se activa
	// para evitar fallos innecesarios de HandBrakeCLI.
	if err := crearDirectorio(rutaSalida); err != nil {
		log.Printf("[WORKER] no se pudo crear directorio de salida %s: %v", rutaSalida, err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		elemento, ok := miCola.EsperarElemento()
		if !ok {
			// Cola liberada y vacía: fin del worker.
			return
		}

		// Obtener el nombre base del archivo de entrada.
		nombreArchivo := filepath.Base(elemento.Ruta)

		// Construir la ruta final de salida.
		rutaFinal := filepath.Join(rutaSalida, nombreArchivo)

		log.Printf("[WORKER] Procesando: %s -> %s", elemento.Ruta, rutaFinal)

		ok = compressVideo(elemento.Ruta, rutaFinal)

		if ok {
			actualizarEstadoArchivo(nombreArchivo, "ready")
			log.Printf("[JOB %s] completed: resultado publicado", nombreArchivo)
		} else {
			actualizarEstadoArchivo(nombreArchivo, "failed")
			log.Printf("[JOB %s] failed: procesamiento invalido", nombreArchivo)
		}
	}
}
