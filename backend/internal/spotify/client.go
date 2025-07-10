package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

// TokenRefreshResponse represents the response from Spotify's token refresh endpoint
type TokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

// SpotifyClient wraps the zmb3/spotify client with custom token refresh
type SpotifyClient struct {
	client         *spotify.Client
	onTokenRefresh func(accessToken, refreshToken string) error
	auth           *spotifyauth.Authenticator
	currentToken   *oauth2.Token
	clientID       string
	clientSecret   string
	mutex          sync.RWMutex
}

// NewSpotifyClient creates a new Spotify API client using zmb3/spotify with custom token refresh
func NewSpotifyClient(accessToken, refreshToken string, onTokenRefresh func(string, string) error) *SpotifyClient {
	// Get client credentials from environment
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Printf("Warning: Spotify client credentials not found in environment variables")
	}

	// Create Spotify authenticator
	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserModifyPlaybackState,
			spotifyauth.ScopeUserReadCurrentlyPlaying,
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopeUserReadRecentlyPlayed,
		),
	)

	// Create initial OAuth2 token
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}

	// Create HTTP client with the token
	httpClient := auth.Client(context.Background(), token)

	// Create Spotify client
	client := spotify.New(httpClient)

	log.Printf("Spotify client created with custom token refresh")

	return &SpotifyClient{
		client:         client,
		onTokenRefresh: onTokenRefresh,
		auth:           auth,
		currentToken:   token,
		clientID:       clientID,
		clientSecret:   clientSecret,
	}
}

// refreshToken performs a custom token refresh using HTTP request like your CURL example
func (s *SpotifyClient) refreshToken() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.Printf("Refreshing Spotify token using custom method...")

	// Create the authorization header (base64 encoded client_id:client_secret)
	auth := base64.StdEncoding.EncodeToString([]byte(s.clientID + ":" + s.clientSecret))

	// Prepare the request body
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", s.currentToken.RefreshToken)

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "twitch-spotify-request/1.0")

	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var tokenResp TokenRefreshResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	log.Printf("Token refresh successful - new access token: %s", truncateToken(tokenResp.AccessToken))

	// Update the current token
	newToken := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	// If no new refresh token was provided, keep the old one
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = s.currentToken.RefreshToken
	}

	s.currentToken = newToken

	// Update the HTTP client with the new token
	httpClient := s.auth.Client(context.Background(), newToken)
	s.client = spotify.New(httpClient)

	// Call the callback to update the database
	if s.onTokenRefresh != nil {
		if err := s.onTokenRefresh(newToken.AccessToken, newToken.RefreshToken); err != nil {
			log.Printf("Error calling token refresh callback: %v", err)
			return fmt.Errorf("token refresh callback failed: %w", err)
		}
	}

	log.Printf("Token refresh completed successfully")
	return nil
}

// executeWithRetry executes a function and retries with token refresh on 401 errors
func (s *SpotifyClient) executeWithRetry(fn func() error) error {
	err := fn()
	if err != nil {
		// Check if it's a 401 error
		if strings.Contains(strings.ToLower(err.Error()), "401") ||
			strings.Contains(strings.ToLower(err.Error()), "unauthorized") ||
			strings.Contains(strings.ToLower(err.Error()), "invalid access token") ||
			strings.Contains(strings.ToLower(err.Error()), "token expired") {
			
			log.Printf("Detected 401/unauthorized error, attempting token refresh...")
			
			// Refresh the token
			if refreshErr := s.refreshToken(); refreshErr != nil {
				log.Printf("Token refresh failed: %v", refreshErr)
				return fmt.Errorf("original error: %w, refresh error: %v", err, refreshErr)
			}
			
			// Retry the original function
			log.Printf("Retrying operation after token refresh...")
			return fn()
		}
	}
	return err
}

// truncateToken truncates a token for logging (security)
func truncateToken(token string) string {
	if len(token) > 8 {
		return token[:8] + "..."
	}
	return token
}

// SongItemToReadable converts a Spotify track to a readable format
func SongItemToReadable(track *spotify.FullTrack) string {
	if track == nil {
		return ""
	}

	var artistNames []string
	for _, artist := range track.Artists {
		artistNames = append(artistNames, artist.Name)
	}

	leftPart := strings.Join(artistNames, " & ")
	rightPart := track.Name
	return leftPart + " - " + rightPart
}

