package iec104

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistentCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "latest.json")
	store := NewPersistentCache(path)
	values := []PointValue{{
		Timestamp:     time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		CommonAddress: 1,
		IOA:           1001,
		Name:          "active_power",
		Type:          "float",
		Value:         12.34,
		Unit:          "MW",
	}}

	if err := store.Save(values); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded count = %d, want 1", len(loaded))
	}
	if loaded[0].Name != "active_power" || loaded[0].IOA != 1001 {
		t.Fatalf("loaded = %#v", loaded[0])
	}
}

func TestPersistentCacheMissingFile(t *testing.T) {
	store := NewPersistentCache(filepath.Join(t.TempDir(), "missing.json"))
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("loaded count = %d, want 0", len(loaded))
	}
}

func TestLatestCacheSeed(t *testing.T) {
	cache := NewLatestCache()
	cache.Seed([]PointValue{{CommonAddress: 1, IOA: 1001, Value: 1}})
	cache.Update(PointValue{CommonAddress: 1, IOA: 1001, Value: 2})
	values := cache.Snapshot(time.Now(), 0)
	if len(values) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(values))
	}
	if values[0].Value != 2 {
		t.Fatalf("value = %v, want 2", values[0].Value)
	}
}

func TestPersistentCacheInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPersistentCache(path).Load(); err == nil {
		t.Fatal("Load returned nil error for invalid JSON")
	}
}
