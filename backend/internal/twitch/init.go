package twitch

import (
	"log"

	"github.com/emcifuntik/twitch-spotify-request/internal/db"
)

func StartTwitchHandlers() {
	database := db.GetDB()
	if database == nil {
		panic("Database is not initialized")
	}

	var streamers []db.Streamer

	result := database.Find(&streamers)
	if result.Error != nil {
		log.Printf("Failed to load streamers: %v", result.Error)
		return
	}

	for _, streamer := range streamers {
		NewRewardListener(&streamer)
	}
}
