package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bookwork-api/internal/auth"
	"bookwork-api/internal/database"
	"bookwork-api/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EventHandler struct {
	db *database.DB
}

func NewEventHandler(db *database.DB) *EventHandler {
	return &EventHandler{db: db}
}

func (h *EventHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "clubId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid club ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check if user is a member of the club
	if !h.isClubMember(r.Context(), clubID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "You are not a member of this club", nil)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	eventType := r.URL.Query().Get("type")

	offset := (page - 1) * limit

	// Build query
	query := `
		SELECT id, club_id, title, description, event_date, event_time, location, 
		       book, type, max_attendees, is_public, created_by, attendees, created_at, updated_at
		FROM events
		WHERE club_id = $1`

	args := []interface{}{clubID}
	argCount := 1

	if from != "" {
		argCount++
		query += ` AND event_date >= $` + strconv.Itoa(argCount)
		args = append(args, from)
	}

	if to != "" {
		argCount++
		query += ` AND event_date <= $` + strconv.Itoa(argCount)
		args = append(args, to)
	}

	if eventType != "" {
		argCount++
		query += ` AND type = $` + strconv.Itoa(argCount)
		args = append(args, eventType)
	}

	query += ` ORDER BY event_date DESC, event_time DESC LIMIT $` + strconv.Itoa(argCount+1) + ` OFFSET $` + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error querying events: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get events", nil)
		return
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var attendees models.UUIDArray

		err := rows.Scan(
			&event.ID, &event.ClubID, &event.Title, &event.Description,
			&event.Date, &event.Time, &event.Location, &event.Book,
			&event.Type, &event.MaxAttendees, &event.IsPublic, &event.CreatedBy,
			&attendees, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}

		event.Attendees = attendees
		events = append(events, event)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM events WHERE club_id = $1`
	countArgs := []interface{}{clubID}

	if from != "" {
		countQuery += ` AND event_date >= $2`
		countArgs = append(countArgs, from)
	}

	var total int
	h.db.QueryRowContext(r.Context(), countQuery, countArgs...).Scan(&total)

	totalPages := (total + limit - 1) / limit

	// Transform events to frontend format
	var frontendEvents []*models.FrontendEvent
	for _, event := range events {
		frontendEvents = append(frontendEvents, event.ToFrontendFormat())
	}

	response := map[string]interface{}{
		"events": frontendEvents,
		"pagination": models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	h.writeSuccessResponse(w, response, "Events retrieved successfully")
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "clubId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid club ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check if user can create events in this club
	if !h.canManageEvents(r.Context(), clubID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Validate required fields
	if req.Title == "" || req.Date == "" || req.Time == "" || req.Location == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Title, date, time, and location are required", nil)
		return
	}

	// Validate date format and ensure it's in the future
	eventDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid date format. Use YYYY-MM-DD", nil)
		return
	}

	if eventDate.Before(time.Now().Truncate(24 * time.Hour)) {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Event date must be in the future", nil)
		return
	}

	// Validate time format
	if !h.isValidTimeFormat(req.Time) {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid time format. Use HH:MM", nil)
		return
	}

	// Validate event type
	validTypes := []string{"discussion", "meeting", "social", "author_event"}
	if !h.contains(validTypes, req.Type) {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid event type", nil)
		return
	}

	// Create event
	eventID := uuid.New()
	query := `
		INSERT INTO events (id, club_id, title, description, event_date, event_time, location, 
		                   book, type, max_attendees, is_public, created_by, attendees) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	attendees := models.UUIDArray{}
	_, err = h.db.ExecContext(r.Context(), query,
		eventID, clubID, req.Title, req.Description, req.Date, req.Time,
		req.Location, req.Book, req.Type, req.MaxAttendees, req.IsPublic,
		userID, attendees,
	)
	if err != nil {
		log.Printf("Error creating event: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create event", nil)
		return
	}

	event := &models.Event{
		ID:          eventID,
		ClubID:      clubID,
		Title:       req.Title,
		Description: req.Description,
		Date:        req.Date,
		Time:        req.Time,
		Location:    req.Location,
		Book:        req.Book,
		Type:        req.Type,
		Attendees:   attendees,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}

	response := map[string]interface{}{
		"event": event,
	}

	w.WriteHeader(http.StatusCreated)
	h.writeSuccessResponse(w, response, "Event created successfully")
}

