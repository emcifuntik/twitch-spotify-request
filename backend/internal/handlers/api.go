package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/emcifuntik/twitch-spotify-request/internal/db"
	"github.com/emcifuntik/twitch-spotify-request/internal/twitch"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// UserProfileResponse represents user profile data
type UserProfileResponse struct {
	ID                uint   `json:"id"`
	ChannelID         string `json:"channel_id"`
	Name              string `json:"name"`
	HasSpotifyLinked  bool   `json:"has_spotify_linked"`
	HasTwitchLinked   bool   `json:"has_twitch_linked"`
	RewardsConfigured bool   `json:"rewards_configured"`
	BroadcasterType   string `json:"broadcaster_type"`
	UseCommands       bool   `json:"use_commands"`
	CanUseRewards     bool   `json:"can_use_rewards"`
}

// QueueResponse represents queue data for API
type QueueResponse struct {
	CurrentSong        string       `json:"current_song"`
	CurrentSongImage   string       `json:"current_song_image,omitempty"`
	CurrentSongArtists []string     `json:"current_song_artists,omitempty"`
	Progress           int          `json:"progress"`
	Duration           int          `json:"duration"`
	Queue              []QueueTrack `json:"queue"`
	Timestamp          int64        `json:"timestamp"`
}

// QueueTrack represents a track in the queue
type QueueTrack struct {
	Name     string   `json:"name"`
	Artists  []string `json:"artists"`
	Duration int      `json:"duration"`
	URI      string   `json:"uri"`
	Image    string   `json:"image,omitempty"`
}

// SettingsRequest represents a settings update request
type SettingsRequest struct {
	MaxSongLength    *int  `json:"max_song_length,omitempty"`
	CooldownSameSong *int  `json:"cooldown_same_song,omitempty"`
	WebUIEnabled     *bool `json:"web_ui_enabled,omitempty"`
}

// SettingsResponse represents current settings
type SettingsResponse struct {
	MaxSongLength    int  `json:"max_song_length"`
	CooldownSameSong int  `json:"cooldown_same_song"`
	WebUIEnabled     bool `json:"web_ui_enabled"`
}

// BlockRequest represents a block add/remove request
type BlockRequest struct {
	SpotifyID string `json:"spotify_id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "artist" or "track"
}

// SpotifySearchRequest represents a Spotify search request
type SpotifySearchRequest struct {
	Query string `json:"query"`
	Type  string `json:"type"` // "artist" or "track"
	Limit int    `json:"limit,omitempty"`
}

// SpotifySearchResult represents a single search result
type SpotifySearchResult struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Image   string   `json:"image,omitempty"`
	Artists []string `json:"artists,omitempty"` // For tracks
}

// SpotifySearchResponse represents the response from Spotify search
type SpotifySearchResponse struct {
	Results []SpotifySearchResult `json:"results"`
}

// ModeratorResponse represents a moderator for API responses
type ModeratorResponse struct {
	ID         uint   `json:"id"`
	TwitchID   string `json:"twitch_id"`
	TwitchName string `json:"twitch_name"`
	Avatar     string `json:"avatar"`
	AddedAt    string `json:"added_at"`
}

// AddModeratorRequest represents the request to add a moderator
type AddModeratorRequest struct {
	TwitchName string `json:"twitch_name"`
}

// TwitchUserSearchResult represents a Twitch user search result
type TwitchUserSearchResult struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`
}

// CommandResponse represents a command for API responses
type CommandResponse struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	IsEnabled bool   `json:"is_enabled"`
}

// CommandRequest represents a command update request
type CommandRequest struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	IsEnabled bool   `json:"is_enabled"`
}

// RequestModeToggleRequest represents a request to toggle between commands and rewards
type RequestModeToggleRequest struct {
	UseCommands bool `json:"use_commands"`
}

// GetUserProfile returns the user's profile information
func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	// This is a simplified version - in production you'd get user ID from JWT/session
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get user from database
	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if rewards are actually configured by checking with the reward listener
	rewardsConfigured := false
	rewardListener := twitch.GetRewardListener(streamer.ChannelID)
	if rewardListener != nil {
		rewardsConfigured = rewardListener.CheckRewardsConfigured()
	}

	// Check if user can use rewards based on broadcaster type
	canUseRewards, err := db.CanUseRewards(database, streamer.ChannelID)
	if err != nil {
		log.Printf("Error checking reward eligibility: %v", err)
		canUseRewards = false
	}

	profile := UserProfileResponse{
		ID:                streamer.ID,
		ChannelID:         streamer.ChannelID,
		Name:              streamer.Name,
		HasSpotifyLinked:  streamer.SpotifyToken != "",
		HasTwitchLinked:   streamer.TwitchToken != "",
		RewardsConfigured: rewardsConfigured,
		BroadcasterType:   streamer.BroadcasterType,
		UseCommands:       streamer.UseCommands,
		CanUseRewards:     canUseRewards,
	}

	writeAPISuccess(w, profile)
}

