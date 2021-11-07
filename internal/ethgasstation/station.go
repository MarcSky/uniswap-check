package ethgasstation

type Station struct {
	Fast      int64   `json:"fast"`
	Fastest   int64   `json:"fastest"`
	SafeLow   int64   `json:"safeLow"`
	Average   int64   `json:"average"`
	BlockTime float64 `json:"block_time"`
}
