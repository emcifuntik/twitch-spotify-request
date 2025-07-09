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
	CurrentSong  string                 `json:"currentSong"`
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
	rl.StartPeriodicCleanup()

	// Initialize cooldown manager cleanup (only once globally)
	globalCooldownManager.StartPeriodicCleanup()

	rewardListeners[streamer.ChannelID] = rl
	AddStreamer(streamer.ChannelID)

	return rl
}

// updateSpotifyToken updates the Spotify token in the database
func updateSpotifyToken(streamerID uint, newToken string) error {
	// Get database connection
	database := db.GetDB() // Assuming there's a GetDB function
	if database == nil {
		log.Printf("Database not available for token update")
		return fmt.Errorf("database not available")
	}

	// Update the streamer's Spotify token
	result := database.Model(&db.Streamer{}).Where("streamer_id = ?", streamerID).Update("streamer_spotify_token", newToken)
	if result.Error != nil {
		log.Printf("Error updating Spotify token: %v", result.Error)
		return result.Error
	}

	log.Printf("Spotify token updated for streamer %d", streamerID)
	return nil
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
	response, err := rl.client.CreateCustomReward(&helix.ChannelCustomRewardsParams{
		BroadcasterID:       rl.streamer.ChannelID,
		Title:               "Request song",
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
	response, err := rl.client.CreateCustomReward(&helix.ChannelCustomRewardsParams{
		BroadcasterID:       rl.streamer.ChannelID,
		Title:               "Skip song",
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
func HandleRewardRedemption(streamerID string, rewardID string, userID string, userName string, promptText string) error {
	log.Printf("Handling reward redemption for streamer ID: %s", streamerID)

	rl, exists := rewardListeners[streamerID]
	if !exists {
		log.Printf("No RewardListener found for streamer ID: %s", streamerID)
		return nil
	}

	log.Printf("Found RewardListener for streamer ID: %s", streamerID)
	return rl.HandleRewardRedemption(rewardID, userID, userName, promptText)
}

// HandleRewardRedemption processes individual reward redemptions
func (rl *RewardListener) HandleRewardRedemption(rewardID string, userID string, userName string, promptText string) error {
	log.Printf("Reward redeemed by %s (ID: %s) for reward ID: %s with input: %s", userName, userID, rewardID, promptText)

	// Find the reward by Twitch ID
	var rewardType RewardID = 0
	for _, reward := range rl.rewards {
		if reward.TwitchID == rewardID {
			rewardType = RewardID(reward.InternalID)
			break
		}
	}

	// Handle the reward based on type
	switch rewardType {
	case RewardIDRequestSong:
		return rl.handleSongRequest(userName, promptText, rewardID)
	case RewardIDSkipSong:
		return rl.handleSongSkip(userName, rewardID)
	default:
		// Fallback to old logic for unknown rewards
		if promptText != "" {
			return rl.handleSongRequest(userName, promptText, rewardID)
		} else {
			return rl.handleSongSkip(userName, rewardID)
		}
	}
}

// handleSongRequest processes song request rewards
func (rl *RewardListener) handleSongRequest(userName, query, rewardID string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in handleSongRequest: %v", r)
		}
	}()

	// Check if it's a Spotify URL
	if spotify.IsSpotifyURL(query) {
		return rl.handleSpotifyURL(userName, query, rewardID)
	}

	// Search for the track
	return rl.handleSearchQuery(userName, query, rewardID)
}

// handleSpotifyURL processes Spotify URL requests
func (rl *RewardListener) handleSpotifyURL(userName, url, rewardID string) error {
	trackID := spotify.GetTrackIDFromURL(url)
	if trackID == "" {
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	track, err := rl.spotifyClient.GetTrackByID(trackID)
	if err != nil || track.URI == "" {
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	return rl.enqueueTrack(userName, track, rewardID)
}

// handleSearchQuery processes search query requests
func (rl *RewardListener) handleSearchQuery(userName, query, rewardID string) error {
	searchResult, err := rl.spotifyClient.SearchTracks(query)
	if err != nil {
		log.Printf("Error searching tracks: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	if len(searchResult.Tracks.Tracks) == 0 {
		rl.sendMessage(fmt.Sprintf("@%s ничего не найдено в Spotify", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	track := &searchResult.Tracks.Tracks[0]
	return rl.enqueueTrack(userName, track, rewardID)
}

// enqueueTrack adds a track to the Spotify queue with enhanced validation
func (rl *RewardListener) enqueueTrack(userName string, track *spotifylib.FullTrack, rewardID string) error {
	database := db.GetDB()
	if database == nil {
		log.Printf("Database not available for track validation")
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при обработке запроса", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
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
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	// Check max song length
	maxLength := db.GetMaxSongLength(database, rl.streamer.ID)
	if int(track.Duration) > maxLength*1000 { // Duration is in milliseconds
		minutes := maxLength / 60
		seconds := maxLength % 60
		rl.sendMessage(fmt.Sprintf("@%s трек слишком длинный (макс. %d:%02d)", userName, minutes, seconds))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	// Check cooldown for the same song
	cooldownManager := GetCooldownManager()
	cooldownSeconds := db.GetCooldownSameSong(database, rl.streamer.ID)

	if cooldownManager.IsOnCooldown(rl.streamer.ChannelID, string(track.URI), cooldownSeconds) {
		remaining := cooldownManager.GetRemainingCooldown(rl.streamer.ChannelID, string(track.URI), cooldownSeconds)
		timeStr := formatDuration(remaining)
		rl.sendMessage(fmt.Sprintf("@%s этот трек недавно играл, повторить можно через %s", userName, timeStr))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	// Check for duplicates (existing logic)
	if spotify.GlobalDuplicateStore.Exists(string(track.URI)) {
		rl.sendMessage(fmt.Sprintf("@%s этот трек уже играл за последний час", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	songName := spotify.SongItemToReadable(track)

	// Add to Spotify queue
	if err := rl.spotifyClient.EnqueueTrack(track.URI); err != nil {
		log.Printf("Error enqueueing track: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при добавлении трека", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	// Add to duplicate store
	spotify.GlobalDuplicateStore.Add(string(track.URI))

	// Add to cooldown manager
	cooldownManager.AddCooldown(rl.streamer.ChannelID, string(track.URI))

	rl.sendMessage(fmt.Sprintf("@%s %s добавлена в очередь", userName, songName))
	return rl.updateRedemptionStatus(rewardID, "FULFILLED")
}

// handleSongSkip processes song skip rewards
func (rl *RewardListener) handleSongSkip(userName, rewardID string) error {
	if err := rl.spotifyClient.NextTrack(); err != nil {
		log.Printf("Error skipping track: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при пропуске трека", userName))
		return rl.updateRedemptionStatus(rewardID, "CANCELED")
	}

	rl.sendMessage(fmt.Sprintf("@%s трек пропущен по твоему запросу", userName))
	return rl.updateRedemptionStatus(rewardID, "FULFILLED")
}

// updateRedemptionStatus updates the status of a reward redemption
func (rl *RewardListener) updateRedemptionStatus(redemptionID, status string) error {
	// Find the reward ID - this is a simplified implementation
	// In a real implementation, you'd store and lookup the reward mapping

	// For now, we'll skip the actual API call as we need both reward ID and redemption ID
	// and our current event structure doesn't provide both cleanly
	log.Printf("Would update redemption %s status to %s", redemptionID, status)

	// Here's how you would do it with proper IDs:
	// _, err := rl.client.ManageRedemption(&helix.ManageRedemptionParams{
	//     BroadcasterID: rl.streamer.ChannelID,
	//     RewardID:      actualRewardID,
	//     RedemptionIDs: []string{redemptionID},
	//     Status:        status,
	// })
	// return err

	return nil
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
			CurrentSong:  "",
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
			CurrentSong:  spotify.SongItemToReadable(currentTrack.Item),
			CurrentTrack: currentTrack.Item,
			Duration:     int(currentTrack.Item.Duration),
			Progress:     int(currentTrack.Progress),
		}, nil
	}

	rl.lastQueue = &SongQueueData{
		Queue:        spotifyQueue.Items,
		CurrentSong:  spotify.SongItemToReadable(currentTrack.Item),
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

// HandleChatCommand processes chat commands
func (rl *RewardListener) HandleChatCommand(userName, command, args string) {
	switch ChatCommand(command) {
	case ChatCommandSongHelp:
		rl.handleSongHelp(userName)
	case ChatCommandSongVolume:
		rl.handleVolumeCommand(userName, args)
	case ChatCommandSongsRecent:
		rl.handleRecentSongs(userName)
	case ChatCommandSongCurrent:
		rl.handleCurrentSong(userName)
	case ChatCommandSongQueue:
		rl.handleSongQueue(userName)
	}
}

// handleSongHelp shows available commands
func (rl *RewardListener) handleSongHelp(userName string) {
	rl.sendMessage(fmt.Sprintf("@%s список доступных комманд: !sc - узнать текущий трек; !sr - список прошлых треков", userName))
}

// handleVolumeCommand changes the volume
func (rl *RewardListener) handleVolumeCommand(userName, args string) {
	if args == "" {
		return
	}

	volume, err := strconv.Atoi(args)
	if err != nil {
		return
	}

	// Check if user is mod or broadcaster
	if !rl.isUserModOrBroadcaster(userName) {
		rl.sendMessage(fmt.Sprintf("@%s только модераторы и стример могут менять громкость", userName))
		return
	}

	// Clamp volume between 0 and 100
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}

	if err := rl.spotifyClient.SetVolume(volume); err != nil {
		log.Printf("Error setting volume: %v", err)
		return
	}

	rl.sendMessage(fmt.Sprintf("@%s звук выставлен на %d%%", userName, volume))
}

// isUserModOrBroadcaster checks if a user is a moderator or the broadcaster
func (rl *RewardListener) isUserModOrBroadcaster(userName string) bool {
	channelName := rl.getChannelName()
	if channelName == "" {
		return false
	}

	// Check if user is the broadcaster
	if strings.EqualFold(userName, channelName) {
		return true
	}

	// Get moderators list
	mods, err := rl.client.GetModerators(&helix.GetModeratorsParams{
		BroadcasterID: rl.streamer.ChannelID,
	})

	if err != nil {
		log.Printf("Error getting moderators: %v", err)
		return false
	}

	// Check if user is in moderators list
	for _, mod := range mods.Data.Moderators {
		if strings.EqualFold(mod.UserName, userName) {
			return true
		}
	}

	return false
}

// handleRecentSongs shows recently played tracks
func (rl *RewardListener) handleRecentSongs(userName string) {
	recentTracks, err := rl.spotifyClient.GetRecentlyPlayed(5)
	if err != nil {
		log.Printf("Error getting recent tracks: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при получении последних треков", userName))
		return
	}

	var trackNames []string
	for _, item := range recentTracks {
		trackNames = append(trackNames, spotify.SongItemToReadableSimple(&item.Track))
	}

	rl.sendMessage(fmt.Sprintf("@%s последние проигранные треки: %s", userName, strings.Join(trackNames, "; ")))
}

// handleCurrentSong shows the currently playing track
func (rl *RewardListener) handleCurrentSong(userName string) {
	currentTrack, err := rl.spotifyClient.GetCurrentTrack()
	if err != nil || currentTrack.Item == nil {
		rl.sendMessage(fmt.Sprintf("@%s сорян, у меня какая-то ошибка произошла", userName))
		return
	}

	songName := spotify.SongItemToReadable(currentTrack.Item)
	rl.sendMessage(fmt.Sprintf("@%s текущий трек %s", userName, songName))
}

// handleSongQueue shows the current queue
func (rl *RewardListener) handleSongQueue(userName string) {
	queueData, err := rl.GetQueueData()
	if err != nil {
		log.Printf("Error getting queue data: %v", err)
		rl.sendMessage(fmt.Sprintf("@%s произошла ошибка при получении очереди", userName))
		return
	}

	var prettyQueue []string
	for _, item := range queueData.Queue {
		songName := spotify.SongItemToReadable(&item)
		prettyQueue = append(prettyQueue, songName)
	}

	if len(prettyQueue) > 5 {
		rl.sendMessage(fmt.Sprintf("@%s текущая очередь треков: %s. https://catjammusic.com/queue/%d",
			userName, strings.Join(prettyQueue[:5], "; "), rl.streamer.ID))
	} else {
		rl.sendMessage(fmt.Sprintf("@%s текущая очередь треков: %s", userName, strings.Join(prettyQueue, "; ")))
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

// StartPeriodicCleanup starts background cleanup tasks
func (rl *RewardListener) StartPeriodicCleanup() {
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
