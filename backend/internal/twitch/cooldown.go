package twitch

import (
	"sync"
	"time"
)

// CooldownManager manages cooldowns for songs per streamer
type CooldownManager struct {
	cooldowns map[string]map[string]time.Time // streamerID -> trackURI -> last played time
	mutex     sync.RWMutex
}

var globalCooldownManager = &CooldownManager{
	cooldowns: make(map[string]map[string]time.Time),
}

// GetCooldownManager returns the global cooldown manager
func GetCooldownManager() *CooldownManager {
	return globalCooldownManager
}

// AddCooldown adds a cooldown for a track for a specific streamer
func (cm *CooldownManager) AddCooldown(streamerID, trackURI string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.cooldowns[streamerID] == nil {
		cm.cooldowns[streamerID] = make(map[string]time.Time)
	}

	cm.cooldowns[streamerID][trackURI] = time.Now()
}

// IsOnCooldown checks if a track is on cooldown for a specific streamer
func (cm *CooldownManager) IsOnCooldown(streamerID, trackURI string, cooldownSeconds int) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	streamerCooldowns, exists := cm.cooldowns[streamerID]
	if !exists {
		return false
	}

	lastPlayed, exists := streamerCooldowns[trackURI]
	if !exists {
		return false
	}

	return time.Since(lastPlayed).Seconds() < float64(cooldownSeconds)
}

// GetRemainingCooldown returns the remaining cooldown time in seconds
func (cm *CooldownManager) GetRemainingCooldown(streamerID, trackURI string, cooldownSeconds int) int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	streamerCooldowns, exists := cm.cooldowns[streamerID]
	if !exists {
		return 0
	}

	lastPlayed, exists := streamerCooldowns[trackURI]
	if !exists {
		return 0
	}

	elapsed := time.Since(lastPlayed).Seconds()
	remaining := float64(cooldownSeconds) - elapsed

	if remaining <= 0 {
		return 0
	}

	return int(remaining)
}

// CleanupExpiredCooldowns removes expired cooldowns to prevent memory leaks
func (cm *CooldownManager) CleanupExpiredCooldowns() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	maxCooldown := 24 * time.Hour // Remove cooldowns older than 24 hours

	for streamerID, streamerCooldowns := range cm.cooldowns {
		for trackURI, lastPlayed := range streamerCooldowns {
			if now.Sub(lastPlayed) > maxCooldown {
				delete(streamerCooldowns, trackURI)
			}
		}

		// If no cooldowns left for this streamer, remove the map
		if len(streamerCooldowns) == 0 {
			delete(cm.cooldowns, streamerID)
		}
	}
}

// StartPeriodicCleanup starts a goroutine that periodically cleans up expired cooldowns
func (cm *CooldownManager) StartPeriodicCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			cm.CleanupExpiredCooldowns()
		}
	}()
}
