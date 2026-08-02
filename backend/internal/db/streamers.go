package db

import (
	"errors"
	"time"

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

	authorizedAt := time.Now().UTC()
	return db.Model(&streamer).Updates(map[string]interface{}{
		"streamer_spotify_token":           accessToken,
		"streamer_spotify_refresh":         refreshToken,
		"streamer_spotify_authorized_at":   authorizedAt,
		"streamer_spotify_reauth_required": false,
	}).Error
}

// UpdateSpotifyTokens stores tokens returned by a successful refresh. Spotify
// may omit a replacement refresh token, in which case the existing one is kept.
func UpdateSpotifyTokens(db *gorm.DB, streamerID uint, accessToken, refreshToken string) error {
	updates := map[string]interface{}{
		"streamer_spotify_token":           accessToken,
		"streamer_spotify_reauth_required": false,
	}
	if refreshToken != "" {
		updates["streamer_spotify_refresh"] = refreshToken
	}

	return db.Model(&Streamer{}).Where("streamer_id = ?", streamerID).Updates(updates).Error
}

// InvalidateSpotifyTokens discards an unusable token pair after Spotify returns
// invalid_grant. The authorization timestamp is retained for diagnostics.
func InvalidateSpotifyTokens(db *gorm.DB, streamerID uint) error {
	return db.Model(&Streamer{}).Where("streamer_id = ?", streamerID).Updates(map[string]interface{}{
		"streamer_spotify_token":           "",
		"streamer_spotify_refresh":         "",
		"streamer_spotify_reauth_required": true,
	}).Error
}

// UpdateSpotifyState associates a new Spotify OAuth attempt with a logged-in streamer.
func UpdateSpotifyState(db *gorm.DB, channelID, state string) error {
	result := db.Model(&Streamer{}).
		Where("streamer_channel_id = ?", channelID).
		Update("streamer_spotify_state", state)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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
