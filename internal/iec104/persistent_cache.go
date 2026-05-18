package iec104

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type PersistentCache struct {
	path string
}

func NewPersistentCache(path string) *PersistentCache {
	return &PersistentCache{path: path}
}

func (c *PersistentCache) Load() ([]PointValue, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read cache %q: %w", c.path, err)
	}
	var values []PointValue
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse cache %q: %w", c.path, err)
	}
	return values, nil
}

func (c *PersistentCache) Save(values []PointValue) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode cache: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(c.path), ".cache-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create cache temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write cache temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close cache temp file: %w", err)
	}
	if err := os.Rename(tmpName, c.path); err != nil {
		return fmt.Errorf("failed to replace cache %q: %w", c.path, err)
	}
	return nil
}

func (c *LatestCache) Seed(values []PointValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, value := range values {
		c.values[PointKey{CommonAddress: value.CommonAddress, IOA: value.IOA}] = value
	}
}

func (c *LatestCache) SaveTo(store *PersistentCache) error {
	return store.Save(c.Snapshot(time.Now(), 0))
}
