package twitch

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/emcifuntik/twitch-spotify-request/internal/db"
	"github.com/emcifuntik/twitch-spotify-request/internal/spotify"
	"github.com/nicklaw5/helix/v2"
	spotifylib "github.com/zmb3/spotify/v2"
)

// formatDuration formats seconds into a human readable time format
// For times >= 60 minutes, shows hours:minutes:seconds
// For times < 60 minutes, shows minutes:seconds
func formatDuration(totalSeconds int) string {
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// RewardID represents internal reward types
type RewardID int8

const (
	RewardIDRequestSong RewardID = 1
	RewardIDSkipSong    RewardID = 2
)

// ChatCommand represents chat commands
type ChatCommand string

const (
	ChatCommandSongQueue   ChatCommand = "!sq"
	ChatCommandSongCurrent ChatCommand = "!sc"
	ChatCommandSongsRecent ChatCommand = "!sr"
	ChatCommandSongVolume  ChatCommand = "!volume"
	ChatCommandSongHelp    ChatCommand = "!songhelp"
)

// SongQueueData represents the current queue state using Spotify's native queue
type SongQueueData struct {
	Queue        []spotifylib.FullTrack `json:"q"`
	CurrentTrack *spotifylib.FullTrack  `json:"currentTrack,omitempty"`
	Progress     int                    `json:"progress"`
	Duration     int                    `json:"duration"`
	LastUpdated  int64                  `json:"lastUpdated"`
}

// RewardListener handles Twitch rewards and chat commands for a streamer
type RewardListener struct {
	streamer           *db.Streamer
	client             *helix.Client
	spotifyClient      *spotify.SpotifyClient
	rewards            []db.Reward
	lastQueueCalcTime  int64
	lastQueue          *SongQueueData
	streamerName       string
	streamerNameUpdate int64
}

// Constants
const (
	QueueCalcDelay          = 30 * time.Second
	AllowedBroadcasterTypes = "partner,affiliate"
)

// Map with Twitch streamer IDs to their corresponding RewardListener instances.
var rewardListeners = make(map[string]*RewardListener)

// GetOrCreateRewardListener gets an existing listener or creates a new one
func GetOrCreateRewardListener(streamer *db.Streamer) *RewardListener {
	// Check if listener already exists
	if existing, exists := rewardListeners[streamer.ChannelID]; exists {
		log.Printf("Listener already exists for channel %s, returning existing listener", streamer.ChannelID)
		return existing
	}

	log.Printf("Creating new listener for channel %s", streamer.ChannelID)
	return NewRewardListener(streamer)
}

// InvalidateRewardListener invalidates the listener for a specific streamer
func InvalidateRewardListener(streamerID string) {
	if listener, exists := rewardListeners[streamerID]; exists {
		log.Printf("Invalidating listener for streamer %s", streamerID)
		listener.client = nil               // Invalidate the client
		delete(rewardListeners, streamerID) // Remove from map
	} else {
		log.Printf("No listener found for streamer %s to invalidate", streamerID)
	}
}

// NewRewardListener creates a new reward listener for a streamer
func NewRewardListener(streamer *db.Streamer) *RewardListener {
	httpClient := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false, // Force HTTP/1.1
			TLSNextProto:      make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		},
	}

	client, err := helix.NewClient(&helix.Options{
		ClientID:        os.Getenv("TWITCH_CLIENT_ID"),
		ClientSecret:    os.Getenv("TWITCH_CLIENT_SECRET"),
		UserAccessToken: streamer.TwitchToken,
		RefreshToken:    streamer.TwitchRefresh,
		HTTPClient:      httpClient, // Use HTTP/1.1 client
	})

	if err != nil {
		log.Printf("Error creating Twitch client: %v", err)
		return nil
	}

	// Create Spotify client
	spotifyClient := spotify.NewSpotifyClient(
		streamer.SpotifyToken,
		streamer.SpotifyRefresh,
		func(accessToken, refreshToken string) error {
			return updateSpotifyTokens(streamer.ID, accessToken, refreshToken)
		},
	)

	rl := &RewardListener{
		streamer:      streamer,
		client:        client,
		spotifyClient: spotifyClient,
	}

	// Load existing rewards
	if err := rl.loadRewards(); err != nil {
		log.Printf("Error loading rewards: %v", err)
	}

	// Try to set up rewards
	if err := rl.setupRewards(); err != nil {
		log.Printf("Error setting up rewards: %v", err)
	}

	// Start periodic cleanup
	rl.startPeriodicCleanup()

	// Initialize cooldown manager cleanup (only once globally)
	globalCooldownManager.StartPeriodicCleanup()

	rewardListeners[streamer.ChannelID] = rl
	AddStreamer(streamer.ChannelID)

	return rl
}