// GetQueue returns the current queue for a user
func GetQueue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get the reward listener for this user
	rewardListener := twitch.GetRewardListener(userID)
	if rewardListener == nil {
		writeAPIError(w, "User not found or not active", http.StatusNotFound)
		return
	}

	// Get queue data
	queueData, err := rewardListener.GetQueueData()
	if err != nil {
		log.Printf("Error getting queue data: %v", err)
		writeAPIError(w, "Failed to get queue data", http.StatusInternalServerError)
		return
	}

	// Convert to API format
	var tracks []QueueTrack
	for _, track := range queueData.Queue {
		var artists []string
		for _, artist := range track.Artists {
			artists = append(artists, artist.Name)
		}

		// Get the largest image URL
		var imageURL string
		if len(track.Album.Images) > 0 {
			imageURL = track.Album.Images[0].URL
		}

		tracks = append(tracks, QueueTrack{
			Name:     track.Name,
			Artists:  artists,
			Duration: int(track.Duration),
			URI:      string(track.URI),
			Image:    imageURL,
		})
	}

	// Get current track image and artists
	var currentSongImage string
	var currentSongArtists []string
	if queueData.CurrentTrack != nil {
		if len(queueData.CurrentTrack.Album.Images) > 0 {
			currentSongImage = queueData.CurrentTrack.Album.Images[0].URL
		}
		for _, artist := range queueData.CurrentTrack.Artists {
			currentSongArtists = append(currentSongArtists, artist.Name)
		}
	}
	var currentSongName string
	if queueData.CurrentTrack != nil {
		currentSongName = queueData.CurrentTrack.Name
	}

	response := QueueResponse{
		CurrentSong:        currentSongName,
		CurrentSongImage:   currentSongImage,
		CurrentSongArtists: currentSongArtists,
		Progress:           queueData.Progress,
		Duration:           queueData.Duration,
		Queue:              tracks,
		Timestamp:          queueData.LastUpdated,
	}

	writeAPISuccess(w, response)
}

