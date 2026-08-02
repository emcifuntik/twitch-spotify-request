package handlers

import (
	"testing"

	spotifylib "github.com/zmb3/spotify/v2"
)

func TestParseSpotifySearchType(t *testing.T) {
	got, err := parseSpotifySearchType("artist, track")
	if err != nil {
		t.Fatalf("parseSpotifySearchType: %v", err)
	}
	want := spotifylib.SearchType(spotifylib.SearchTypeArtist | spotifylib.SearchTypeTrack)
	if got != want {
		t.Fatalf("search type = %v, want %v", got, want)
	}

	if _, err := parseSpotifySearchType("album"); err == nil {
		t.Fatal("unsupported search type did not return an error")
	}
}