// updateSpotifyTokens updates both Spotify access and refresh tokens in the database
func updateSpotifyTokens(streamerID uint, accessToken, refreshToken string) error {
	// Get database connection
	database := db.GetDB()
	if database == nil {
		log.Printf("Database not available for token update")
		return fmt.Errorf("database not available")
	}

	// Update both tokens
	updates := map[string]interface{}{
		"streamer_spotify_token": accessToken,
	}

	// Only update refresh token if it's provided (not empty)
	if refreshToken != "" {
		updates["streamer_spotify_refresh"] = refreshToken
		log.Printf("Updating both access and refresh tokens for streamer %d", streamerID)
	} else {
		log.Printf("Updating only access token for streamer %d", streamerID)
	}

	result := database.Model(&db.Streamer{}).Where("streamer_id = ?", streamerID).Updates(updates)
	if result.Error != nil {
		log.Printf("Error updating Spotify tokens: %v", result.Error)
		return result.Error
	}

	log.Printf("Spotify tokens updated successfully for streamer %d", streamerID)
	return nil
}

// loadRewards loads the rewards for this streamer from the database
func (rl *RewardListener) loadRewards() error {
	database := db.GetDB()
	if database == nil {
		log.Printf("Database not available for loading rewards")
		rl.rewards = []db.Reward{} // Initialize empty slice
		return nil
	}

	var rewards []db.Reward
	result := database.Where("reward_streamer = ?", rl.streamer.ID).Find(&rewards)
	if result.Error != nil {
		log.Printf("Error loading rewards: %v", result.Error)
		rl.rewards = []db.Reward{} // Initialize empty slice on error
		return result.Error
	}

	rl.rewards = rewards
	log.Printf("Loaded %d rewards for streamer %d", len(rewards), rl.streamer.ID)
	return nil
}

// saveReward saves a reward to the database
func (rl *RewardListener) saveReward(internalID RewardID, twitchID string) error {
	database := db.GetDB()
	if database == nil {
		log.Printf("Database not available for saving reward")
		return fmt.Errorf("database not available")
	}

	reward := db.Reward{
		StreamerID: rl.streamer.ID,
		InternalID: int8(internalID),
		TwitchID:   twitchID,
	}

	result := database.Create(&reward)
	if result.Error != nil {
		log.Printf("Error saving reward: %v", result.Error)
		return result.Error
	}

	// Add to local rewards slice
	rl.rewards = append(rl.rewards, reward)
	log.Printf("Saved reward %s for streamer %d", twitchID, rl.streamer.ID)
	return nil
}

// setupRewards sets up the Twitch rewards
func (rl *RewardListener) setupRewards() error {
	// Get channel info to check broadcaster type
	channelResp, err := rl.client.GetChannelInformation(&helix.GetChannelInformationParams{
		BroadcasterIDs: []string{rl.streamer.ChannelID},
	})

	if err != nil {
		return fmt.Errorf("failed to get channel info: %w", err)
	}

	if len(channelResp.Data.Channels) == 0 {
		return fmt.Errorf("no channel data found")
	}

	// Check if broadcaster type is allowed (partner or affiliate)
	// Note: The helix library might not expose BroadcasterType in ChannelInformation
	// We'll skip this check for now and implement it later if needed
	log.Printf("Setting up rewards for channel %s", rl.streamer.ChannelID)

	// Set up song request reward
	if !rl.hasReward(RewardIDRequestSong) {
		if err := rl.setupSongRequestReward(); err != nil {
			log.Printf("Error setting up song request reward: %v", err)
		}
	} else {
		log.Printf("Song request reward already exists for streamer %d", rl.streamer.ID)
	}

	// Set up skip song reward
	if !rl.hasReward(RewardIDSkipSong) {
		if err := rl.setupSkipSongReward(); err != nil {
			log.Printf("Error setting up skip song reward: %v", err)
		}
	} else {
		log.Printf("Skip song reward already exists for streamer %d", rl.streamer.ID)
	}

	return nil
}