// GetPublicQueue returns the queue for public viewing (no auth required)
func GetPublicQueue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamerID := vars["streamerID"]

	if streamerID == "" {
		writeAPIError(w, "Streamer ID is required", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamer db.Streamer
	var result *gorm.DB

	// Try to parse as integer first (streamer_id)
	if streamerIDInt, err := strconv.Atoi(streamerID); err == nil {
		result = database.Where("streamer_id = ?", streamerIDInt).First(&streamer)
	} else {
		// Try as string - could be channel_id or name
		result = database.Where("streamer_channel_id = ? OR streamer_name = ?", streamerID, streamerID).First(&streamer)
	}

	if result.Error != nil {
		writeAPIError(w, "Streamer not found", http.StatusNotFound)
		return
	}

	// Get the reward listener for this streamer
	rl := twitch.GetRewardListener(streamer.ChannelID)
	if rl == nil {
		writeAPIError(w, "Streamer not active", http.StatusNotFound)
		return
	}

	// Get queue data
	queueData, err := rl.GetQueueData()
	if err != nil {
		writeAPIError(w, "Failed to get queue data", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var tracks []QueueTrack
	for _, track := range queueData.Queue {
		var artists []string
		for _, artist := range track.Artists {
			artists = append(artists, artist.Name)
		}

		// Get the largest image URL
		var imageURL string
		if len(track.Album.Images) > 0 {
			imageURL = track.Album.Images[0].URL
		}

		tracks = append(tracks, QueueTrack{
			Name:     track.Name,
			Artists:  artists,
			Duration: int(track.Duration),
			URI:      string(track.URI),
			Image:    imageURL,
		})
	}

	// Get current track image and artists
	var currentSongImage string
	var currentSongArtists []string
	if queueData.CurrentTrack != nil {
		if len(queueData.CurrentTrack.Album.Images) > 0 {
			currentSongImage = queueData.CurrentTrack.Album.Images[0].URL
		}
		for _, artist := range queueData.CurrentTrack.Artists {
			currentSongArtists = append(currentSongArtists, artist.Name)
		}
	}
	var currentSongName string
	if queueData.CurrentTrack != nil {
		currentSongName = queueData.CurrentTrack.Name
	}

	response := QueueResponse{
		CurrentSong:        currentSongName,
		CurrentSongImage:   currentSongImage,
		CurrentSongArtists: currentSongArtists,
		Progress:           queueData.Progress,
		Duration:           queueData.Duration,
		Queue:              tracks,
		Timestamp:          queueData.LastUpdated,
	}

	writeAPISuccess(w, response)
}

// UpdateUserSettings updates user settings
func UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeAPIError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Implement settings update logic
	// This would involve updating database records for user preferences

	writeAPISuccess(w, map[string]string{"message": "Settings updated successfully"})
}

// GetStreamers returns a list of all active streamers for the landing page
func GetStreamers(w http.ResponseWriter, r *http.Request) {
	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamers []db.Streamer
	result := database.Where("streamer_spotify_token != '' AND streamer_spotify_token IS NOT NULL AND streamer_twitch_token != '' AND streamer_twitch_token IS NOT NULL").Find(&streamers)
	if result.Error != nil {
		log.Printf("Error getting streamers: %v", result.Error)
		writeAPIError(w, "Failed to get streamers", http.StatusInternalServerError)
		return
	}

	log.Printf("Found %d streamers", len(streamers)) // Debug log

	// Convert to public format (don't expose tokens)
	var publicStreamers []map[string]interface{}
	for _, streamer := range streamers {
		publicStreamers = append(publicStreamers, map[string]interface{}{
			"id":   streamer.ID,
			"name": streamer.Name,
		})
	}

	writeAPISuccess(w, publicStreamers)
}

// Debug endpoint to test API functionality
func DebugAPI(w http.ResponseWriter, r *http.Request) {
	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamers []db.Streamer
	result := database.Find(&streamers)
	if result.Error != nil {
		writeAPIError(w, "Database query failed", http.StatusInternalServerError)
		return
	}

	debugInfo := map[string]interface{}{
		"total_streamers": len(streamers),
		"database_status": "connected",
		"api_status":      "working",
	}

	writeAPISuccess(w, debugInfo)
}

// GetCurrentUser returns basic info about the currently authenticated user
// GetCurrentUser returns the current authenticated user's information
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := GetClaimsFromContext(r)
	if !ok {
		writeAPIError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Get streamer by channel ID
	var streamer db.Streamer
	if err := database.Where("streamer_channel_id = ?", claims.ChannelID).First(&streamer).Error; err != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	profile := UserProfileResponse{
		ID:                streamer.ID,
		ChannelID:         streamer.ChannelID,
		Name:              streamer.Name,
		HasSpotifyLinked:  streamer.SpotifyToken != "",
		HasTwitchLinked:   streamer.TwitchToken != "",
		RewardsConfigured: true, // You might want to add more logic here
	}

	writeAPIResponse(w, profile)
}

// GetSettings returns the current settings for a user
func GetSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Get streamer by channel ID (userID)
	var streamer db.Streamer
	if err := database.Where("streamer_channel_id = ?", userID).First(&streamer).Error; err != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	settings := SettingsResponse{
		MaxSongLength:    db.GetMaxSongLength(database, streamer.ID),
		CooldownSameSong: db.GetCooldownSameSong(database, streamer.ID),
		WebUIEnabled:     db.IsWebUIEnabled(database, streamer.ID),
	}

	writeAPIResponse(w, settings)
}

// UpdateSettings updates settings for a user
func UpdateSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	var req SettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Get streamer by channel ID (userID)
	var streamer db.Streamer
	if err := database.Where("streamer_channel_id = ?", userID).First(&streamer).Error; err != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Update settings if provided
	if req.MaxSongLength != nil {
		if err := db.SetConfigInt(database, streamer.ID, db.ConfigKeyMaxSongLength, *req.MaxSongLength); err != nil {
			writeAPIError(w, "Failed to update max song length", http.StatusInternalServerError)
			return
		}
	}

	if req.CooldownSameSong != nil {
		if err := db.SetConfigInt(database, streamer.ID, db.ConfigKeyCooldownSameSong, *req.CooldownSameSong); err != nil {
			writeAPIError(w, "Failed to update cooldown", http.StatusInternalServerError)
			return
		}
	}

	if req.WebUIEnabled != nil {
		if err := db.SetConfigBool(database, streamer.ID, db.ConfigKeyWebUIEnabled, *req.WebUIEnabled); err != nil {
			writeAPIError(w, "Failed to update web UI setting", http.StatusInternalServerError)
			return
		}
	}

	// Return updated settings
	settings := SettingsResponse{
		MaxSongLength:    db.GetMaxSongLength(database, streamer.ID),
		CooldownSameSong: db.GetCooldownSameSong(database, streamer.ID),
		WebUIEnabled:     db.IsWebUIEnabled(database, streamer.ID),
	}

	writeAPIResponse(w, settings)
}

