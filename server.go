// server.go
// Migración idiomática de server.c (Mongoose) a Go estándar.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"Streamline/internal/config"
	"Streamline/internal/storage"
	"github.com/skip2/go-qrcode"
)

const (
	maxUploadSize   int64 = 1 << 30 // 1 GB, equivalente a MG_MAX_RECV_SIZE
	maxHistoryItems       = 1000
)

const (
	statusUploading  = "uploading"
	statusProcessing = "processing"
	statusReady      = "ready"
	statusFailed     = "failed"
	statusDownloaded = "downloaded"
	statusCanceled   = "canceled"
)

// HistoryItem representa una entrada del historial.
// Equivale a HistoryItem en C.
type HistoryItem = storage.HistoryItem

var (
	historyMu     sync.Mutex
	history       = make([]HistoryItem, 0, maxHistoryItems)
	historyStore  *storage.Store
	runtimeConfig = config.Load()
)

// EnqueueFunc permite inyectar el envío a la cola de procesamiento real.
// El C original llama a enqueue(&miCola, fpath). Sustitúyela por la
// integración con el paquete queue de tu proyecto Go.
var EnqueueFunc func(path string) = func(path string) {
	log.Printf("[QUEUE] pendiente de integración: %s", path)
}

func enqueueFile(path string) {
	if EnqueueFunc != nil {
		EnqueueFunc(path)
	}
}

// nombreSeguro replica las comprobaciones de nombre_seguro() del C
// y añade filepath.Base para prevenir path traversal.
func nombreSeguro(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.Base(name) != name {
		return false
	}
	for _, r := range name {
		if r < 32 || strings.ContainsRune(`/\:"'&|;<>`+"`", r) {
			return false
		}
	}
	return true
}

// obtenerIPLocal devuelve la primera IPv4 no loopback.
// Equivale a obtener_ip_local().
func obtenerIPLocal() string {
	if connection, err := net.Dial("udp4", "8.8.8.8:80"); err == nil {
		defer connection.Close()
		if address, ok := connection.LocalAddr().(*net.UDPAddr); ok && address.IP.To4() != nil {
			return address.IP.To4().String()
		}
	}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	linkLocal := ""
	for _, addr := range addresses {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				if ip4.IsLinkLocalUnicast() {
					linkLocal = ip4.String()
					continue
				}
				return ip4.String()
			}
		}
	}
	if linkLocal != "" {
		return linkLocal
	}
	return "127.0.0.1"
}

// cargarHistorial carga history.json al iniciar.
func cargarHistorial() {
	store, err := storage.Open(runtimeConfig.DatabasePath)
	if err == nil {
		historyStore = store
		loaded, migrateErr := store.MigrateJSON("history.json", maxHistoryItems)
		if migrateErr == nil {
			historyMu.Lock()
			history = loaded
			historyMu.Unlock()
			return
		}
		log.Printf("[HIST] error cargando SQLite: %v", migrateErr)
	} else {
		log.Printf("[HIST] SQLite no disponible, usando JSON: %v", err)
	}

	data, err := os.ReadFile("history.json")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[HIST] error leyendo history.json: %v", err)
		}
		return
	}
	var loaded []HistoryItem
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("[HIST] error parseando history.json: %v", err)
		return
	}
	historyMu.Lock()
	if len(loaded) > maxHistoryItems {
		loaded = loaded[:maxHistoryItems]
	}
	history = loaded
	historyMu.Unlock()
}

// guardarHistorial escribe history.json de forma atómica.
// Debe llamarse con historyMu bloqueado.
func guardarHistorialLocked() {
	if historyStore != nil {
		if err := historyStore.ReplaceAll(history); err != nil {
			log.Printf("[HIST] error escribiendo SQLite: %v", err)
		}
	}

	tmp, err := os.CreateTemp(".", "history.json.part")
	if err != nil {
		log.Printf("[HIST] no se pudo crear temporal: %v", err)
		return
	}
	tmpName := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(history); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		log.Printf("[HIST] error escribiendo JSON: %v", err)
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		log.Printf("[HIST] error sync: %v", err)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		log.Printf("[HIST] error cerrando temporal: %v", err)
		return
	}
	if err := os.Rename(tmpName, "history.json"); err != nil {
		os.Remove(tmpName)
		log.Printf("[HIST] error renombrando: %v", err)
	}
}

