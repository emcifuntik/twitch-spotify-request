package spotify

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

// SpotifyClient wraps the zmb3/spotify client with our custom logic
type SpotifyClient struct {
	client         *spotify.Client
	onTokenRefresh func(accessToken, refreshToken string) error
	auth           *spotifyauth.Authenticator
	currentToken   *oauth2.Token
}

// NewSpotifyClient creates a new Spotify API client using zmb3/spotify with proper token refresh
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

	return &SpotifyClient{
		client:         client,
		onTokenRefresh: onTokenRefresh,
		auth:           auth,
		currentToken:   token,
	}
}

// refreshTokenIfNeeded checks if token needs refresh and refreshes it
func (s *SpotifyClient) refreshTokenIfNeeded() error {
	if s.currentToken == nil || s.currentToken.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Check if token is expired or about to expire
	if s.currentToken.Valid() {
		return nil // Token is still valid
	}

	log.Printf("Refreshing Spotify token...")

	// Use the authenticator to refresh the token
	newToken, err := s.auth.RefreshToken(context.Background(), s.currentToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update our stored token
	s.currentToken = newToken

	// Call the callback if provided
	if s.onTokenRefresh != nil {
		refreshToken := ""
		if newToken.RefreshToken != "" {
			refreshToken = newToken.RefreshToken
		}

		log.Printf("Spotify token refreshed - calling update callback")
		if err := s.onTokenRefresh(newToken.AccessToken, refreshToken); err != nil {
			log.Printf("Error calling token refresh callback: %v", err)
		}
	}

	// Update the HTTP client with the new token
	httpClient := s.auth.Client(context.Background(), newToken)
	s.client = spotify.New(httpClient)

	return nil
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

	// Try the search operation
	results, err := s.client.Search(ctx, query, spotify.SearchTypeTrack, spotify.Limit(1))
	if err != nil {
		// Check if this is a token expiration error and try to refresh
		if s.isTokenExpiredError(err) {
			log.Printf("Token expired during search, attempting refresh...")
			if refreshErr := s.refreshTokenIfNeeded(); refreshErr != nil {
				log.Printf("Failed to refresh token: %v", refreshErr)
				return nil, fmt.Errorf("failed to search tracks (token refresh failed): %w", err)
			}
			// Retry the search with refreshed token
			results, err = s.client.Search(ctx, query, spotify.SearchTypeTrack, spotify.Limit(1))
		}

		if err != nil {
			return nil, fmt.Errorf("failed to search tracks: %w", err)
		}
	}
	return results, nil
}

// isTokenExpiredError checks if the error is due to an expired access token
func (s *SpotifyClient) isTokenExpiredError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "access token expired") ||
		strings.Contains(errStr, "token expired") ||
		strings.Contains(errStr, "invalid access token") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "401")
}

// EnqueueTrack adds a track to the Spotify queue
func (s *SpotifyClient) EnqueueTrack(trackURI spotify.URI) error {
	ctx := context.Background()

	// Extract the track ID from the URI
	// URI format is "spotify:track:ID", we need just the ID part
	uriStr := string(trackURI)
	parts := strings.Split(uriStr, ":")
	if len(parts) != 3 || parts[0] != "spotify" || parts[1] != "track" {
		return fmt.Errorf("invalid track URI format: %s", uriStr)
	}

	trackID := spotify.ID(parts[2])
	log.Printf("Enqueueing track - URI: %s, ID: %s", uriStr, trackID)

	err := s.client.QueueSong(ctx, trackID)
	if err != nil {
		// Check if this is a token expiration error and try to refresh
		if s.isTokenExpiredError(err) {
			log.Printf("Token expired during enqueue, attempting refresh...")
			if refreshErr := s.refreshTokenIfNeeded(); refreshErr != nil {
				log.Printf("Failed to refresh token: %v", refreshErr)
				return fmt.Errorf("failed to enqueue track (token refresh failed): %w", err)
			}
			// Retry the enqueue with refreshed token
			err = s.client.QueueSong(ctx, trackID)
		}

		if err != nil {
			return fmt.Errorf("failed to enqueue track: %w", err)
		}
	}
	return nil
}

// NextTrack skips to the next track
func (s *SpotifyClient) NextTrack() error {
	ctx := context.Background()
	err := s.client.Next(ctx)
	if err != nil {
		return fmt.Errorf("failed to skip track: %w", err)
	}
	return nil
}

// GetCurrentTrack gets the currently playing track
func (s *SpotifyClient) GetCurrentTrack() (*spotify.CurrentlyPlaying, error) {
	ctx := context.Background()

	current, err := s.client.PlayerCurrentlyPlaying(ctx)
	if err != nil {
		// Check if this is a token expiration error and try to refresh
		if s.isTokenExpiredError(err) {
			log.Printf("Token expired during get current track, attempting refresh...")
			if refreshErr := s.refreshTokenIfNeeded(); refreshErr != nil {
				log.Printf("Failed to refresh token: %v", refreshErr)
				return nil, fmt.Errorf("failed to get current track (token refresh failed): %w", err)
			}
			// Retry with refreshed token
			current, err = s.client.PlayerCurrentlyPlaying(ctx)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to get current track: %w", err)
		}
	}
	return current, nil
}

// GetTrackByID gets track information by Spotify track ID
func (s *SpotifyClient) GetTrackByID(trackID string) (*spotify.FullTrack, error) {
	ctx := context.Background()
	track, err := s.client.GetTrack(ctx, spotify.ID(trackID))
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	return track, nil
}

// GetRecentlyPlayed gets recently played tracks
func (s *SpotifyClient) GetRecentlyPlayed(limit int) ([]spotify.RecentlyPlayedItem, error) {
	ctx := context.Background()

	items, err := s.client.PlayerRecentlyPlayed(ctx)
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
	ctx := context.Background()
	err := s.client.Volume(ctx, volume)
	if err != nil {
		return fmt.Errorf("failed to set volume: %w", err)
	}
	return nil
}

// GetQueue gets the user's queue from Spotify
func (s *SpotifyClient) GetQueue() (*spotify.Queue, error) {
	ctx := context.Background()

	queue, err := s.client.GetQueue(ctx)
	if err != nil {
		// Check if this is a token expiration error and try to refresh
		if s.isTokenExpiredError(err) {
			log.Printf("Token expired during get queue, attempting refresh...")
			if refreshErr := s.refreshTokenIfNeeded(); refreshErr != nil {
				log.Printf("Failed to refresh token: %v", refreshErr)
				return nil, fmt.Errorf("failed to get queue (token refresh failed): %w", err)
			}
			// Retry with refreshed token
			queue, err = s.client.GetQueue(ctx)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to get queue: %w", err)
		}
	}
	return queue, nil
}