// SongItemToReadableSimple converts a SimpleTrack to a readable format
func SongItemToReadableSimple(track *spotify.SimpleTrack) string {
	if track == nil {
		return ""
	}

	var artistNames []string
	for _, artist := range track.Artists {
		artistNames = append(artistNames, artist.Name)
	}

	leftPart := strings.Join(artistNames, " & ")
	rightPart := track.Name
	return leftPart + " - " + rightPart
}

// SearchTracks searches for tracks on Spotify
func (s *SpotifyClient) SearchTracks(query string) (*spotify.SearchResult, error) {
	ctx := context.Background()
	var results *spotify.SearchResult

	err := s.executeWithRetry(func() error {
		var err error
		results, err = s.client.Search(ctx, query, spotify.SearchTypeTrack, spotify.Limit(1))
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}
	return results, nil
}

// EnqueueTrack adds a track to the Spotify queue
func (s *SpotifyClient) EnqueueTrack(trackURI spotify.URI) error {
	ctx := context.Background()

	// Extract the track ID from the URI
	uriStr := string(trackURI)
	parts := strings.Split(uriStr, ":")
	if len(parts) != 3 || parts[0] != "spotify" || parts[1] != "track" {
		return fmt.Errorf("invalid track URI format: %s", uriStr)
	}

	trackID := spotify.ID(parts[2])
	log.Printf("Enqueueing track - URI: %s, ID: %s", uriStr, trackID)

	err := s.executeWithRetry(func() error {
		return s.client.QueueSong(ctx, trackID)
	})

	if err != nil {
		return fmt.Errorf("failed to enqueue track: %w", err)
	}
	return nil
}

// NextTrack skips to the next track
func (s *SpotifyClient) NextTrack() error {
	ctx := context.Background()
	
	err := s.executeWithRetry(func() error {
		return s.client.Next(ctx)
	})

	if err != nil {
		return fmt.Errorf("failed to skip track: %w", err)
	}
	return nil
}

// GetCurrentTrack gets the currently playing track
func (s *SpotifyClient) GetCurrentTrack() (*spotify.CurrentlyPlaying, error) {
	ctx := context.Background()
	var current *spotify.CurrentlyPlaying

	err := s.executeWithRetry(func() error {
		var err error
		current, err = s.client.PlayerCurrentlyPlaying(ctx)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get current track: %w", err)
	}
	return current, nil
}

// GetTrackByID gets track information by Spotify track ID
func (s *SpotifyClient) GetTrackByID(trackID string) (*spotify.FullTrack, error) {
	ctx := context.Background()
	var track *spotify.FullTrack

	err := s.executeWithRetry(func() error {
		var err error
		track, err = s.client.GetTrack(ctx, spotify.ID(trackID))
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	return track, nil
}

// GetRecentlyPlayed gets recently played tracks
func (s *SpotifyClient) GetRecentlyPlayed(limit int) ([]spotify.RecentlyPlayedItem, error) {
	ctx := context.Background()
	var items []spotify.RecentlyPlayedItem

	err := s.executeWithRetry(func() error {
		var err error
		items, err = s.client.PlayerRecentlyPlayed(ctx)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get recently played: %w", err)
	}

	// Limit the results if we got more than requested
	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// SetVolume sets the player volume
func (s *SpotifyClient) SetVolume(volume int) error {
	log.Printf("Setting Spotify volume to %d%% via API", volume)
	ctx := context.Background()

	err := s.executeWithRetry(func() error {
		return s.client.Volume(ctx, volume)
	})

	if err != nil {
		log.Printf("Spotify API error setting volume to %d%%: %v", volume, err)
		return fmt.Errorf("failed to set volume: %w", err)
	}

	log.Printf("Successfully set Spotify volume to %d%%", volume)
	return nil
}

// GetQueue gets the user's queue from Spotify
func (s *SpotifyClient) GetQueue() (*spotify.Queue, error) {
	ctx := context.Background()
	var queue *spotify.Queue

	err := s.executeWithRetry(func() error {
		var err error
		queue, err = s.client.GetQueue(ctx)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	return queue, nil
}
