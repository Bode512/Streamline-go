package main

import (
	"path/filepath"
	"strings"
)

// extensionesVideo contiene las extensiones aceptadas como vídeo.
// Equivale a la lista estática del C.
var extensionesVideo = map[string]struct{}{
	".mp4":  {},
	".mkv":  {},
	".avi":  {},
	".mov":  {},
	".wmv":  {},
	".flv":  {},
	".webm": {},
	".m4v":  {},
	".3gp":  {},
	".3g2":  {},
	".mts":  {},
	".m2ts": {},
	".ts":   {},
	".vob":  {},
	".mpeg": {},
	".mpg":  {},
}

// EsVideo informa si el nombre indicado corresponde a un archivo de vídeo.
//
// A diferencia del C, devuelve bool en lugar de const char* / NULL.
// En Go el nombre original ya está disponible en el llamador,
// por lo que no es necesario devolverlo de nuevo.
func EsVideo(nombre string) bool {
	if nombre == "" {
		return false
	}

	// filepath.Base permite usar tanto "video.mp4" como
	// "videos/video.mp4" sin que un punto en el directorio
	// interfiera en la detección.
	nombre = filepath.Base(nombre)

	// filepath.Ext devuelve la extensión desde el último punto.
	ext := strings.ToLower(filepath.Ext(nombre))

	_, ok := extensionesVideo[ext]
	return ok
}