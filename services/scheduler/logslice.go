package scheduler

import (
	"strings"
	"sync"
)

type LogSlice struct {
	lock  sync.RWMutex
	array []string
}

func NewLogSlice() *LogSlice {
	return &LogSlice{
		lock:  sync.RWMutex{},
		array: []string{},
	}
}

func (c *LogSlice) Append(items ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.array = append(c.array, items...)
}

func (c *LogSlice) Join(sep string) string {
	return strings.Join(c.array, sep)
}
