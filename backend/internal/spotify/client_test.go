package spotify

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	spotifylib "github.com/zmb3/spotify/v2"
)

func TestRefreshTokenKeepsExistingRefreshTokenWhenSpotifyOmitsReplacement(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "client-id")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "client-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Basic "+base64.StdEncoding.EncodeToString([]byte("client-id:client-secret")); got != want {
			t.Errorf("Authorization header = %q, want %q", got, want)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q", got)
		}
		if got := r.Form.Get("refresh_token"); got != "old-refresh" {
			t.Errorf("refresh_token = %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access","token_type":"Bearer","expires_in":3600}`))
	}))
	defer server.Close()

	var savedAccess, savedRefresh string
	client := newSpotifyClient(
		"old-access",
		"old-refresh",
		func(accessToken, refreshToken string) error {
			savedAccess = accessToken
			savedRefresh = refreshToken
			return nil
		},
		func() error {
			t.Fatal("invalid-token callback was called")
			return nil
		},
		server.URL,
		server.Client(),
	)

	if err := client.refreshToken(); err != nil {
		t.Fatalf("refreshToken: %v", err)
	}
	if got := client.currentToken.AccessToken; got != "new-access" {
		t.Fatalf("access token = %q", got)
	}
	if got := client.currentToken.RefreshToken; got != "old-refresh" {
		t.Fatalf("refresh token = %q", got)
	}
	if savedAccess != "new-access" || savedRefresh != "old-refresh" {
		t.Fatalf("saved tokens = (%q, %q)", savedAccess, savedRefresh)
	}
}

func TestRefreshTokenStoresReplacementRefreshToken(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "client-id")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "client-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access","token_type":"Bearer","expires_in":3600,"refresh_token":"new-refresh"}`))
	}))
	defer server.Close()

	var savedRefresh string
	client := newSpotifyClient(
		"old-access",
		"old-refresh",
		func(_, refreshToken string) error {
			savedRefresh = refreshToken
			return nil
		},
		nil,
		server.URL,
		server.Client(),
	)

	if err := client.refreshToken(); err != nil {
		t.Fatalf("refreshToken: %v", err)
	}
	if got := client.currentToken.RefreshToken; got != "new-refresh" {
		t.Fatalf("refresh token = %q", got)
	}
	if savedRefresh != "new-refresh" {
		t.Fatalf("saved refresh token = %q", savedRefresh)
	}
}

func TestRefreshTokenInvalidGrantRequiresReauthorization(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "client-id")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "client-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Refresh token expired"}`))
	}))
	defer server.Close()

	invalidated := false
	client := newSpotifyClient(
		"old-access",
		"expired-refresh",
		func(_, _ string) error {
			t.Fatal("refresh callback was called")
			return nil
		},
		func() error {
			invalidated = true
			return nil
		},
		server.URL,
		server.Client(),
	)

	err := client.refreshToken()
	if !errors.Is(err, ErrReauthorizationRequired) {
		t.Fatalf("refreshToken error = %v, want ErrReauthorizationRequired", err)
	}
	if !invalidated {
		t.Fatal("invalid-token callback was not called")
	}
	if client.currentToken.AccessToken != "" || client.currentToken.RefreshToken != "" {
		t.Fatalf("tokens were not discarded: %#v", client.currentToken)
	}
}

func TestExecuteWithRetryPreservesReauthorizationError(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "client-id")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "client-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()

	client := newSpotifyClient("old-access", "expired-refresh", nil, nil, server.URL, server.Client())
	err := client.executeWithRetry(func() error {
		return errors.New("spotify returned 401 unauthorized")
	})

	if !errors.Is(err, ErrReauthorizationRequired) {
		t.Fatalf("executeWithRetry error = %v, want ErrReauthorizationRequired", err)
	}
}

func TestIsUnauthorizedUsesSpotifyStatus(t *testing.T) {
	if !isUnauthorized(spotifylib.Error{Status: http.StatusUnauthorized, Message: "expired"}) {
		t.Fatal("Spotify 401 was not recognized")
	}
	if isUnauthorized(spotifylib.Error{Status: http.StatusForbidden, Message: "forbidden"}) {
		t.Fatal("Spotify 403 was recognized as an expired access token")
	}
}
