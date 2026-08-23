package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewareAllowsDesktopOrigin(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	request.Header.Set("Origin", "wails://wails.localhost")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS header: %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddlewareHandlesPreflight(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run for OPTIONS")
	}))
	request := httptest.NewRequest(http.MethodOptions, "/api/jobs/cancel", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}
