package main

import (
	"context"
	"testing"
)

func TestJobManagerCancel(t *testing.T) {
	manager := newJobManager()
	ctx := manager.start(context.Background(), "clip.mp4")
	if !manager.cancelJob("clip.mp4") {
		t.Fatal("expected active job")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("job context was not canceled")
	}
	manager.finish("clip.mp4")
	if manager.cancelJob("clip.mp4") {
		t.Fatal("finished job should not be cancelable")
	}
}
