package memory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	saveParam = Station{
		Fast:      1,
		Fastest:   2,
		SafeLow:   3,
		Average:   4,
		BlockTime: 5,
	}
)

func TestStorage_Memory_GetEmpty(t *testing.T) {
	storage := NewStorage()
	_, flag := storage.Get()

	assert.EqualValues(t, false, flag)
}

func TestStorage_Memory_GetValue(t *testing.T) {
	storage := NewStorage()
	storage.Set(saveParam, 1*time.Minute)
	content, _ := storage.Get()
	assert.EqualValues(t, saveParam, *content)
}

func TestStorage_Memory_GetExpiredValue(t *testing.T) {
	storage := NewStorage()
	storage.Set(saveParam, 800*time.Millisecond)
	time.Sleep(1 * time.Second)
	_, flag := storage.Get()

	assert.EqualValues(t, false, flag)
}