func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
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

	// Get event to check ownership and club membership
	event, err := h.getEventByID(r.Context(), eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Event not found", nil)
			return
		}
		log.Printf("Error getting event: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get event", nil)
		return
	}

	// Check if user can update this event
	if !h.canManageEvents(r.Context(), event.ClubID, userID) && event.CreatedBy != userID {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Build update query dynamically
	setParts := []string{}
	args := []interface{}{}
	argCount := 0

	for key, value := range updates {
		switch key {
		case "title", "description", "location", "book":
			if str, ok := value.(string); ok && str != "" {
				argCount++
				setParts = append(setParts, key+" = $"+strconv.Itoa(argCount))
				args = append(args, str)
			}
		case "date":
			if str, ok := value.(string); ok {
				if _, err := time.Parse("2006-01-02", str); err == nil {
					argCount++
					setParts = append(setParts, "event_date = $"+strconv.Itoa(argCount))
					args = append(args, str)
				}
			}
		case "time":
			if str, ok := value.(string); ok && h.isValidTimeFormat(str) {
				argCount++
				setParts = append(setParts, "event_time = $"+strconv.Itoa(argCount))
				args = append(args, str)
			}
		}
	}

	if len(setParts) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "No valid fields to update", nil)
		return
	}

	argCount++
	args = append(args, eventID)

	query := `UPDATE events SET ` + strings.Join(setParts, ", ") + `, updated_at = NOW() WHERE id = $` + strconv.Itoa(argCount)

	_, err = h.db.ExecContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error updating event: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update event", nil)
		return
	}

	response := map[string]interface{}{
		"event": map[string]interface{}{
			"id":        eventID,
			"updatedAt": time.Now(),
		},
	}

	// Add updated fields to response
	for key, value := range updates {
		if key == "date" {
			response["event"].(map[string]interface{})["date"] = value
		} else if key == "time" {
			response["event"].(map[string]interface{})["time"] = value
		} else {
			response["event"].(map[string]interface{})[key] = value
		}
	}

	h.writeSuccessResponse(w, response, "Event updated successfully")
}

func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
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

	// Get event to check ownership and permissions
	event, err := h.getEventByID(r.Context(), eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Event not found", nil)
			return
		}
		log.Printf("Error getting event: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get event", nil)
		return
	}

	// Check permissions
	if !h.canManageEvents(r.Context(), event.ClubID, userID) && event.CreatedBy != userID {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	query := `DELETE FROM events WHERE id = $1`
	result, err := h.db.ExecContext(r.Context(), query, eventID)
	if err != nil {
		log.Printf("Error deleting event: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete event", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Event not found", nil)
		return
	}

	response := map[string]string{
		"message": "Event deleted successfully",
	}

	h.writeSuccessResponse(w, response, "Event deleted successfully")
}

// Helper methods
func (h *EventHandler) isClubMember(ctx context.Context, clubID, userID uuid.UUID) bool {
	query := `SELECT 1 FROM club_members WHERE club_id = $1 AND user_id = $2 AND is_active = true`
	var exists int
	err := h.db.QueryRowContext(ctx, query, clubID, userID).Scan(&exists)
	return err == nil
}

func (h *EventHandler) canManageEvents(ctx context.Context, clubID, userID uuid.UUID) bool {
	query := `SELECT role FROM club_members WHERE club_id = $1 AND user_id = $2 AND is_active = true`
	var role string
	err := h.db.QueryRowContext(ctx, query, clubID, userID).Scan(&role)
	if err != nil {
		return false
	}
	return role == "owner" || role == "moderator"
}

func (h *EventHandler) getEventByID(ctx context.Context, eventID uuid.UUID) (*models.Event, error) {
	query := `
		SELECT id, club_id, title, description, event_date, event_time, location, 
		       book, type, max_attendees, is_public, created_by, attendees, created_at, updated_at
		FROM events WHERE id = $1`

	var event models.Event
	var attendees models.UUIDArray

	err := h.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID, &event.ClubID, &event.Title, &event.Description,
		&event.Date, &event.Time, &event.Location, &event.Book,
		&event.Type, &event.MaxAttendees, &event.IsPublic, &event.CreatedBy,
		&attendees, &event.CreatedAt, &event.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	event.Attendees = attendees
	return &event, nil
}

func (h *EventHandler) isValidTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

func (h *EventHandler) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (h *EventHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")

	response := models.NewAPIResponse(true, data, message)
	json.NewEncoder(w).Encode(response)
}

func (h *EventHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
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
