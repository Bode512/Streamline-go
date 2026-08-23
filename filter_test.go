package main

import "testing"

func TestEsVideo(t *testing.T) {
	casos := []struct {
		nombre string
		quiere bool
	}{
		{"video.mp4", true},
		{"video.MP4", true},
		{"video.Mkv", true},
		{"archivo.txt", false},
		{"sin_extension", false},
		{"video.mp4.exe", false},
		{"", false},
		{"videos/video.mov", true},
		{"video.", false},
	}

	for _, c := range casos {
		if got := EsVideo(c.nombre); got != c.quiere {
			t.Errorf("EsVideo(%q) = %v; se esperaba %v", c.nombre, got, c.quiere)
		}
	}
}