// histUpsert actualiza o crea una entrada de historial.
// Equivale a hist_upsert() del C.
func histUpsert(deviceID, deviceInfo, filename string, originalSize int64, status string) {
	if filename == "" {
		return
	}
	historyMu.Lock()
	defer historyMu.Unlock()

	// Actualizar entrada existente.
	for i := range history {
		if history[i].Filename == filename &&
			(deviceID == "" || history[i].DeviceID == deviceID) {
			if status != "" {
				history[i].Status = status
			}
			if originalSize != 0 {
				history[i].OriginalSize = originalSize
			}
			guardarHistorialLocked()
			events.publish(StreamEvent{Type: "history.updated", Filename: filename, DeviceID: history[i].DeviceID, Status: history[i].Status})
			return
		}
	}

	// Crear nueva.
	if len(history) >= maxHistoryItems {
		return
	}
	id := fmt.Sprintf("vid_%d_%d", time.Now().Unix(), rand.Intn(9999))
	dev := deviceID
	if dev == "" {
		dev = "anonimo"
	}
	info := deviceInfo
	if info == "" {
		info = "Movil"
	}
	st := status
	if st == "" {
		st = statusProcessing
	}
	it := HistoryItem{
		ID:           id,
		DeviceID:     dev,
		DeviceInfo:   info,
		Filename:     filename,
		OriginalSize: originalSize,
		Status:       st,
		UploadTime:   time.Now().Unix(),
	}
	history = append(history, it)
	guardarHistorialLocked()
	events.publish(StreamEvent{Type: "history.updated", Filename: filename, DeviceID: dev, Status: st})
}

// histRemove elimina una entrada del historial.
func histRemove(filename, deviceID string) {
	historyMu.Lock()
	defer historyMu.Unlock()
	for i := range history {
		if history[i].Filename == filename &&
			(deviceID == "" || history[i].DeviceID == deviceID) {
			history = append(history[:i], history[i+1:]...)
			guardarHistorialLocked()
			break
		}
	}
}

// historyStatus comprueba si una entrada tiene un estado concreto.
func historyStatus(filename, deviceID, wanted string) bool {
	historyMu.Lock()
	defer historyMu.Unlock()
	for _, it := range history {
		if it.Filename == filename &&
			(deviceID == "" || it.DeviceID == deviceID) {
			return it.Status == wanted
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[HTTP] error escribiendo JSON: %v", err)
	}
}

func writeErrorJSON(w http.ResponseWriter, status int, msg string, extra map[string]any) {
	payload := map[string]any{"status": "error", "msg": msg}
	for k, v := range extra {
		payload[k] = v
	}
	writeJSON(w, status, payload)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, sHTML)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, sDashHTML)
}

