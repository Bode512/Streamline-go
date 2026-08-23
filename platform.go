package main

import "os"

// crearDirectorio crea un directorio, incluyendo padres si es necesario.
// Si el directorio ya existe no devuelve error.
// Equivale a crear_directorio() del C (que devolvía 1 en éxito, 0 en error).
func crearDirectorio(ruta string) error {
	return os.MkdirAll(ruta, 0o755)
}

// existeFichero informa si la ruta existe, sea fichero o directorio.
// Equivale a existe_fichero() del C.
func existeFichero(ruta string) bool {
	_, err := os.Stat(ruta)
	return err == nil
}
