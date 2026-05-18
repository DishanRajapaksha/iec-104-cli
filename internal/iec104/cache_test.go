package iec104

import (
	"testing"
	"time"
)

func TestLatestCacheSnapshotMarksStale(t *testing.T) {
	cache := NewLatestCache()
	cache.Update(PointValue{
		Timestamp:     time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		CommonAddress: 1,
		IOA:           1001,
		Value:         12.34,
	})

	values := cache.Snapshot(time.Date(2026, 5, 18, 12, 1, 0, 0, time.UTC), 30*time.Second)
	if len(values) != 1 {
		t.Fatalf("snapshot length = %d, want 1", len(values))
	}
	if !values[0].Stale {
		t.Fatal("value was not marked stale")
	}
}

func TestLatestCacheUpdateReplacesSameKey(t *testing.T) {
	cache := NewLatestCache()
	cache.Update(PointValue{CommonAddress: 1, IOA: 1001, Value: 1})
	cache.Update(PointValue{CommonAddress: 1, IOA: 1001, Value: 2})

	values := cache.Snapshot(time.Now(), 0)
	if len(values) != 1 {
		t.Fatalf("snapshot length = %d, want 1", len(values))
	}
	if values[0].Value != 2 {
		t.Fatalf("value = %v, want 2", values[0].Value)
	}
}
