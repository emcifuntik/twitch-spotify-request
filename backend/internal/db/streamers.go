package db

import (
	"errors"

	"gorm.io/gorm"
)

// CreateOrUpdateTwitchData creates a new Streamer record if none exists or updates the existing one.
// twitchUserID is expected as a numeric string.
func CreateOrUpdateTwitchData(db *gorm.DB, twitchUserID, twitchUserName, accessToken, refreshToken, spotifyState string) error {
	var streamer Streamer
	result := db.Where("streamer_channel_id = ?", twitchUserID).First(&streamer)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			streamer = Streamer{
				ChannelID:     twitchUserID,
				Name:          twitchUserName,
				TwitchToken:   accessToken,
				TwitchRefresh: refreshToken,
				SpotifyState:  spotifyState,
			}
			return db.Create(&streamer).Error
		}
		return result.Error
	}

	streamer.Name = twitchUserName
	streamer.TwitchToken = accessToken
	streamer.TwitchRefresh = refreshToken
	streamer.SpotifyState = spotifyState

	return db.Save(&streamer).Error
}

// UpdateSpotifyTokensByState finds a streamer by SpotifyState and updates SpotifyToken and SpotifyRefresh.
func UpdateSpotifyTokensByState(db *gorm.DB, state, accessToken, refreshToken string) error {
	var streamer Streamer
	result := db.Where("streamer_spotify_state = ?", state).First(&streamer)
	if result.Error != nil {
		return result.Error
	}

	streamer.SpotifyToken = accessToken
	streamer.SpotifyRefresh = refreshToken

	return db.Save(&streamer).Error
}

// GetStreamerBySpotifyState finds a streamer by their Spotify state
func GetStreamerBySpotifyState(db *gorm.DB, state string) (*Streamer, error) {
	var streamer Streamer
	result := db.Where("streamer_spotify_state = ?", state).First(&streamer)
	if result.Error != nil {
		return nil, result.Error
	}
	return &streamer, nil
}
