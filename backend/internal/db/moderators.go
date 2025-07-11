package db

import (
	"gorm.io/gorm"
)

// GetModerators retrieves all moderators for a streamer
func GetModerators(db *gorm.DB, streamerID uint) ([]Moderator, error) {
	var moderators []Moderator
	err := db.Where("moderator_streamer_id = ?", streamerID).Find(&moderators).Error
	return moderators, err
}

// AddModerator adds a new moderator for a streamer
func AddModerator(db *gorm.DB, streamerID uint, twitchID, twitchName, avatar string) error {
	// Check if moderator already exists
	var existing Moderator
	err := db.Where("moderator_streamer_id = ? AND moderator_twitch_id = ?", streamerID, twitchID).First(&existing).Error
	if err == nil {
		// Moderator already exists, update their info
		existing.TwitchName = twitchName
		existing.Avatar = avatar
		return db.Save(&existing).Error
	}

	if err != gorm.ErrRecordNotFound {
		return err
	}

	// Create new moderator
	moderator := Moderator{
		StreamerID: streamerID,
		TwitchID:   twitchID,
		TwitchName: twitchName,
		Avatar:     avatar,
	}

	return db.Create(&moderator).Error
}

// RemoveModerator removes a moderator for a streamer
func RemoveModerator(db *gorm.DB, streamerID uint, moderatorID uint) error {
	return db.Where("moderator_streamer_id = ? AND moderator_id = ?", streamerID, moderatorID).Delete(&Moderator{}).Error
}

// IsBotModerator checks if a user is a bot moderator for a streamer
func IsBotModerator(db *gorm.DB, streamerID uint, twitchID string) bool {
	var moderator Moderator
	err := db.Where("moderator_streamer_id = ? AND moderator_twitch_id = ?", streamerID, twitchID).First(&moderator).Error
	return err == nil
}

// IsBotModeratorByName checks if a user is a bot moderator by their Twitch name
func IsBotModeratorByName(db *gorm.DB, streamerID uint, twitchName string) bool {
	var moderator Moderator
	err := db.Where("moderator_streamer_id = ? AND moderator_twitch_name = ?", streamerID, twitchName).First(&moderator).Error
	return err == nil
}