// handleUpload procesa POST /upload con body de un bloque.
// Replica la lógica de subida reanudable del C, pero transmite el body
// directamente a disco sin acumularlo en memoria.
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	filename := q.Get("file")
	deviceID := q.Get("deviceId")
	deviceInfo := q.Get("deviceInfo")
	offsetStr := q.Get("offset")
	totalStr := q.Get("total")

	if filename == "" {
		filename = fmt.Sprintf("video_%d.mp4", time.Now().Unix())
	}
	if !nombreSeguro(filename) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid filename", nil)
		return
	}

	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil {
		offset = 0
	}
	total, err := strconv.ParseUint(totalStr, 10, 64)
	if err != nil || total == 0 || offset > total {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid upload offset", nil)
		return
	}

	if deviceID == "" {
		deviceID = "anonimo"
	}

	if err := crearDirectorio(runtimeConfig.InputDir); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "Cannot create videos dir", nil)
		return
	}
	finalPath := filepath.Join(runtimeConfig.InputDir, filename)
	tempPath := filepath.Join(runtimeConfig.InputDir, ".upload_"+filename+".part")

	// En offset 0 no puede existir ya el fichero final.
	if offset == 0 && existeFichero(finalPath) {
		writeErrorJSON(w, http.StatusConflict, "File already exists", nil)
		return
	}

	partialInfo, statErr := os.Stat(tempPath)
	var partialSize uint64
	if statErr == nil {
		partialSize = uint64(partialInfo.Size())
	} else if !os.IsNotExist(statErr) {
		writeErrorJSON(w, http.StatusInternalServerError, "Cannot stat partial", nil)
		return
	}
	if partialSize != offset {
		writeErrorJSON(w, http.StatusConflict, "Upload offset mismatch", map[string]any{"offset": partialSize})
		return
	}

	// Límite de lectura: el bloque no puede superar lo que falta ni el máximo global.
	remaining := total - offset
	limit := maxUploadSize
	if remaining < uint64(limit) {
		limit = int64(remaining)
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)

	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if offset == 0 {
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	file, err := os.OpenFile(tempPath, flags, 0o644)
	if err != nil {
		histUpsert(deviceID, deviceInfo, filename, 0, statusFailed)
		writeErrorJSON(w, http.StatusInternalServerError, "Cannot open file", nil)
		return
	}

	written, copyErr := io.Copy(file, r.Body)
	closeErr := file.Close()

	if copyErr != nil {
		os.Remove(tempPath)
		histUpsert(deviceID, deviceInfo, filename, int64(written), statusFailed)
		writeErrorJSON(w, http.StatusRequestEntityTooLarge, "Upload too large", nil)
		return
	}
	if closeErr != nil {
		os.Remove(tempPath)
		histUpsert(deviceID, deviceInfo, filename, int64(written), statusFailed)
		writeErrorJSON(w, http.StatusInternalServerError, "Upload write failed", nil)
		return
	}

	// Validar Content-Length cuando esté disponible.
	if r.ContentLength >= 0 && r.ContentLength != int64(written) {
		os.Remove(tempPath)
		histUpsert(deviceID, deviceInfo, filename, int64(written), statusFailed)
		writeErrorJSON(w, http.StatusBadRequest, "Incomplete upload", nil)
		return
	}

	received := offset + uint64(written)
	if received > total {
		os.Remove(tempPath)
		histUpsert(deviceID, deviceInfo, filename, int64(written), statusFailed)
		writeErrorJSON(w, http.StatusInternalServerError, "Upload write failed", nil)
		return
	}

	log.Printf("[UPLOAD] %s offset=%d received=%d total=%d device=%s", filename, offset, received, total, deviceID)

	// Aún faltan bloques.
	if received < total {
		histUpsert(deviceID, deviceInfo, filename, int64(received), statusUploading)
		writeJSON(w, http.StatusAccepted, map[string]any{"status": statusUploading, "offset": received})
		return
	}

	// Completado: publicar el fichero.
	if err := os.Rename(tempPath, finalPath); err != nil || !existeFichero(finalPath) {
		os.Remove(tempPath)
		histUpsert(deviceID, deviceInfo, filename, int64(received), statusFailed)
		writeErrorJSON(w, http.StatusInternalServerError, "Upload publish failed", nil)
		return
	}

	log.Printf("[UPLOAD] Completo: %s (%d bytes)", finalPath, received)
	histUpsert(deviceID, deviceInfo, filename, int64(received), statusProcessing)
	enqueueFile(finalPath)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": statusProcessing,
		"file":   filename,
	})
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("deviceId")

	historyMu.Lock()
	items := make([]HistoryItem, 0, len(history))
	for _, it := range history {
		if deviceID == "" || it.DeviceID == deviceID {
			items = append(items, it)
		}
	}
	historyMu.Unlock()

	writeJSON(w, http.StatusOK, items)
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	channel := events.subscribe()
	defer events.unsubscribe(channel)

	_, _ = io.WriteString(w, ": connected\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-channel:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: streamline\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func handleShareQR(w http.ResponseWriter, r *http.Request) {
	port := r.URL.Query().Get("port")
	if port == "" {
		port = runtimeConfig.Port
	}
	shareURL := "http://" + obtenerIPLocal() + ":" + port + "/"
	png, err := qrcode.Encode(shareURL, qrcode.Medium, 512)
	if err != nil {
		http.Error(w, "could not generate QR", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func handleNetwork(w http.ResponseWriter, r *http.Request) {
	port := r.URL.Query().Get("port")
	if port == "" {
		port = runtimeConfig.Port
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"port":       port,
		"shareURL":   "http://" + obtenerIPLocal() + ":" + port + "/",
		"interfaces": localInterfaces(),
	})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"port":         runtimeConfig.Port,
		"inputDir":     runtimeConfig.InputDir,
		"outputDir":    runtimeConfig.OutputDir,
		"databasePath": runtimeConfig.DatabasePath,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleUploadOffset(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	var offset uint64
	if nombreSeguro(filename) {
		partial := filepath.Join(runtimeConfig.InputDir, ".upload_"+filename+".part")
		if info, err := os.Stat(partial); err == nil {
			offset = uint64(info.Size())
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"offset": offset})
}

func handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filename := r.URL.Query().Get("file")
	if !nombreSeguro(filename) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid filename", nil)
		return
	}
	if !jobs.cancelJob(filename) {
		writeErrorJSON(w, http.StatusNotFound, "Job is not running", nil)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": statusCanceled, "file": filename})
}

func handleRetryJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filename := r.URL.Query().Get("file")
	if !nombreSeguro(filename) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid filename", nil)
		return
	}
	path := filepath.Join(runtimeConfig.InputDir, filename)
	if !existeFichero(path) {
		writeErrorJSON(w, http.StatusNotFound, "Input file is unavailable", nil)
		return
	}
	histUpsert("", "", filename, 0, statusProcessing)
	enqueueFile(path)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": statusProcessing, "file": filename})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	deviceID := r.URL.Query().Get("deviceId")
	if !nombreSeguro(filename) {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(runtimeConfig.OutputDir, filename)
	if !historyStatus(filename, deviceID, statusReady) || !existeFichero(path) {
		http.NotFound(w, r)
		return
	}
	log.Printf("[DOWNLOAD] %s -> %s", filename, deviceID)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(w, r, path)
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	deviceID := r.URL.Query().Get("deviceId")
	if !nombreSeguro(filename) || !historyStatus(filename, deviceID, statusReady) {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(runtimeConfig.OutputDir, filename)
	if !existeFichero(path) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func handleCleanup(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	deviceID := r.URL.Query().Get("deviceId")
	purge := r.URL.Query().Get("purge") == "1"

	if filename != "" && nombreSeguro(filename) {
		_ = os.Remove(filepath.Join(runtimeConfig.OutputDir, filename))
		_ = os.Remove(filepath.Join(runtimeConfig.InputDir, filename))
		if purge {
			histRemove(filename, deviceID)
			log.Printf("[CLEANUP] Eliminado del servidor y del historial: %s", filename)
		} else {
			histUpsert(deviceID, "", filename, 0, statusDownloaded)
			log.Printf("[CLEANUP] Eliminado: %s", filename)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	type deviceAgg struct {
		DeviceID   string `json:"deviceId"`
		DeviceInfo string `json:"deviceInfo"`
		Videos     int64  `json:"videos"`
		TotalSize  int64  `json:"totalSize"`
		Active     int64  `json:"active"`
		Online     bool   `json:"online"`
		Last       int64  `json:"last"`
	}

	historyMu.Lock()
	agg := make(map[string]*deviceAgg)
	for i := range history {
		it := &history[i]
		d := agg[it.DeviceID]
		if d == nil {
			d = &deviceAgg{DeviceID: it.DeviceID, DeviceInfo: it.DeviceInfo}
			agg[it.DeviceID] = d
		}
		d.Videos++
		d.TotalSize += it.OriginalSize
		if it.UploadTime > d.Last {
			d.Last = it.UploadTime
		}
		if existeFichero(filepath.Join(runtimeConfig.InputDir, it.Filename)) ||
			existeFichero(filepath.Join(runtimeConfig.OutputDir, it.Filename)) {
			d.Active++
		}
	}
	historyMu.Unlock()

	now := time.Now().Unix()
	devices := make([]deviceAgg, 0, len(agg))
	for _, d := range agg {
		d.Online = d.Active > 0 || (now-d.Last < 10*60)
		devices = append(devices, *d)
	}
	writeJSON(w, http.StatusOK, devices)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	var videos, processing, ready, downloaded, bytes int64

	historyMu.Lock()
	for i := range history {
		it := &history[i]
		if info, err := os.Stat(filepath.Join(runtimeConfig.InputDir, it.Filename)); err == nil {
			videos++
			processing++
			bytes += info.Size()
		} else if info, err := os.Stat(filepath.Join(runtimeConfig.OutputDir, it.Filename)); err == nil {
			videos++
			ready++
			bytes += info.Size()
		} else {
			downloaded++
		}
	}
	historyMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"videos":     videos,
		"processing": processing,
		"ready":      ready,
		"downloaded": downloaded,
		"bytes":      bytes,
	})
}

// IniciarServidorMongoose arranca el servidor HTTP.
// Equivale a iniciar_servidor_mongoose().
func IniciarServidorMongoose(puerto string) error {
	runtimeConfig.Port = puerto
	ip := obtenerIPLocal()

	if err := crearDirectorio(runtimeConfig.InputDir); err != nil {
		return fmt.Errorf("crear videos: %w", err)
	}
	if err := crearDirectorio(runtimeConfig.OutputDir); err != nil {
		return fmt.Errorf("crear convertidos: %w", err)
	}
	cargarHistorial()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/upload", handleUpload)
	mux.HandleFunc("/api/history", handleHistory)
	mux.HandleFunc("/api/events", handleEvents)
	mux.HandleFunc("/api/qr", handleShareQR)
	mux.HandleFunc("/api/network", handleNetwork)
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/upload-offset", handleUploadOffset)
	mux.HandleFunc("/api/jobs/cancel", handleCancelJob)
	mux.HandleFunc("/api/jobs/retry", handleRetryJob)
	mux.HandleFunc("/download", handleDownload)
	mux.HandleFunc("/preview", handlePreview)
	mux.HandleFunc("/api/cleanup", handleCleanup)
	mux.HandleFunc("/api/devices", handleDevices)
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/dashboard", handleDashboard)

	srv := &http.Server{
		Addr:              ":" + puerto,
		Handler:           corsMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute, // subidas por bloques de 4 MB
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}

	// Graceful shutdown con Ctrl+C.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[SERVER] shutdown: %v", err)
		}
	}()

	log.Printf("========================================")
	log.Printf("  STREAMLINE — Video Server (Go)")
	log.Printf("========================================")
	log.Printf("  IP local : %s", ip)
	log.Printf("  Puerto   : %s", puerto)
	log.Printf("  Abre en móvil: http://%s:%s", ip, puerto)
	log.Printf("========================================")

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("servidor: %w", err)
	}
	return nil
}

