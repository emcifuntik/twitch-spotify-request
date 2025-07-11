package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/emcifuntik/twitch-spotify-request/internal/twitch"
	"github.com/gorilla/mux"
)

func RegisterRoutes(r *mux.Router) {
	// Register the routes for the application.

	r.HandleFunc("/auth", AuthHandler).Methods("GET")
	r.HandleFunc("/oauth/twitch", TwitchOAuthCallbackHandler).Methods("GET")
	r.HandleFunc("/oauth/spotify", SpotifyOAuthCallbackHandler).Methods("GET")
	r.HandleFunc("/login", LoginHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", LogoutHandler).Methods("POST")
	r.HandleFunc("/refresh", RefreshTokenHandler).Methods("POST")

	// Twitch event handler
	eventSubHandler, err := twitch.InitTwitchEventSub()
	if err != nil {
		panic(err) // Handle error appropriately in production code.
	}
	r.HandleFunc("/eventsub", eventSubHandler).Methods("POST")

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Public routes (no auth required)
	api.HandleFunc("/streamers", GetStreamers).Methods("GET")
	api.HandleFunc("/streamer/{streamerID}/queue", GetPublicQueue).Methods("GET")
	api.HandleFunc("/debug", DebugAPI).Methods("GET")

	// Auth routes (require authentication)
	authAPI := api.PathPrefix("").Subrouter()
	authAPI.Use(AuthMiddleware)
	authAPI.HandleFunc("/user/current", GetCurrentUser).Methods("GET")

	// User-specific routes (require authentication and user validation)
	userAPI := authAPI.PathPrefix("/user/{userID}").Subrouter()
	userAPI.Use(UserValidationMiddleware)
	userAPI.HandleFunc("/profile", GetUserProfile).Methods("GET")
	userAPI.HandleFunc("/queue", GetQueue).Methods("GET")
	userAPI.HandleFunc("/settings", UpdateUserSettings).Methods("POST", "PUT")
	userAPI.HandleFunc("/fix-rewards", FixRewards).Methods("POST")

	// New settings and blocks endpoints
	userAPI.HandleFunc("/config", GetSettings).Methods("GET")
	userAPI.HandleFunc("/config", UpdateSettings).Methods("POST", "PUT")
	userAPI.HandleFunc("/blocks", GetBlocks).Methods("GET")
	userAPI.HandleFunc("/blocks", AddBlock).Methods("POST")
	userAPI.HandleFunc("/blocks/{blockID}", RemoveBlock).Methods("DELETE")
	userAPI.HandleFunc("/spotify/search", SpotifySearch).Methods("GET")

	// Enable CORS for all API routes
	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})
	// Static file serving for React build assets
	staticDir := filepath.Join("web", "static")
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Join(staticDir, "assets")))))

	// Serve all other static files (like images, manifest, etc.)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Catch-all for SPA routing (must be last)
	r.PathPrefix("/").HandlerFunc(ServeIndex)
}

// ServeIndex serves the main index.html for SPA routing
func ServeIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join("web", "static", "index.html")
	http.ServeFile(w, r, indexPath)
}
