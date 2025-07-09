package db

import (
	"gorm.io/gorm"
)

// BlockType represents the type of block
type BlockType string

const (
	BlockTypeArtist BlockType = "artist"
	BlockTypeTrack  BlockType = "track"
)

// AddBlock adds a new block for a streamer using Spotify ID
func AddBlock(db *gorm.DB, streamerID uint, blockType BlockType, spotifyID, name string) error {
	// Check if block already exists
	var existingBlock Block
	err := db.Where("block_streamer_id = ? AND block_spotify_id = ?", streamerID, spotifyID).First(&existingBlock).Error
	if err == nil {
		// Block already exists
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return err
	}

	// Create new block
	block := Block{
		StreamerID: streamerID,
		SpotifyID:  spotifyID,
		Type:       string(blockType),
		Name:       name,
	}

	return db.Create(&block).Error
}

// RemoveBlock removes a block for a streamer by block ID
func RemoveBlockByID(db *gorm.DB, streamerID uint, blockID uint) error {
	return db.Where("block_id = ? AND block_streamer_id = ?", blockID, streamerID).Delete(&Block{}).Error
}

// RemoveBlock removes a block for a streamer by Spotify ID
func RemoveBlock(db *gorm.DB, streamerID uint, spotifyID string) error {
	return db.Where("block_streamer_id = ? AND block_spotify_id = ?", streamerID, spotifyID).Delete(&Block{}).Error
}

// IsBlocked checks if an artist or track is blocked for a streamer using Spotify IDs
func IsBlocked(db *gorm.DB, streamerID uint, artistIDs []string, trackID string) bool {
	var count int64

	// Check if track is directly blocked
	if trackID != "" {
		db.Model(&Block{}).Where("block_streamer_id = ? AND block_spotify_id = ? AND block_type = ?",
			streamerID, trackID, "track").Count(&count)
		if count > 0 {
			return true
		}
	}

	// Check if any of the artists are blocked
	for _, artistID := range artistIDs {
		db.Model(&Block{}).Where("block_streamer_id = ? AND block_spotify_id = ? AND block_type = ?",
			streamerID, artistID, "artist").Count(&count)
		if count > 0 {
			return true
		}
	}

	return false
}

// GetBlocks returns all blocks for a streamer
func GetBlocks(db *gorm.DB, streamerID uint) ([]Block, error) {
	var blocks []Block
	err := db.Where("block_streamer_id = ?", streamerID).Find(&blocks).Error
	return blocks, err
}

// BlockInfo represents block information for API responses
type BlockInfo struct {
	ID        uint   `json:"id"`
	SpotifyID string `json:"spotify_id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "artist" or "track"
}

// GetBlocksInfo returns formatted block information for API responses
func GetBlocksInfo(db *gorm.DB, streamerID uint) ([]BlockInfo, error) {
	blocks, err := GetBlocks(db, streamerID)
	if err != nil {
		return nil, err
	}

	var blocksInfo []BlockInfo
	for _, block := range blocks {
		blocksInfo = append(blocksInfo, BlockInfo{
			ID:        block.ID,
			SpotifyID: block.SpotifyID,
			Name:      block.Name,
			Type:      block.Type,
		})
	}

	return blocksInfo, nil
}
