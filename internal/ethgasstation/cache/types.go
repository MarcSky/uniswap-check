package cache

import (
	"time"
	"uniswap-bot/internal/ethgasstation/cache/memory"
)

// Storage Cache Interface
type Storage interface {
	Get() (*memory.Station, bool)
	GetWithoutExpiration() *memory.Station
	Set(station memory.Station, duration time.Duration)
}
