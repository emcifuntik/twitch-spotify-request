package service

import (
	"errors"

	"github.com/emcifuntik/twitch-spotify-request/internal/db"

	"gorm.io/gorm"
)

// defaultConfig holds the default key/value pairs.
var defaultConfig = map[string]string{
	"max_track_length":   "-1",
	"same_song_cooldown": "3600",
}

type ConfigStoreAccessor struct {
	db *gorm.DB
}

func NewConfigStoreAccessor(db *gorm.DB) *ConfigStoreAccessor {
	return &ConfigStoreAccessor{db: db}
}

func (csa *ConfigStoreAccessor) Get(streamerID uint, key string) (string, error) {
	var cs db.ConfigStore
	if err := csa.db.Where("cs_streamer_id = ? AND cs_key = ?", streamerID, key).First(&cs).Error; err == nil {
		return cs.Value, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	// Return default value if present, otherwise return an error.
	if val, ok := defaultConfig[key]; ok {
		return val, nil
	}
	return "", errors.New("no configuration found for key")
}

func (csa *ConfigStoreAccessor) Set(streamerID uint, key string, value string) error {
	var cs db.ConfigStore
	err := csa.db.Where("cs_streamer_id = ? AND cs_key = ?", streamerID, key).First(&cs).Error
	if err == nil {
		cs.Value = value
		return csa.db.Save(&cs).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		cs = db.ConfigStore{
			StreamerID: streamerID,
			Key:        key,
			Value:      value,
		}
		return csa.db.Create(&cs).Error
	}
	return err
}
