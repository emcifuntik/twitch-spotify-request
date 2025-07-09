package spotify

import (
	"regexp"
)

var spotifyURLRegex = regexp.MustCompile(`https://open\.spotify\.com/track/([0-9A-Za-z]+)(\?.+)?`)

// IsSpotifyURL checks if the given string is a Spotify track URL
func IsSpotifyURL(url string) bool {
	return spotifyURLRegex.MatchString(url)
}

// GetTrackIDFromURL extracts the track ID from a Spotify URL
func GetTrackIDFromURL(url string) string {
	matches := spotifyURLRegex.FindStringSubmatch(url)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
