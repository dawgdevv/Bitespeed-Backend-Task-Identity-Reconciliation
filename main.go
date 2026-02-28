package main

import (
	"log"
	"net/http"
	"os"

	"bitespeed/internal/database"
	"bitespeed/internal/handlers"

	"github.com/gorilla/mux"
)

func main() {
	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Get database path from environment or default to local file
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "./bitespeed.db"
	}

	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create handler
	identifyHandler := handlers.NewIdentifyHandler(db)

	// Setup router
	router := mux.NewRouter()
	router.HandleFunc("/identify", identifyHandler.Handle).Methods("POST")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	// Start server
	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
