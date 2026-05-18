package iec104

import (
	"sync"
	"time"
)

type PointKey struct {
	CommonAddress uint16
	IOA           uint32
}

type LatestCache struct {
	mu     sync.RWMutex
	values map[PointKey]PointValue
}

func NewLatestCache() *LatestCache {
	return &LatestCache{values: map[PointKey]PointValue{}}
}

func (c *LatestCache) Update(value PointValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[PointKey{CommonAddress: value.CommonAddress, IOA: value.IOA}] = value
}

func (c *LatestCache) Snapshot(now time.Time, staleAfter time.Duration) []PointValue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	values := make([]PointValue, 0, len(c.values))
	for _, value := range c.values {
		if staleAfter > 0 && !value.Timestamp.IsZero() && now.Sub(value.Timestamp) > staleAfter {
			value.Stale = true
		}
		values = append(values, value)
	}
	return values
}
