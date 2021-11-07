package memory

import (
	"sync"
	"time"
)

type Station struct {
	Fast      int64
	Fastest   int64
	SafeLow   int64
	Average   int64
	BlockTime float64
}

// Storage mechanism for caching strings in memory
type Storage struct {
	mu  *sync.RWMutex
	st  *Station
	exp time.Time
}

// NewStorage creates a new in memory storage
func NewStorage() *Storage {
	return &Storage{
		st: &Station{},
		mu: &sync.RWMutex{},
	}
}

// Expired returns true if the item has expired.
func (s *Storage) Expired() bool {
	return time.Now().After(s.exp)
}

// Get a cached content by key
func (s Storage) Get() (*Station, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Expired() {
		return nil, false
	}

	return s.st, true
}

// Set a cached content by key
func (s *Storage) Set(station Station, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.st = &station
	s.exp = time.Now().Add(duration)
}

func (s Storage) GetWithoutExpiration() *Station {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.st
}
