package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"bitespeed/internal/database"
	"bitespeed/internal/models"
	"bitespeed/internal/service"
)

// IdentifyHandler handles the /identify endpoint
type IdentifyHandler struct {
	service *service.ReconciliationService
}

// NewIdentifyHandler creates a new identify handler
func NewIdentifyHandler(db *database.DB) *IdentifyHandler {
	return &IdentifyHandler{
		service: service.NewReconciliationService(db),
	}
}

// Handle processes the identify request
func (h *IdentifyHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.IdentifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request - at least one of email or phoneNumber must be provided
	if (req.Email == nil || *req.Email == "") && (req.PhoneNumber == nil || *req.PhoneNumber == "") {
		http.Error(w, "Either email or phoneNumber must be provided", http.StatusBadRequest)
		return
	}

	response, err := h.service.Identify(req)
	if err != nil {
		log.Printf("Error processing identify request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