// setupSongRequestReward creates the song request reward
func (rl *RewardListener) setupSongRequestReward() error {
	// Use a unique title to avoid conflicts
	title := fmt.Sprintf("Request song (Bot %d)", rl.streamer.ID)

	response, err := rl.client.CreateCustomReward(&helix.ChannelCustomRewardsParams{
		BroadcasterID:       rl.streamer.ChannelID,
		Title:               title,
		Cost:                300,
		Prompt:              "Enter artist and song name to add request",
		IsUserInputRequired: true,
		BackgroundColor:     "#aaaa00",
		IsEnabled:           true,
	})

	if err != nil {
		return fmt.Errorf("failed to create song request reward: %w", err)
	}

	if len(response.Data.ChannelCustomRewards) == 0 {
		return fmt.Errorf("no reward data returned")
	}

	rewardID := response.Data.ChannelCustomRewards[0].ID
	log.Printf("Created song request reward with ID: %s", rewardID)

	// Save to database
	if err := rl.saveReward(RewardIDRequestSong, rewardID); err != nil {
		log.Printf("Error saving song request reward to database: %v", err)
	}

	return nil
}

// setupSkipSongReward creates the skip song reward
func (rl *RewardListener) setupSkipSongReward() error {
	// Use a unique title to avoid conflicts
	title := fmt.Sprintf("Skip song (Bot %d)", rl.streamer.ID)

	response, err := rl.client.CreateCustomReward(&helix.ChannelCustomRewardsParams{
		BroadcasterID:       rl.streamer.ChannelID,
		Title:               title,
		Cost:                1000,
		Prompt:              "Skip current song",
		IsUserInputRequired: false,
		BackgroundColor:     "#00aaaa",
		IsEnabled:           true,
	})

	if err != nil {
		return fmt.Errorf("failed to create skip song reward: %w", err)
	}

	if len(response.Data.ChannelCustomRewards) == 0 {
		return fmt.Errorf("no reward data returned")
	}

	rewardID := response.Data.ChannelCustomRewards[0].ID
	log.Printf("Created skip song reward with ID: %s", rewardID)

	// Save to database
	if err := rl.saveReward(RewardIDSkipSong, rewardID); err != nil {
		log.Printf("Error saving skip song reward to database: %v", err)
	}

	return nil
}

// HandleRewardRedemption handles reward redemptions from EventSub
func HandleRewardRedemption(streamerID string, redemptionID string, rewardID string, userID string, userName string, promptText string) error {
	log.Printf("Handling reward redemption for streamer ID: %s", streamerID)

	rl, exists := rewardListeners[streamerID]
	if !exists {
		log.Printf("No RewardListener found for streamer ID: %s", streamerID)
		return nil
	}

	log.Printf("Found RewardListener for streamer ID: %s", streamerID)
	return rl.HandleRewardRedemption(redemptionID, rewardID, userID, userName, promptText)
}

// HandleRewardRedemption processes individual reward redemptions
func (rl *RewardListener) HandleRewardRedemption(redemptionID string, rewardID string, userID string, userName string, promptText string) error {
	log.Printf("Reward redeemed by %s (ID: %s) for reward ID: %s with input: %s", userName, userID, rewardID, promptText)

	// Find the reward by Twitch ID
	var rewardType RewardID = 0
	var foundReward bool = false
	for _, reward := range rl.rewards {
		if reward.TwitchID == rewardID {
			rewardType = RewardID(reward.InternalID)
			foundReward = true
			log.Printf("Found matching reward: Internal ID %d, Type: %d", reward.InternalID, rewardType)
			break
		}
	}

	if !foundReward {
		log.Printf("No matching reward found for Twitch reward ID: %s. Available rewards:", rewardID)
		for _, reward := range rl.rewards {
			log.Printf("  - Internal ID: %d, Twitch ID: %s", reward.InternalID, reward.TwitchID)
		}
		return nil // Don't process unknown rewards
	}

	// Validate that this is a known reward type
	if !isValidRewardType(rewardType) {
		log.Printf("Invalid reward type %d for reward ID %s, ignoring", rewardType, rewardID)
		return nil
	}

	// Handle the reward based on type
	switch rewardType {
	case RewardIDRequestSong:
		return rl.handleSongRequest(userName, promptText, redemptionID, rewardID)
	case RewardIDSkipSong:
		return rl.handleSongSkip(userName, redemptionID, rewardID)
	default:
		log.Printf("Unknown reward type for reward ID %s, ignoring", rewardID)
		return nil
	}
}

// handleSongRequest processes song request rewards
func (rl *RewardListener) handleSongRequest(userName, query, redemptionID, rewardID string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleSongRequest: %v", r)
		}
	}()

	// Check if it's a Spotify URL
	if spotify.IsSpotifyURL(query) {
		return rl.handleSpotifyURL(userName, query, redemptionID, rewardID)
	}

	// Search for the track
	return rl.handleSearchQuery(userName, query, redemptionID, rewardID)
}

