package db

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// GetStreamerCommands retrieves all commands for a streamer
func GetStreamerCommands(db *gorm.DB, streamerID uint) ([]Command, error) {
	var commands []Command
	err := db.Where("command_streamer_id = ?", streamerID).Find(&commands).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get commands for streamer %d: %w", streamerID, err)
	}
	return commands, nil
}

// GetCommandByType retrieves a command by type for a streamer
func GetCommandByType(db *gorm.DB, streamerID uint, commandType string) (*Command, error) {
	var command Command
	err := db.Where("command_streamer_id = ? AND command_type = ?", streamerID, commandType).First(&command).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get command %s for streamer %d: %w", commandType, streamerID, err)
	}
	return &command, nil
}

// CreateOrUpdateCommand creates or updates a command for a streamer
func CreateOrUpdateCommand(db *gorm.DB, streamerID uint, commandType, name string, enabled bool) error {
	var command Command
	err := db.Where("command_streamer_id = ? AND command_type = ?", streamerID, commandType).First(&command).Error

	if err != nil {
		// Command doesn't exist, create it
		command = Command{
			StreamerID: streamerID,
			Type:       commandType,
			Name:       name,
			IsEnabled:  enabled,
		}
		err = db.Create(&command).Error
		if err != nil {
			return fmt.Errorf("failed to create command %s for streamer %d: %w", commandType, streamerID, err)
		}
		log.Printf("Created command %s (%s) for streamer %d", commandType, name, streamerID)
	} else {
		// Command exists, update it
		command.Name = name
		command.IsEnabled = enabled
		err = db.Save(&command).Error
		if err != nil {
			return fmt.Errorf("failed to update command %s for streamer %d: %w", commandType, streamerID, err)
		}
		log.Printf("Updated command %s (%s) for streamer %d", commandType, name, streamerID)
	}

	return nil
}

// DeleteCommand deletes a command for a streamer
func DeleteCommand(db *gorm.DB, streamerID uint, commandType string) error {
	err := db.Where("command_streamer_id = ? AND command_type = ?", streamerID, commandType).Delete(&Command{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete command %s for streamer %d: %w", commandType, streamerID, err)
	}
	log.Printf("Deleted command %s for streamer %d", commandType, streamerID)
	return nil
}

// InitializeDefaultCommands creates default commands for a streamer
func InitializeDefaultCommands(db *gorm.DB, streamerID uint) error {
	defaultCommands := []struct {
		Type    string
		Name    string
		Enabled bool
	}{
		{"request", "sr", true},
		{"block", "block", true},
		{"volume", "volume", true},
		{"skip", "skip", true},
		{"queue", "queue", true},
	}

	for _, cmd := range defaultCommands {
		err := CreateOrUpdateCommand(db, streamerID, cmd.Type, cmd.Name, cmd.Enabled)
		if err != nil {
			return fmt.Errorf("failed to initialize default command %s: %w", cmd.Type, err)
		}
	}

	log.Printf("Initialized default commands for streamer %d", streamerID)
	return nil
}

// UpdateStreamerBroadcasterType updates the broadcaster type for a streamer
func UpdateStreamerBroadcasterType(db *gorm.DB, channelID, broadcasterType string) error {
	err := db.Model(&Streamer{}).Where("streamer_channel_id = ?", channelID).Update("broadcaster_type", broadcasterType).Error
	if err != nil {
		return fmt.Errorf("failed to update broadcaster type for streamer %s: %w", channelID, err)
	}
	log.Printf("Updated broadcaster type to %s for streamer %s", broadcasterType, channelID)
	return nil
}

// UpdateStreamerUseCommands updates the use_commands flag for a streamer
func UpdateStreamerUseCommands(db *gorm.DB, channelID string, useCommands bool) error {
	err := db.Model(&Streamer{}).Where("streamer_channel_id = ?", channelID).Update("use_commands", useCommands).Error
	if err != nil {
		return fmt.Errorf("failed to update use_commands for streamer %s: %w", channelID, err)
	}
	log.Printf("Updated use_commands to %v for streamer %s", useCommands, channelID)
	return nil
}

// CanUseRewards checks if a streamer can use channel point rewards
func CanUseRewards(db *gorm.DB, channelID string) (bool, error) {
	var streamer Streamer
	err := db.Where("streamer_channel_id = ?", channelID).First(&streamer).Error
	if err != nil {
		return false, fmt.Errorf("failed to get streamer %s: %w", channelID, err)
	}

	// Only affiliates and partners can use rewards
	return streamer.BroadcasterType == "affiliate" || streamer.BroadcasterType == "partner", nil
}
