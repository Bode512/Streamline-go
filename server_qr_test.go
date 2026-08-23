package main

import (
	"net/http/httptest"
	"testing"
)

func TestHandleShareQRReturnsPNG(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/qr?port=8000", nil)
	response := httptest.NewRecorder()

	handleShareQR(response, request)

	if response.Code != 200 {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content type = %q, want image/png", got)
	}
	body := response.Body.Bytes()
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("response is not a PNG")
	}
}
