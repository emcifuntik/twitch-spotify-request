package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/emcifuntik/twitch-spotify-request/internal/db"
	"github.com/emcifuntik/twitch-spotify-request/internal/service"
	"github.com/emcifuntik/twitch-spotify-request/internal/twitch"
	"golang.org/x/oauth2"
)

var (
	twitchConfig  *oauth2.Config
	spotifyConfig *oauth2.Config
)

func init() {
	botHost := os.Getenv("BOT_HOST")
	twitchConfig = &oauth2.Config{
		ClientID:     os.Getenv("TWITCH_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		RedirectURL:  botHost + "oauth/twitch",
		Scopes:       []string{"user:read:chat", "user:write:chat", "channel:bot", "user:bot", "channel:read:redemptions", "channel:manage:redemptions"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://id.twitch.tv/oauth2/authorize",
			TokenURL: "https://id.twitch.tv/oauth2/token",
		},
	}
	spotifyConfig = &oauth2.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURL:  botHost + "oauth/spotify",
		Scopes: []string{
			"user-modify-playback-state",
			"user-read-currently-playing",
			"user-read-playback-position",
			"user-read-playback-state",
			"user-read-recently-played",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	// Redirect to Twitch OAuth authorization endpoint.
	http.Redirect(w, r, twitchConfig.AuthCodeURL(""), http.StatusFound)
}

func TwitchOAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	token, err := twitchConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		http.Error(w, "Failed to create validation request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Validation request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Invalid token validation response", http.StatusInternalServerError)
		return
	}
	var valData struct {
		ClientID  string   `json:"client_id"`
		Login     string   `json:"login"`
		Scopes    []string `json:"scopes"`
		UserID    string   `json:"user_id"`
		ExpiresIn int      `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&valData); err != nil {
		http.Error(w, "Decoding validation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	spotifyState := makeRandomString(32)
	if err := db.CreateOrUpdateTwitchData(db.GetDB(), valData.UserID, valData.Login, token.AccessToken, token.RefreshToken, spotifyState); err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, spotifyConfig.AuthCodeURL(spotifyState), http.StatusFound)
}

func SpotifyOAuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	token, err := spotifyConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Spotify token exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		http.Error(w, "Invalid spotify token response", http.StatusBadRequest)
		return
	}
	if err := db.UpdateSpotifyTokensByState(db.GetDB(), state, token.AccessToken, token.RefreshToken); err != nil {
		http.Error(w, "Database update failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user info to pass to the redirect and initialize listeners
	streamer, err := db.GetStreamerBySpotifyState(db.GetDB(), state)
	if err == nil && streamer != nil {
		// Initialize reward and chat listeners for the newly authenticated user (idempotent)
		log.Printf("Initializing listeners for user: %s (%s)", streamer.Name, streamer.ChannelID)
		twitch.GetOrCreateRewardListener(streamer)

		// Generate JWT token
		token, err := service.GenerateToken(streamer.ChannelID, streamer.ChannelID, streamer.Name)
		if err != nil {
			log.Printf("Failed to generate JWT token: %v", err)
			http.Redirect(w, r, "/dashboard?auth=error", http.StatusFound)
			return
		}

		// Set JWT token as HTTP-only cookie
		cookie := &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   24 * 60 * 60, // 24 hours
		}
		http.SetCookie(w, cookie)

		// Redirect with user context
		http.Redirect(w, r, "/dashboard?auth=success&user="+streamer.ChannelID, http.StatusFound)
	} else {
		log.Printf("Failed to get streamer info after auth: %v", err)
		// Fallback redirect without user context
		http.Redirect(w, r, "/dashboard?auth=error", http.StatusFound)
	}
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string              `json:"token"`
	User  UserProfileResponse `json:"user"`
}

// LoginHandler handles login requests (can be used for refreshing tokens)
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// For now, redirect to Twitch OAuth
	AuthHandler(w, r)
}

// LogoutHandler handles logout requests
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
	}
	http.SetCookie(w, cookie)

	writeAPIResponse(w, map[string]string{"message": "Logged out successfully"})
}

// RefreshTokenHandler refreshes the JWT token
func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Get current token from cookie or header
	var tokenString string

	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			writeAPIError(w, "No token provided", http.StatusUnauthorized)
			return
		}
		tokenString = cookie.Value
	}

	// Refresh the token
	newToken, err := service.RefreshToken(tokenString)
	if err != nil {
		writeAPIError(w, "Failed to refresh token", http.StatusUnauthorized)
		return
	}

	// Set new token as cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    newToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 24 hours
	}
	http.SetCookie(w, cookie)

	writeAPIResponse(w, map[string]string{
		"token":   newToken,
		"message": "Token refreshed successfully",
	})
}

func makeRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	ret := make([]byte, n)
	for i := range ret {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			ret[i] = letters[0]
		} else {
			ret[i] = letters[num.Int64()]
		}
	}
	return string(ret)
}
