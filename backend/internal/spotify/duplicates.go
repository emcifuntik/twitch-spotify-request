package spotify

import (
	"sync"
	"time"
)

// DuplicateStore manages recently played songs to prevent duplicates
type DuplicateStore struct {
	holdTime time.Duration
	store    map[string]time.Time
	mutex    sync.RWMutex
}

// NewDuplicateStore creates a new duplicate store with 1 hour hold time
func NewDuplicateStore() *DuplicateStore {
	return &DuplicateStore{
		holdTime: time.Hour, // 1 hour hold time like in TypeScript
		store:    make(map[string]time.Time),
	}
}

// Add adds a song ID to the store with current timestamp
func (ds *DuplicateStore) Add(id string) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.store[id] = time.Now()
}

// Exists checks if a song was played recently (within hold time)
func (ds *DuplicateStore) Exists(id string) bool {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	lastPlayTime, exists := ds.store[id]
	if !exists {
		return false
	}

	timeSinceLastPlay := time.Since(lastPlayTime)
	if timeSinceLastPlay < ds.holdTime {
		return true
	}

	// Clean up expired entry
	delete(ds.store, id)
	return false
}

// Cleanup removes all expired entries from the store
func (ds *DuplicateStore) Cleanup() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	now := time.Now()
	for id, playTime := range ds.store {
		if now.Sub(playTime) >= ds.holdTime {
			delete(ds.store, id)
		}
	}
}

// Global duplicate store instance
var GlobalDuplicateStore = NewDuplicateStore()