// ───────────────────────── HTML embebido ─────────────────────────

// sHTML es una versión funcional equivalente a la interfaz del C.
// Se ha simplificado ligeramente y el JS evita template literals para
// poder usar raw strings de Go sin conflictos con backticks.
const sHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Streamline — Video Server</title>
<style>
:root{font-family:Georgia,'Times New Roman',serif;color:#f4f1e8;background:#101313}
*{box-sizing:border-box}
body{background:radial-gradient(circle at 85% 5%,#244846 0,transparent 28%),#101313;max-width:760px;margin:0 auto;padding:20px;color:#f4f1e8}
body:before{content:'STREAMLINE';display:block;color:#d6a85e;font:11px Arial,sans-serif;letter-spacing:.22em;border-bottom:1px solid #3b4c48;padding:0 0 18px;margin-bottom:26px}
.card{background:#18201f;border:1px solid #3b4c48;padding:24px;margin-bottom:14px}
h1{font-size:28px;font-weight:normal;margin:0 0 8px}p{font:14px Arial,sans-serif;color:#9bb1a7;line-height:1.5}
input[type=file]{width:100%;margin:16px 0;padding:14px;border:1px dashed #53645f;color:#9bb1a7;background:#101313}
button{background:#d6a85e;color:#101313;border:0;padding:12px 18px;font-weight:bold;cursor:pointer}button:disabled{opacity:.4;cursor:not-allowed}
.file-row{display:flex;justify-content:space-between;align-items:center;gap:12px;padding:13px;border:1px solid #3b4c48;background:#202d2a;margin-top:8px}.file-row a{color:#d6a85e;font:bold 12px Arial,sans-serif;text-decoration:none;white-space:nowrap}
.progress{height:4px;background:#35433f;margin-top:8px}.progress>div{height:100%;background:#d6a85e;width:0}
.tab{margin:0 6px 8px 0;background:#263632;color:#9bb1a7;padding:10px 12px}.tab.on{background:#d6a85e;color:#101313}
.thumb{width:64px;height:42px;object-fit:cover;background:#2b4541}.file-row>div:first-child{flex:1;min-width:0}.file-row>div:first-child>div:first-child{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
a{color:#d6a85e}
</style>
</head>
<body>
<div class="card">
<h1>Transferencia de Video</h1>
<p>Subida directa por LAN. Archivos de hasta 1 GB.</p>
<input type="file" id="fi" accept="video/*" multiple>
<div id="flist"></div>
<button id="start-btn" onclick="startUpload()">Iniciar Transferencia</button>
</div>
<div class="card">
<h1>Panel de Archivos</h1>
<div>
<button class="tab on" id="t-ready" onclick="setTab('ready',this)">Listos (<span id="c-ready">0</span>)</button>
<button class="tab" id="t-proc" onclick="setTab('processing',this)">En Proceso (<span id="c-proc">0</span>)</button>
<button class="tab" id="t-hist" onclick="setTab('downloaded',this)">Historial (<span id="c-hist">0</span>)</button>
</div>
<div id="tab-ct"></div>
</div>
<script>
var did=localStorage.getItem('sl_did');
if(!did){did='dev_'+Math.random().toString(36).slice(2,12);localStorage.setItem('sl_did',did);}
var dinfo=navigator.userAgent.indexOf('iPhone')>=0?'iPhone':navigator.userAgent.indexOf('Android')>=0?'Android':'Desktop';
var files=[],items=[],tab='ready';
document.getElementById('fi').addEventListener('change',function(e){files=Array.from(e.target.files);renderFiles();});
function renderFiles(){
var fl=document.getElementById('flist');
if(!files.length){fl.innerHTML='';return;}
var html='';
files.forEach(function(f,i){
var mb=(f.size/1048576).toFixed(1);
html+='<div class="file-row"><div style="flex:1"><div>'+f.name+'</div><div style="font-size:12px;color:#888">'+mb+' MB</div><div class="progress"><div id="pb-'+i+'"></div></div></div><span id="st-'+i+'" style="font-size:12px;color:#888">Pendiente</span></div>';
});
fl.innerHTML=html;
}
function startUpload(){
if(!files.length)return;
document.getElementById('start-btn').disabled=true;
uploadOne(0);
}
var chunk=4*1024*1024;
function uploadOne(i){
if(i>=files.length){
document.getElementById('start-btn').disabled=false;
fetchHistory();setActive('processing');return;
}
var f=files[i],st=document.getElementById('st-'+i),pb=document.getElementById('pb-'+i);
fetch('/api/upload-offset?file='+encodeURIComponent(f.name))
.then(function(r){return r.json()})
.then(function(d){uploadBlock(i,f,Number(d.offset||0),st,pb)})
.catch(function(){uploadBlock(i,f,0,st,pb)});
}
function uploadBlock(i,f,offset,st,pb){
if(offset>=f.size){uploadOne(i+1);return;}
var end=Math.min(offset+chunk,f.size);
var block=f.slice(offset,end);
var url='/upload?file='+encodeURIComponent(f.name)+'&deviceId='+did+'&deviceInfo='+encodeURIComponent(dinfo)+'&offset='+offset+'&total='+f.size;
var xhr=new XMLHttpRequest();
xhr.open('POST',url,true);
xhr.setRequestHeader('Content-Type','application/octet-stream');
xhr.upload.onprogress=function(e){
if(e.lengthComputable){var p=Math.round((offset+e.loaded)/f.size*100);if(pb)pb.style.width=p+'%';if(st)st.textContent=p+'%';}
};
xhr.onload=function(){
if(xhr.status===200||xhr.status===202){
var data=JSON.parse(xhr.responseText||'{}');
if(data.status==='processing'){if(st)st.textContent='Procesando';if(pb)pb.style.width='100%';uploadOne(i+1);return;}
uploadBlock(i,f,Number(data.offset||end),st,pb);
}else{if(st)st.textContent='Error '+xhr.status;}
};
xhr.onerror=function(){if(st)st.textContent='Conexión perdida; reintenta';};
if(st)st.textContent='Conectando...';
xhr.send(block);
}
function setTab(name,btn){
tab=name;
document.querySelectorAll('.tab').forEach(function(b){b.classList.remove('on')});
btn.classList.add('on');
renderTab();
}
function setActive(name){
tab=name;
var map={ready:'t-ready',processing:'t-proc',downloaded:'t-hist'};
document.querySelectorAll('.tab').forEach(function(b){b.classList.remove('on')});
var el=document.getElementById(map[name]);
if(el)el.classList.add('on');
renderTab();
}
function fetchHistory(){
fetch('/api/history?deviceId='+did).then(function(r){return r.json()}).then(function(d){items=d;renderTab();}).catch(function(){});
}
function renderTab(){
var ready=items.filter(function(x){return x.status==='ready'});
var proc=items.filter(function(x){return x.status==='processing'||x.status==='uploading'});
var hist=items.filter(function(x){return x.status==='downloaded'});
document.getElementById('c-ready').textContent=ready.length;
document.getElementById('c-proc').textContent=proc.length;
document.getElementById('c-hist').textContent=hist.length;
var cur=tab==='ready'?ready:tab==='processing'?proc:hist;
var ct=document.getElementById('tab-ct');
if(!cur.length){ct.innerHTML='<div style="text-align:center;padding:32px;color:#888">Sin archivos en esta categoría.</div>';return;}
var html='';
cur.forEach(function(it){
var mb=(it.originalSize/1048576).toFixed(1);
if(tab==='ready'){
html+='<div class="file-row"><video class="thumb" src="/preview?file='+encodeURIComponent(it.filename)+'" preload="metadata" muted></video><div><div>'+it.filename+'</div><div style="font-size:12px;color:#9bb1a7">'+mb+' MB — Listo para descargar</div></div><a href="/download?file='+encodeURIComponent(it.filename)+'&deviceId='+did+'" download="'+it.filename+'">Descargar</a><button onclick="delEntry(\''+it.filename+'\',0)">Eliminar</button></div>';
}else if(tab==='processing'){
html+='<div class="file-row"><div><div>'+it.filename+'</div><div style="font-size:12px;color:#888">'+mb+' MB — Procesando...</div></div><button onclick="delEntry(\''+it.filename+'\',1)">Eliminar</button></div>';
}else{
html+='<div class="file-row"><div><div>'+it.filename+'</div><div style="font-size:12px;color:#888">Descargado y eliminado</div></div><button onclick="delEntry(\''+it.filename+'\',1)">Quitar</button></div>';
}
});
ct.innerHTML=html;
}
function delEntry(name,purge){
fetch('/api/cleanup?file='+encodeURIComponent(name)+'&deviceId='+did+(purge?'&purge=1':'')).then(function(r){return r.json()}).then(function(){fetchHistory();});
}
setInterval(fetchHistory,2000);
fetchHistory();
</script>
</body></html>`
