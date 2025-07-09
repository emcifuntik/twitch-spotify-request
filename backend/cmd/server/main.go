package main

import (
	"log"
	"net/http"
	"os"

	"github.com/emcifuntik/twitch-spotify-request/internal/db"
	"github.com/emcifuntik/twitch-spotify-request/internal/handlers"
	"github.com/emcifuntik/twitch-spotify-request/internal/twitch"
	"github.com/gorilla/mux"
)

func main() {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306" // Default MySQL port
	}
	dbName := "twspoty"

	db.InitDB(dbUser, dbPassword, dbHost, dbPort, dbName)

	// Initialize cooldown manager
	twitch.GetCooldownManager().StartPeriodicCleanup()

	twitch.InitTwitchWhClient()
	twitch.StartTwitchHandlers()

	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
