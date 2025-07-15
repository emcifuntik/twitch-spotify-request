package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/emcifuntik/twitch-spotify-request/internal/db"
	"github.com/gorilla/mux"
)

// GetCommands returns all commands for a user
func GetCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Get streamer ID
	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Get commands
	commands, err := db.GetStreamerCommands(database, streamer.ID)
	if err != nil {
		log.Printf("Error getting commands for user %s: %v", userID, err)
		writeAPIError(w, "Failed to get commands", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var response []CommandResponse
	for _, cmd := range commands {
		response = append(response, CommandResponse{
			ID:        cmd.ID,
			Type:      cmd.Type,
			Name:      cmd.Name,
			IsEnabled: cmd.IsEnabled,
		})
	}

	writeAPISuccess(w, response)
}

// UpdateCommand updates a command for a user
func UpdateCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Type == "" || req.Name == "" {
		writeAPIError(w, "Type and name are required", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Get streamer ID
	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Update command
	err := db.CreateOrUpdateCommand(database, streamer.ID, req.Type, req.Name, req.IsEnabled)
	if err != nil {
		log.Printf("Error updating command for user %s: %v", userID, err)
		writeAPIError(w, "Failed to update command", http.StatusInternalServerError)
		return
	}

	writeAPISuccess(w, map[string]string{"message": "Command updated successfully"})
}

// ToggleRequestMode toggles between commands and rewards for a user
func ToggleRequestMode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var req RequestModeToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Check if user can use rewards if they're trying to switch to rewards
	if !req.UseCommands {
		canUseRewards, err := db.CanUseRewards(database, userID)
		if err != nil {
			log.Printf("Error checking reward eligibility for user %s: %v", userID, err)
			writeAPIError(w, "Failed to check reward eligibility", http.StatusInternalServerError)
			return
		}

		if !canUseRewards {
			writeAPIError(w, "Only Twitch Affiliates and Partners can use channel point rewards", http.StatusForbidden)
			return
		}
	}

	// Update use_commands flag
	err := db.UpdateStreamerUseCommands(database, userID, req.UseCommands)
	if err != nil {
		log.Printf("Error updating request mode for user %s: %v", userID, err)
		writeAPIError(w, "Failed to update request mode", http.StatusInternalServerError)
		return
	}

	writeAPISuccess(w, map[string]string{"message": "Request mode updated successfully"})
}

// InitializeCommands initializes default commands for a user
func InitializeCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		writeAPIError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	database := db.GetDB()
	if database == nil {
		writeAPIError(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Get streamer ID
	var streamer db.Streamer
	result := database.Where("streamer_channel_id = ?", userID).First(&streamer)
	if result.Error != nil {
		writeAPIError(w, "User not found", http.StatusNotFound)
		return
	}

	// Initialize default commands
	err := db.InitializeDefaultCommands(database, streamer.ID)
	if err != nil {
		log.Printf("Error initializing commands for user %s: %v", userID, err)
		writeAPIError(w, "Failed to initialize commands", http.StatusInternalServerError)
		return
	}

	writeAPISuccess(w, map[string]string{"message": "Commands initialized successfully"})
}
