package db

import (
	"strconv"

	"gorm.io/gorm"
)

// ConfigKeys for various settings
const (
	ConfigKeyMaxSongLength    = "max_song_length"
	ConfigKeyCooldownSameSong = "cooldown_same_song"
	ConfigKeyWebUIEnabled     = "web_ui_enabled"
)

// GetConfig retrieves a configuration value for a streamer
func GetConfig(db *gorm.DB, streamerID uint, key string) (string, error) {
	var config ConfigStore
	err := db.Where("cs_streamer_id = ? AND cs_key = ?", streamerID, key).First(&config).Error
	if err != nil {
		return "", err
	}
	return config.Value, nil
}

// SetConfig sets a configuration value for a streamer
func SetConfig(db *gorm.DB, streamerID uint, key, value string) error {
	var config ConfigStore
	err := db.Where("cs_streamer_id = ? AND cs_key = ?", streamerID, key).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		// Create new config entry
		config = ConfigStore{
			StreamerID: streamerID,
			Key:        key,
			Value:      value,
		}
		return db.Create(&config).Error
	} else if err != nil {
		return err
	}

	// Update existing config entry
	config.Value = value
	return db.Save(&config).Error
}

// GetConfigInt retrieves a configuration value as integer
func GetConfigInt(db *gorm.DB, streamerID uint, key string, defaultValue int) int {
	value, err := GetConfig(db, streamerID, key)
	if err != nil {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// SetConfigInt sets a configuration value as integer
func SetConfigInt(db *gorm.DB, streamerID uint, key string, value int) error {
	return SetConfig(db, streamerID, key, strconv.Itoa(value))
}

// GetConfigBool retrieves a configuration value as boolean
func GetConfigBool(db *gorm.DB, streamerID uint, key string, defaultValue bool) bool {
	value, err := GetConfig(db, streamerID, key)
	if err != nil {
		return defaultValue
	}

	return value == "true"
}

// SetConfigBool sets a configuration value as boolean
func SetConfigBool(db *gorm.DB, streamerID uint, key string, value bool) error {
	strValue := "false"
	if value {
		strValue = "true"
	}
	return SetConfig(db, streamerID, key, strValue)
}

// GetMaxSongLength returns the maximum song length in seconds (default: 10 minutes)
func GetMaxSongLength(db *gorm.DB, streamerID uint) int {
	return GetConfigInt(db, streamerID, ConfigKeyMaxSongLength, 600) // 10 minutes default
}

// GetCooldownSameSong returns the cooldown for the same song in seconds (default: 1 hour)
func GetCooldownSameSong(db *gorm.DB, streamerID uint) int {
	return GetConfigInt(db, streamerID, ConfigKeyCooldownSameSong, 3600) // 1 hour default
}

// IsWebUIEnabled returns whether the web UI is enabled for a streamer
func IsWebUIEnabled(db *gorm.DB, streamerID uint) bool {
	return GetConfigBool(db, streamerID, ConfigKeyWebUIEnabled, true) // enabled by default
}
