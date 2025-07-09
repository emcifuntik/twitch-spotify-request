package db

import (
	"time"
)

// Streamer represents the streamer table.
type Streamer struct {
	ID             uint          `gorm:"primaryKey;autoIncrement;column:streamer_id"`
	ChannelID      string        `gorm:"column:streamer_channel_id;size:64;unique;not null"`
	Name           string        `gorm:"column:streamer_name;size:64;not null"`
	TwitchToken    string        `gorm:"column:streamer_twitch_token;size:256"`
	TwitchRefresh  string        `gorm:"column:streamer_twitch_refresh;size:256"`
	SpotifyToken   string        `gorm:"column:streamer_spotify_token;size:256"`
	SpotifyRefresh string        `gorm:"column:streamer_spotify_refresh;size:256"`
	SpotifyState   string        `gorm:"column:streamer_spotify_state;size:64;unique"`
	Rewards        []Reward      `gorm:"foreignKey:StreamerID"`
	Blocks         []Block       `gorm:"foreignKey:StreamerID"`
	ConfigStore    []ConfigStore `gorm:"foreignKey:StreamerID"`
	Requests       []Request     `gorm:"foreignKey:StreamerID"`
}

// Reward represents the rewards table.
type Reward struct {
	ID         uint   `gorm:"primaryKey;autoIncrement;column:reward_id"`
	StreamerID uint   `gorm:"column:reward_streamer;not null"`
	InternalID int8   `gorm:"column:reward_internal_id;not null"`
	TwitchID   string `gorm:"column:reward_twitch_id;size:128"`
	// Optional: Association with Streamer if needed
	Streamer Streamer `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

// Block represents the blocks table.
type Block struct {
	ID         uint   `gorm:"primaryKey;autoIncrement;column:block_id"`
	StreamerID uint   `gorm:"column:block_streamer_id;not null;index"`
	SpotifyID  string `gorm:"column:block_spotify_id;size:128;not null"`
	Type       string `gorm:"column:block_type;size:16;not null"`  // "artist" or "track"
	Name       string `gorm:"column:block_name;size:256;not null"` // Display name for UI
}

// ConfigStore represents the config_store table.
type ConfigStore struct {
	ID         uint   `gorm:"primaryKey;autoIncrement;column:cs_id"`
	StreamerID uint   `gorm:"column:cs_streamer_id;not null;index"`
	Key        string `gorm:"column:cs_key;size:128;not null"`
	Value      string `gorm:"column:cs_value;size:128;not null"`
}

// User represents the users table.
type User struct {
	ID         uint      `gorm:"primaryKey;autoIncrement;column:user_id"`
	TwitchID   string    `gorm:"column:user_twitch_id;size:64;unique;not null"`
	TwitchName string    `gorm:"column:user_twitch_name;size:64"`
	Requests   []Request `gorm:"foreignKey:UserID"`
}

// Request represents the requests table.
type Request struct {
	ID           uint      `gorm:"primaryKey;autoIncrement;column:request_id"`
	StreamerID   uint      `gorm:"column:request_streamer_id;not null;index"`
	UserID       uint      `gorm:"column:request_user_id;not null;index"`
	SearchPrompt string    `gorm:"column:request_search_prompt;type:text"`
	TrackID      string    `gorm:"column:request_track_id;size:256"`
	RequestTime  time.Time `gorm:"column:request_time;autoCreateTime"`
	// Optional: Associations
	Streamer Streamer `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	User     User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}