// GetBlocks returns the blocklist for a user
func GetBlocks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Get streamer by channel ID (userID)
	var streamer db.Streamer
	if err := database.Where("streamer_channel_id = ?", userID).First(&streamer).Error; err != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	blocks, err := db.GetBlocksInfo(database, streamer.ID)
	if err != nil {
		writeAPIError(w, "Failed to get blocks", http.StatusInternalServerError)
		return
	}

	writeAPIResponse(w, blocks)
}

// AddBlock adds a new block for a user
func AddBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	var req BlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.SpotifyID == "" || req.Name == "" {
		writeAPIError(w, "Spotify ID and name are required", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Get streamer by channel ID (userID)
	var streamer db.Streamer
	if err := database.Where("streamer_channel_id = ?", userID).First(&streamer).Error; err != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Add block
	blockType := db.BlockTypeTrack
	if req.Type == "artist" {
		blockType = db.BlockTypeArtist
	}

	if err := db.AddBlock(database, streamer.ID, blockType, req.SpotifyID, req.Name); err != nil {
		writeAPIError(w, "Failed to add block", http.StatusInternalServerError)
		return
	}

	writeAPIResponse(w, map[string]string{"message": "Block added successfully"})
}

// RemoveBlock removes a block for a user
func RemoveBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]
	blockIDStr := vars["blockID"]

	blockID, err := strconv.ParseUint(blockIDStr, 10, 32)
	if err != nil {
		writeAPIError(w, "Invalid block ID", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Get streamer by channel ID (userID)
	var streamer db.Streamer
	if err := database.Where("streamer_channel_id = ?", userID).First(&streamer).Error; err != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Remove block (ensure it belongs to this streamer)
	result := database.Where("block_id = ? AND block_streamer_id = ?", uint(blockID), streamer.ID).Delete(&db.Block{})
	if result.Error != nil {
		writeAPIError(w, "Failed to remove block", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		writeAPIError(w, "Block not found", http.StatusNotFound)
		return
	}

	writeAPIResponse(w, map[string]string{"message": "Block removed successfully"})
}

// SpotifySearch searches Spotify for artists or tracks for autocomplete
func SpotifySearch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	query := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	if query == "" {
		writeAPIError(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	if searchType == "" {
		searchType = "artist,track"
	}

	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Get streamer by channel ID (userID)
	var streamer db.Streamer
	if err := database.Where("streamer_channel_id = ?", userID).First(&streamer).Error; err != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	if streamer.SpotifyToken == "" {
		writeAPIError(w, "Spotify not connected", http.StatusUnauthorized)
		return
	}

	// Make request to Spotify Web API
	spotifyURL := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=%s&limit=%d",
		url.QueryEscape(query), searchType, limit)

	req, err := http.NewRequest("GET", spotifyURL, nil)
	if err != nil {
		writeAPIError(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", "Bearer "+streamer.SpotifyToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		writeAPIError(w, "Failed to search Spotify", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// Token might be expired, try to refresh
		// TODO: Implement token refresh logic here
		writeAPIError(w, "Spotify token expired", http.StatusUnauthorized)
		return
	}

	if resp.StatusCode != 200 {
		writeAPIError(w, "Spotify API error", http.StatusInternalServerError)
		return
	}

	var spotifyResp struct {
		Artists struct {
			Items []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Images []struct {
					URL string `json:"url"`
				} `json:"images"`
			} `json:"items"`
		} `json:"artists"`
		Tracks struct {
			Items []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Images []struct {
						URL string `json:"url"`
					} `json:"images"`
				} `json:"album"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&spotifyResp); err != nil {
		writeAPIError(w, "Failed to parse Spotify response", http.StatusInternalServerError)
		return
	}

	var results []SpotifySearchResult

	// Process artists
	for _, artist := range spotifyResp.Artists.Items {
		result := SpotifySearchResult{
			ID:   artist.ID,
			Name: artist.Name,
			Type: "artist",
		}
		if len(artist.Images) > 0 {
			result.Image = artist.Images[0].URL
		}
		results = append(results, result)
	}

	// Process tracks
	for _, track := range spotifyResp.Tracks.Items {
		result := SpotifySearchResult{
			ID:   track.ID,
			Name: track.Name,
			Type: "track",
		}

		// Add artist names
		for _, artist := range track.Artists {
			result.Artists = append(result.Artists, artist.Name)
		}

		// Add album image
		if len(track.Album.Images) > 0 {
			result.Image = track.Album.Images[0].URL
		}

		results = append(results, result)
	}

	response := SpotifySearchResponse{
		Results: results,
	}

	writeAPIResponse(w, response)
}

// FixRewards attempts to fix/recreate rewards for a user
func FixRewards(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get user from database to verify they exist
	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Try to fix rewards
	if err := twitch.FixRewardsForChannel(userID); err != nil {
		log.Printf("Error fixing rewards for user %s: %v", userID, err)
		writeAPIError(w, fmt.Sprintf("Failed to fix rewards: %v", err), http.StatusInternalServerError)
		return
	}

	writeAPISuccess(w, map[string]string{
		"message": "Rewards have been fixed successfully",
	})
}

// GetModerators returns the moderators for a user
func GetModerators(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get user from database
	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Get moderators
	moderators, err := db.GetModerators(database, streamer.ID)
	if err != nil {
		writeAPIError(w, "Failed to get moderators", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var moderatorResponses []ModeratorResponse
	for _, mod := range moderators {
		moderatorResponses = append(moderatorResponses, ModeratorResponse{
			ID:         mod.ID,
			TwitchID:   mod.TwitchID,
			TwitchName: mod.TwitchName,
			Avatar:     mod.Avatar,
			AddedAt:    mod.AddedAt.Format("2006-01-02 15:04:05"),
		})
	}

	writeAPISuccess(w, moderatorResponses)
}

// AddModerator adds a new moderator for a user
func AddModerator(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req AddModeratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.TwitchName == "" {
		writeAPIError(w, "Twitch name is required", http.StatusBadRequest)
		return
	}

	// Get user from database
	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Get the reward listener to access Twitch API
	rl := twitch.GetRewardListener(userID)
	if rl == nil {
		writeAPIError(w, "Twitch API not available", http.StatusInternalServerError)
		return
	}

	// Search for the user on Twitch to get their info
	twitchUser, err := rl.GetTwitchUserByName(req.TwitchName)
	if err != nil {
		writeAPIError(w, "Failed to find user on Twitch", http.StatusNotFound)
		return
	}

	// Add moderator
	err = db.AddModerator(database, streamer.ID, twitchUser.ID, twitchUser.DisplayName, twitchUser.ProfileImageURL)
	if err != nil {
		writeAPIError(w, "Failed to add moderator", http.StatusInternalServerError)
		return
	}

	writeAPISuccess(w, map[string]string{
		"message": "Moderator added successfully",
	})
}

// RemoveModerator removes a moderator for a user
func RemoveModerator(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]
	moderatorID := vars["moderatorID"]

	if userID == "" || moderatorID == "" {
		writeAPIError(w, "User ID and Moderator ID are required", http.StatusBadRequest)
		return
	}

	// Convert moderator ID to uint
	modID, err := strconv.ParseUint(moderatorID, 10, 32)
	if err != nil {
		writeAPIError(w, "Invalid moderator ID", http.StatusBadRequest)
		return
	}

	// Get user from database
	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Remove moderator
	err = db.RemoveModerator(database, streamer.ID, uint(modID))
	if err != nil {
		writeAPIError(w, "Failed to remove moderator", http.StatusInternalServerError)
		return
	}

	writeAPISuccess(w, map[string]string{
		"message": "Moderator removed successfully",
	})
}

// SearchTwitchUsers searches for Twitch users (for moderator autocomplete)
func SearchTwitchUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]
	query := r.URL.Query().Get("q")

	if userID == "" || query == "" {
		writeAPIError(w, "User ID and query are required", http.StatusBadRequest)
		return
	}

	// Get the reward listener to access Twitch API
	rl := twitch.GetRewardListener(userID)
	if rl == nil {
		writeAPIError(w, "Twitch API not available", http.StatusInternalServerError)
		return
	}

	// Search for users on Twitch
	users, err := rl.SearchTwitchUsers(query)
	if err != nil {
		writeAPIError(w, "Failed to search users", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var userResults []TwitchUserSearchResult
	for _, user := range users {
		userResults = append(userResults, TwitchUserSearchResult{
			ID:          user.ID,
			Login:       user.Login,
			DisplayName: user.DisplayName,
			Avatar:      user.ProfileImageURL,
		})
	}

	writeAPISuccess(w, userResults)
}

// Helper functions
func writeAPISuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)

	response := APIResponse{
		Success: true,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func writeAPIError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(statusCode)

	response := APIResponse{
		Success: false,
		Error:   message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON error response: %v", err)
	}
}

func writeAPIResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)

	response := APIResponse{
		Success: true,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
