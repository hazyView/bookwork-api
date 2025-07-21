package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"bookwork-api/internal/auth"
	"bookwork-api/internal/database"
	"bookwork-api/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AvailabilityHandler struct {
	db *database.DB
}

func NewAvailabilityHandler(db *database.DB) *AvailabilityHandler {
	return &AvailabilityHandler{db: db}
}

func (h *AvailabilityHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid event ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check if user can access this event
	if !h.canAccessEvent(r.Context(), eventID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil)
		return
	}

	query := `
		SELECT user_id, status, notes, updated_at
		FROM availability
		WHERE event_id = $1
		ORDER BY updated_at DESC`

	rows, err := h.db.QueryContext(r.Context(), query, eventID)
	if err != nil {
		log.Printf("Error querying availability: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get availability", nil)
		return
	}
	defer rows.Close()

	availability := make(map[string]*models.Availability)
	summary := &models.AvailabilitySummary{}

	for rows.Next() {
		var avail models.Availability
		avail.EventID = eventID

		err := rows.Scan(&avail.UserID, &avail.Status, &avail.Notes, &avail.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning availability: %v", err)
			continue
		}

		availability[avail.UserID.String()] = &avail

		// Update summary
		switch avail.Status {
		case "available":
			summary.Available++
		case "maybe":
			summary.Maybe++
		case "unavailable":
			summary.Unavailable++
		}
		summary.Total++
	}

	// Transform availability to frontend format
	frontendAvailability := make(map[string]*models.FrontendAvailability)
	for userID, avail := range availability {
		frontendAvailability[userID] = avail.ToFrontendFormat()
	}

	// Return the availability map directly as expected by frontend
	h.writeSuccessResponse(w, frontendAvailability, "Availability retrieved successfully")
}

func (h *AvailabilityHandler) UpdateAvailability(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid event ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check if user can access this event
	if !h.canAccessEvent(r.Context(), eventID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil)
		return
	}

	var req models.AvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Validate status
	validStatuses := []string{"available", "maybe", "unavailable"}
	if !h.contains(validStatuses, req.Status) {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid status. Must be 'available', 'maybe', or 'unavailable'", nil)
		return
	}

	// Use the requesting user's ID for the availability update
	requestUserID := userID
	if req.UserID != uuid.Nil {
		requestUserID = req.UserID
	}

	// Check if user can update availability for the specified user
	if requestUserID != userID {
		// Only owners and moderators can update availability for other users
		if !h.canManageEvent(r.Context(), eventID, userID) {
			h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Cannot update availability for other users", nil)
			return
		}
	}

	// Upsert availability
	query := `
		INSERT INTO availability (id, event_id, user_id, status, notes, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
		ON CONFLICT (event_id, user_id)
		DO UPDATE SET status = $3, notes = $4, updated_at = NOW()`

	_, err = h.db.ExecContext(r.Context(), query, eventID, requestUserID, req.Status, req.Notes)
	if err != nil {
		log.Printf("Error updating availability: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update availability", nil)
		return
	}

	availability := &models.Availability{
		EventID:   eventID,
		UserID:    requestUserID,
		Status:    req.Status,
		Notes:     req.Notes,
		UpdatedAt: time.Now(),
	}

	response := map[string]interface{}{
		"availability": availability,
	}

	h.writeSuccessResponse(w, response, "Availability updated successfully")
}

// Helper methods
func (h *AvailabilityHandler) canAccessEvent(ctx context.Context, eventID, userID uuid.UUID) bool {
	query := `
		SELECT 1 FROM events e
		JOIN club_members cm ON e.club_id = cm.club_id
		WHERE e.id = $1 AND cm.user_id = $2 AND cm.is_active = true`

	var exists int
	err := h.db.QueryRowContext(ctx, query, eventID, userID).Scan(&exists)
	return err == nil
}

func (h *AvailabilityHandler) canManageEvent(ctx context.Context, eventID, userID uuid.UUID) bool {
	query := `
		SELECT cm.role FROM events e
		JOIN club_members cm ON e.club_id = cm.club_id
		WHERE e.id = $1 AND cm.user_id = $2 AND cm.is_active = true`

	var role string
	err := h.db.QueryRowContext(ctx, query, eventID, userID).Scan(&role)
	if err != nil {
		return false
	}

	return role == "owner" || role == "moderator"
}

func (h *AvailabilityHandler) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (h *AvailabilityHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")

	response := models.NewAPIResponse(true, data, message)
	json.NewEncoder(w).Encode(response)
}

func (h *AvailabilityHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := &models.FrontendErrorResponse{
		Error:      code,
		Message:    message,
		StatusCode: statusCode,
		Details:    details,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}
