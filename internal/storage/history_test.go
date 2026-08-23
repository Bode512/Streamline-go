package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreMigratesLegacyJSONAndUpserts(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "streamline.db")
	legacyPath := filepath.Join(dir, "history.json")
	legacy := []HistoryItem{{
		ID:           "vid_1",
		DeviceID:     "phone-1",
		DeviceInfo:   "Test phone",
		Filename:     "clip.mp4",
		OriginalSize: 123,
		Status:       "ready",
		UploadTime:   100,
	}}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	items, err := store.MigrateJSON(legacyPath, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Filename != "clip.mp4" {
		t.Fatalf("unexpected migrated items: %+v", items)
	}

	legacy[0].Status = "downloaded"
	if err := store.Upsert(legacy[0]); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "downloaded" {
		t.Fatalf("upsert did not update item: %+v", items)
	}

	if err := store.Remove("clip.mp4", "phone-1"); err != nil {
		t.Fatal(err)
	}
	items, err = store.List(1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("item was not removed: %+v", items)
	}
}
