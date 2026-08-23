package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// HistoryItem is the durable representation of a processed upload.
type HistoryItem struct {
	ID           string `json:"id"`
	DeviceID     string `json:"deviceId"`
	DeviceInfo   string `json:"deviceInfo"`
	Filename     string `json:"filename"`
	OriginalSize int64  `json:"originalSize"`
	CurrentSize  int64  `json:"currentSize"`
	Status       string `json:"status"`
	UploadTime   int64  `json:"uploadTime"`
}

// Store owns the SQLite database used by Streamline.
type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	store := &Store{db: db}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS history (
 id TEXT PRIMARY KEY,
 device_id TEXT NOT NULL,
 device_info TEXT NOT NULL,
 filename TEXT NOT NULL,
 original_size INTEGER NOT NULL DEFAULT 0,
 status TEXT NOT NULL,
 upload_time INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_history_device_file ON history(device_id, filename);
CREATE INDEX IF NOT EXISTS idx_history_status ON history(status);`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Upsert(item HistoryItem) error {
	_, err := s.db.Exec(`INSERT INTO history
(id, device_id, device_info, filename, original_size, status, upload_time)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(device_id, filename) DO UPDATE SET
 device_info=excluded.device_info,
 original_size=CASE WHEN excluded.original_size <> 0 THEN excluded.original_size ELSE history.original_size END,
 status=CASE WHEN excluded.status <> '' THEN excluded.status ELSE history.status END`,
		item.ID, item.DeviceID, item.DeviceInfo, item.Filename, item.OriginalSize, item.Status, item.UploadTime)
	return err
}

func (s *Store) ReplaceAll(items []HistoryItem) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM history"); err != nil {
		tx.Rollback()
		return err
	}
	for _, item := range items {
		if _, err = tx.Exec(`INSERT INTO history
(id, device_id, device_info, filename, original_size, status, upload_time)
VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.DeviceID, item.DeviceInfo, item.Filename, item.OriginalSize, item.Status, item.UploadTime); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) List(limit int) ([]HistoryItem, error) {
	rows, err := s.db.Query(`SELECT id, device_id, device_info, filename, original_size, status, upload_time
FROM history ORDER BY upload_time DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]HistoryItem, 0)
	for rows.Next() {
		var item HistoryItem
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.DeviceInfo, &item.Filename, &item.OriginalSize, &item.Status, &item.UploadTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Remove(filename, deviceID string) error {
	if deviceID == "" {
		_, err := s.db.Exec("DELETE FROM history WHERE filename = ?", filename)
		return err
	}
	_, err := s.db.Exec("DELETE FROM history WHERE filename = ? AND device_id = ?", filename, deviceID)
	return err
}

// MigrateJSON imports the legacy file only when SQLite has no records.
func (s *Store) MigrateJSON(path string, limit int) ([]HistoryItem, error) {
	items, err := s.List(limit)
	if err != nil || len(items) > 0 {
		return items, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return items, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("decode legacy history: %w", err)
	}
	if len(items) > limit {
		items = items[:limit]
	}
	for i := range items {
		if items[i].UploadTime == 0 {
			items[i].UploadTime = time.Now().Unix()
		}
		if err := s.Upsert(items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}