// handleSpotifyURL processes Spotify URL requests
func (rl *RewardListener) handleSpotifyURL(userName, url, redemptionID, rewardID string) error {
	trackID := spotify.GetTrackIDFromURL(url)
	if trackID == "" {
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	track, err := rl.spotifyClient.GetTrackByID(trackID)
	if err != nil || track.URI == "" {
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	return rl.enqueueTrack(userName, track, redemptionID, rewardID)
}

// handleSearchQuery processes search query requests
func (rl *RewardListener) handleSearchQuery(userName, query, redemptionID, rewardID string) error {
	searchResult, err := rl.spotifyClient.SearchTracks(query)
	if err != nil {
		log.Printf("Error searching tracks: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	if len(searchResult.Tracks.Tracks) == 0 {
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	track := &searchResult.Tracks.Tracks[0]
	return rl.enqueueTrack(userName, track, redemptionID, rewardID)
}

// enqueueTrack adds a track to the Spotify queue with enhanced validation
func (rl *RewardListener) enqueueTrack(userName string, track *spotifylib.FullTrack, redemptionID, rewardID string) error {
	database := db.GetDB()
	if database == nil {
		log.Printf("Database not available for track validation")
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при обработке запроса", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	// Get artist IDs and track ID for blocking check
	var artistIDs []string
	for _, artist := range track.Artists {
		artistIDs = append(artistIDs, string(artist.ID))
	}
	trackID := string(track.ID)

	// Check if track/artist is blocked
	if db.IsBlocked(database, rl.streamer.ID, artistIDs, trackID) {
		rl.sendMessage(fmt.Sprintf("@%s этот трек или исполнитель заблокирован", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	// Check max song length
	maxLength := db.GetMaxSongLength(database, rl.streamer.ID)
	if int(track.Duration) > maxLength*1000 { // Duration is in milliseconds
		minutes := maxLength / 60
		seconds := maxLength % 60
		rl.sendMessage(fmt.Sprintf("@%s трек слишком длинный (макс. %d:%02d)", userName, minutes, seconds))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	// Check cooldown for the same song
	cooldownManager := GetCooldownManager()
	cooldownSeconds := db.GetCooldownSameSong(database, rl.streamer.ID)

	if cooldownManager.IsOnCooldown(rl.streamer.ChannelID, string(track.URI), cooldownSeconds) {
		remaining := cooldownManager.GetRemainingCooldown(rl.streamer.ChannelID, string(track.URI), cooldownSeconds)
		timeStr := formatDuration(remaining)
		rl.sendMessage(fmt.Sprintf("@%s этот трек недавно играл, повторить можно через %s", userName, timeStr))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	// Check for duplicates (existing logic)
	if spotify.GlobalDuplicateStore.Exists(string(track.URI)) {
		rl.sendMessage(fmt.Sprintf("@%s этот трек уже играл за последний час", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	songName := spotify.SongItemToReadable(track)

	// Add to Spotify queue
	if err := rl.spotifyClient.EnqueueTrack(track.URI); err != nil {
		log.Printf("Error enqueueing track: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при добавлении трека", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	// Add to duplicate store
	spotify.GlobalDuplicateStore.Add(string(track.URI))

	// Add to cooldown manager
	cooldownManager.AddCooldown(rl.streamer.ChannelID, string(track.URI))

	rl.sendMessage(fmt.Sprintf("@%s %s добавлена в очередь", userName, songName))
	return rl.updateRedemptionStatus(redemptionID, rewardID, "FULFILLED")
}

// handleSongSkip processes song skip rewards
func (rl *RewardListener) handleSongSkip(userName, redemptionID, rewardID string) error {
	if err := rl.spotifyClient.NextTrack(); err != nil {
		log.Printf("Error skipping track: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при пропуске трека", userName))
		return rl.updateRedemptionStatus(redemptionID, rewardID, "CANCELED")
	}

	rl.sendMessage(fmt.Sprintf("@%s трек пропущен по твоему запросу", userName))
	return rl.updateRedemptionStatus(redemptionID, rewardID, "FULFILLED")
}

// updateRedemptionStatus updates the status of a reward redemption
func (rl *RewardListener) updateRedemptionStatus(redemptionID, rewardID, status string) error {
	// Find the reward ID - this is a simplified implementation
	// In a real implementation, you'd store and lookup the reward mapping

	// For now, we'll skip the actual API call as we need both reward ID and redemption ID
	// and our current event structure doesn't provide both cleanly
	log.Printf("Would update redemption %s status to %s", redemptionID, status)

	// Here's how you would do it with proper IDs:
	_, err := rl.client.UpdateChannelCustomRewardsRedemptionStatus(&helix.UpdateChannelCustomRewardsRedemptionStatusParams{
		BroadcasterID: rl.streamer.ChannelID,
		RewardID:      rewardID,
		ID:            redemptionID,
		Status:        status,
	})
	return err
}

// sendMessage sends a message to the chat using Helix API
func (rl *RewardListener) sendMessage(message string) {
	if rl.streamer.ChannelID == "" {
		log.Printf("Cannot send message: channel ID not available")
		return
	}

	log.Printf("Attempting to send message to channel %s: %s", rl.streamer.ChannelID, message)

	// Use Helix API to send the message
	resp, err := rl.client.SendChatMessage(&helix.SendChatMessageParams{
		BroadcasterID: rl.streamer.ChannelID,
		SenderID:      rl.streamer.ChannelID, // Send as the broadcaster
		Message:       message,
	})

	if err != nil {
		log.Printf("❌ Error sending chat message: %v", err)
		return
	}

	if resp.Error != "" {
		log.Printf("❌ Helix API error sending chat message: %s - %s", resp.Error, resp.ErrorMessage)
		return
	}

	log.Printf("✅ Chat message sent successfully to channel %s: %s", rl.streamer.ChannelID, message)
}

// getChannelName gets the channel name for chat messages
func (rl *RewardListener) getChannelName() string {
	now := time.Now().Unix()

	// Cache channel name for 1 hour
	if now-rl.streamerNameUpdate < 3600 && rl.streamerName != "" {
		return rl.streamerName
	}

	// Get channel info from Helix API
	channelResp, err := rl.client.GetChannelInformation(&helix.GetChannelInformationParams{
		BroadcasterIDs: []string{rl.streamer.ChannelID},
	})

	if err != nil {
		log.Printf("Error getting channel info: %v", err)
		return ""
	}

	if len(channelResp.Data.Channels) == 0 {
		log.Printf("No channel data found for ID: %s", rl.streamer.ChannelID)
		return ""
	}

	rl.streamerName = channelResp.Data.Channels[0].BroadcasterName
	rl.streamerNameUpdate = now
	return rl.streamerName
}

// GetQueueData calculates and returns current queue data using Spotify's native queue
func (rl *RewardListener) GetQueueData() (*SongQueueData, error) {
	now := time.Now().Unix()
	if now-rl.lastQueueCalcTime < int64(QueueCalcDelay.Seconds()) && rl.lastQueue != nil {
		return rl.lastQueue, nil
	}

	currentTrack, err := rl.spotifyClient.GetCurrentTrack()
	if err != nil || currentTrack.Item == nil {
		return &SongQueueData{
			Queue:        []spotifylib.FullTrack{},
			CurrentTrack: nil,
			Duration:     0,
			Progress:     0,
		}, nil
	}

	// Get Spotify's native queue
	spotifyQueue, err := rl.spotifyClient.GetQueue()
	if err != nil {
		log.Printf("Error getting Spotify queue: %v", err)
		return &SongQueueData{
			Queue:        []spotifylib.FullTrack{},
			CurrentTrack: currentTrack.Item,
			Duration:     int(currentTrack.Item.Duration),
			Progress:     int(currentTrack.Progress),
		}, nil
	}

	rl.lastQueue = &SongQueueData{
		Queue:        spotifyQueue.Items,
		CurrentTrack: currentTrack.Item,
		Duration:     int(currentTrack.Item.Duration),
		Progress:     int(currentTrack.Progress),
	}

	rl.lastQueueCalcTime = now
	return rl.lastQueue, nil
}

// GetRewardListener returns the reward listener for a given channel ID
func GetRewardListener(channelID string) *RewardListener {
	return rewardListeners[channelID]
}

// CheckRewardsStatus checks if rewards are properly configured for a channel
func CheckRewardsStatus(channelID string) bool {
	rl := GetRewardListener(channelID)
	if rl == nil {
		return false
	}
	return rl.CheckRewardsConfigured()
}

// FixRewardsForChannel attempts to fix rewards for a channel
func FixRewardsForChannel(channelID string) error {
	rl := GetRewardListener(channelID)
	if rl == nil {
		return fmt.Errorf("no reward listener found for channel %s", channelID)
	}
	return rl.FixRewards()
}

// HandleChatCommand processes chat commands
func (rl *RewardListener) HandleChatCommand(userName, command, args string) {
	log.Printf("Processing chat command: %s from user: %s with args: %s", command, userName, args)

	switch ChatCommand(command) {
	case ChatCommandSongHelp:
		log.Printf("Handling song help command for user: %s", userName)
		rl.handleSongHelp(userName)
	case ChatCommandSongVolume:
		log.Printf("Handling volume command for user: %s with args: %s", userName, args)
		rl.handleVolumeCommand(userName, args)
	case ChatCommandSongsRecent:
		log.Printf("Handling recent songs command for user: %s", userName)
		rl.handleRecentSongs(userName)
	case ChatCommandSongCurrent:
		log.Printf("Handling current song command for user: %s", userName)
		rl.handleCurrentSong(userName)
	case ChatCommandSongQueue:
		log.Printf("Handling song queue command for user: %s", userName)
		rl.handleSongQueue(userName)
	default:
		log.Printf("Unknown command: %s from user: %s", command, userName)
	}
}

// handleSongHelp shows available commands
func (rl *RewardListener) handleSongHelp(userName string) {
	rl.sendMessage(fmt.Sprintf("@%s Available commands: !sc - current song; !sq - view queue; !sr - recent songs; !volume <0-100> - change volume (mods/broadcaster only)", userName))
}

// handleVolumeCommand changes the volume
func (rl *RewardListener) handleVolumeCommand(userName, args string) {
	log.Printf("Volume command received from user: %s, args: %s", userName, args)

	if args == "" {
		log.Printf("Volume command failed: no volume specified by user %s", userName)
		rl.sendMessage(fmt.Sprintf("@%s Please specify a volume level (0-100)", userName))
		return
	}

	volume, err := strconv.Atoi(args)
	if err != nil {
		log.Printf("Volume command failed: invalid volume '%s' from user %s: %v", args, userName, err)
		rl.sendMessage(fmt.Sprintf("@%s Please specify a valid volume level (0-100)", userName))
		return
	}

	log.Printf("Checking permissions for user %s", userName)
	// Check if user is mod or broadcaster
	if !rl.isUserModOrBroadcaster(userName) {
		log.Printf("Volume command denied: user %s is not mod or broadcaster", userName)
		rl.sendMessage(fmt.Sprintf("@%s Only moderators and the broadcaster can change volume", userName))
		return
	}

	log.Printf("Permission check passed for user %s", userName)

	// Clamp volume between 0 and 100
	originalVolume := volume
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}

	if originalVolume != volume {
		log.Printf("Volume clamped from %d to %d", originalVolume, volume)
	}

	log.Printf("Setting Spotify volume to %d%%", volume)
	if err := rl.spotifyClient.SetVolume(volume); err != nil {
		log.Printf("Error setting volume to %d%%: %v", volume, err)
		rl.sendMessage(fmt.Sprintf("@%s Error setting volume: %v", userName, err))
		return
	}

	log.Printf("Volume successfully set to %d%% for user %s", volume, userName)
	rl.sendMessage(fmt.Sprintf("@%s Volume set to %d%%", userName, volume))
}

// isUserModOrBroadcaster checks if a user is a moderator or the broadcaster
func (rl *RewardListener) isUserModOrBroadcaster(userName string) bool {
	log.Printf("Checking if user %s is mod or broadcaster", userName)

	channelName := rl.getChannelName()
	log.Printf("Channel name: %s", channelName)

	if channelName == "" {
		log.Printf("Channel name is empty, denying permission")
		return false
	}

	// Check if user is the broadcaster
	if strings.EqualFold(userName, channelName) {
		log.Printf("User %s is the broadcaster (channel: %s)", userName, channelName)
		return true
	}

	log.Printf("User %s is not the broadcaster, checking bot moderators", userName)

	// Check if user is a bot moderator
	database := db.GetDB()
	if database != nil {
		if db.IsBotModeratorByName(database, rl.streamer.ID, userName) {
			log.Printf("User %s is a bot moderator", userName)
			return true
		}
	}

	log.Printf("User %s is not a bot moderator, checking Twitch moderators", userName)

	// Get moderators list from Twitch
	mods, err := rl.client.GetModerators(&helix.GetModeratorsParams{
		BroadcasterID: rl.streamer.ChannelID,
	})

	if err != nil {
		log.Printf("Error getting moderators for channel %s (ID: %s): %v", channelName, rl.streamer.ChannelID, err)
		return false
	}

	log.Printf("Found %d Twitch moderators for channel %s", len(mods.Data.Moderators), channelName)

	// Check if user is in Twitch moderators list
	for _, mod := range mods.Data.Moderators {
		log.Printf("Checking moderator: %s vs user: %s", mod.UserName, userName)
		if strings.EqualFold(mod.UserName, userName) {
			log.Printf("User %s found in Twitch moderators list", userName)
			return true
		}
	}

	log.Printf("User %s not found in moderators list", userName)
	return false
}

// handleRecentSongs shows recently played tracks
func (rl *RewardListener) handleRecentSongs(userName string) {
	recentTracks, err := rl.spotifyClient.GetRecentlyPlayed(5)
	if err != nil {
		log.Printf("Error getting recent tracks: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s Error getting recent tracks", userName))
		return
	}

	var trackNames []string
	for _, item := range recentTracks {
		trackNames = append(trackNames, spotify.SongItemToReadableSimple(&item.Track))
	}

	rl.sendMessage(fmt.Sprintf("@%s Recent tracks: %s", userName, strings.Join(trackNames, "; ")))
}

// handleCurrentSong shows the currently playing track
func (rl *RewardListener) handleCurrentSong(userName string) {
	currentTrack, err := rl.spotifyClient.GetCurrentTrack()
	if err != nil || currentTrack.Item == nil {
		rl.sendMessage(fmt.Sprintf("@%s Sorry, an error occurred getting current track", userName))
		return
	}

	songName := spotify.SongItemToReadable(currentTrack.Item)
	rl.sendMessage(fmt.Sprintf("@%s Current track: %s", userName, songName))
}

// handleSongQueue shows the current queue
func (rl *RewardListener) handleSongQueue(userName string) {
	queueData, err := rl.GetQueueData()
	if err != nil {
		log.Printf("Error getting queue data: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s Error getting queue", userName))
		return
	}

	var prettyQueue []string
	for _, item := range queueData.Queue {
		songName := spotify.SongItemToReadable(&item)
		prettyQueue = append(prettyQueue, songName)
	}

	if len(prettyQueue) > 5 {
		rl.sendMessage(fmt.Sprintf("@%s Current queue: %s. View full queue: https://catjammusic.com/queue/%d",
			userName, strings.Join(prettyQueue[:5], "; "), rl.streamer.ID))
	} else {
		rl.sendMessage(fmt.Sprintf("@%s Current queue: %s", userName, strings.Join(prettyQueue, "; ")))
	}
}

// isValidRewardType checks if a reward type is valid and expected
func isValidRewardType(rewardType RewardID) bool {
	switch rewardType {
	case RewardIDRequestSong, RewardIDSkipSong:
		return true
	default:
		return false
	}
}

// hasReward checks if a reward of the given type already exists
func (rl *RewardListener) hasReward(rewardType RewardID) bool {
	for _, reward := range rl.rewards {
		if RewardID(reward.InternalID) == rewardType {
			return true
		}
	}
	return false
}

// CheckRewardsConfigured checks if all required rewards are properly configured
func (rl *RewardListener) CheckRewardsConfigured() bool {
	// Check if both required rewards exist
	hasRequestReward := rl.hasReward(RewardIDRequestSong)
	hasSkipReward := rl.hasReward(RewardIDSkipSong)

	if !hasRequestReward || !hasSkipReward {
		return false
	}

	// Additionally, verify that the rewards still exist on Twitch
	// This is important because rewards might have been deleted from the Twitch side
	return rl.verifyRewardsExistOnTwitch()
}

// verifyRewardsExistOnTwitch checks if our stored rewards still exist on Twitch
func (rl *RewardListener) verifyRewardsExistOnTwitch() bool {
	// Get all custom rewards from Twitch
	resp, err := rl.client.GetCustomRewards(&helix.GetCustomRewardsParams{
		BroadcasterID: rl.streamer.ChannelID,
	})

	if err != nil {
		log.Printf("Error getting custom rewards for verification: %v", err)
		return false
	}

	// Create a map of existing Twitch reward IDs
	twitchRewards := make(map[string]bool)
	for _, reward := range resp.Data.ChannelCustomRewards {
		twitchRewards[reward.ID] = true
	}

	// Check if all our stored rewards still exist
	for _, reward := range rl.rewards {
		if !twitchRewards[reward.TwitchID] {
			log.Printf("Reward %s (internal ID: %d) no longer exists on Twitch", reward.TwitchID, reward.InternalID)
			return false
		}
	}

	return true
}

// FixRewards attempts to fix/recreate rewards
func (rl *RewardListener) FixRewards() error {
	log.Printf("Attempting to fix rewards for streamer %d", rl.streamer.ID)

	// First, clean up any existing rewards that might conflict
	if err := rl.cleanupExistingRewards(); err != nil {
		log.Printf("Error cleaning up existing rewards: %v", err)
		// Continue anyway - we'll try to create new ones
	}

	// Clear our internal rewards list
	rl.rewards = []db.Reward{}

	// Clear from database
	database := db.GetDB()
	if database != nil {
		result := database.Where("reward_streamer = ?", rl.streamer.ID).Delete(&db.Reward{})
		if result.Error != nil {
			log.Printf("Error clearing rewards from database: %v", result.Error)
		}
	}

	// Try to set up rewards again
	return rl.setupRewards()
}

// cleanupExistingRewards removes any existing rewards with conflicting names
func (rl *RewardListener) cleanupExistingRewards() error {
	// Get all custom rewards from Twitch
	resp, err := rl.client.GetCustomRewards(&helix.GetCustomRewardsParams{
		BroadcasterID: rl.streamer.ChannelID,
	})

	if err != nil {
		return fmt.Errorf("failed to get existing rewards: %w", err)
	}

	// Delete any rewards that match our titles
	rewardTitles := map[string]bool{
		"Request song": true,
		"Skip song":    true,
	}

	for _, reward := range resp.Data.ChannelCustomRewards {
		if rewardTitles[reward.Title] {
			log.Printf("Found existing reward: %s (ID: %s) - will be replaced", reward.Title, reward.ID)
			// Try to delete the reward - this may fail if we don't have permissions
			// but we'll continue anyway
			_, err := rl.client.DeleteCustomRewards(&helix.DeleteCustomRewardsParams{
				BroadcasterID: rl.streamer.ChannelID,
				ID:            reward.ID,
			})
			if err != nil {
				log.Printf("Could not delete existing reward %s: %v", reward.ID, err)
			}
		}
	}

	return nil
}

// HandleChatMessage handles chat messages from EventSub
func HandleChatMessage(broadcasterUserID string, chatterUserID string, chatterUserName string, messageText string) error {
	log.Printf("Handling chat message from %s (ID: %s) in channel %s: %s", chatterUserName, chatterUserID, broadcasterUserID, messageText)

	rl, exists := rewardListeners[broadcasterUserID]
	if !exists {
		log.Printf("No RewardListener found for broadcaster ID: %s", broadcasterUserID)
		return nil
	}

	// Check if message is a command (starts with !)
	if !strings.HasPrefix(messageText, "!") {
		return nil
	}

	parts := strings.Fields(messageText)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	log.Printf("Processing chat command: %s with args: %s from user: %s", command, args, chatterUserName)
	rl.HandleChatCommand(chatterUserName, command, args)
	return nil
}

// startPeriodicCleanup starts background cleanup tasks
func (rl *RewardListener) startPeriodicCleanup() {
	ticker := time.NewTicker(10 * time.Minute) // Run cleanup every 10 minutes

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			// Clean up duplicate store
			spotify.GlobalDuplicateStore.Cleanup()
			log.Printf("Performed periodic cleanup for streamer %d", rl.streamer.ID)
		}
	}()
}

// GetTwitchUserByName gets a Twitch user by their username
func (rl *RewardListener) GetTwitchUserByName(username string) (*helix.User, error) {
	resp, err := rl.client.GetUsers(&helix.UsersParams{
		Logins: []string{username},
	})

	if err != nil {
		return nil, err
	}

	if len(resp.Data.Users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return &resp.Data.Users[0], nil
}

// SearchTwitchUsers searches for Twitch users (limited functionality due to API restrictions)
func (rl *RewardListener) SearchTwitchUsers(query string) ([]helix.User, error) {
	// Twitch API doesn't have a direct search endpoint for users
	// We'll try to get the user directly by their login name (case-insensitive)
	query = strings.ToLower(strings.TrimSpace(query))

	// Try exact match first
	resp, err := rl.client.GetUsers(&helix.UsersParams{
		Logins: []string{query},
	})

	if err != nil {
		log.Printf("Error searching for user %s: %v", query, err)
		return nil, err
	}

	log.Printf("Found %d users for query '%s'", len(resp.Data.Users), query)
	return resp.Data.Users, nil
}
