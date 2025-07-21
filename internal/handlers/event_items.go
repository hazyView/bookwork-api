package handlers

import (
	"context"
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

type EventItemHandler struct {
	db *database.DB
}

func NewEventItemHandler(db *database.DB) *EventItemHandler {
	return &EventItemHandler{db: db}
}

func (h *EventItemHandler) GetItems(w http.ResponseWriter, r *http.Request) {
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
		SELECT id, event_id, name, category, assigned_to, status, notes, created_by, created_at, updated_at
		FROM event_items
		WHERE event_id = $1
		ORDER BY created_at ASC`

	rows, err := h.db.QueryContext(r.Context(), query, eventID)
	if err != nil {
		log.Printf("Error querying event items: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get items", nil)
		return
	}
	defer rows.Close()

	var items []models.EventItem
	for rows.Next() {
		var item models.EventItem

		err := rows.Scan(
			&item.ID, &item.EventID, &item.Name, &item.Category,
			&item.AssignedTo, &item.Status, &item.Notes, &item.CreatedBy,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning item: %v", err)
			continue
		}

		items = append(items, item)
	}

	// Transform items to frontend format
	var frontendItems []*models.FrontendEventItem
	for _, item := range items {
		frontendItems = append(frontendItems, item.ToFrontendFormat())
	}

	response := map[string]interface{}{
		"items": frontendItems,
	}

	h.writeSuccessResponse(w, response, "Items retrieved successfully")
}

func (h *EventItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
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

	// Check if user can manage items for this event
	if !h.canManageEventItems(r.Context(), eventID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	var req models.CreateEventItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Validate required fields
	if req.Item.Name == "" || req.Item.Category == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Name and category are required", nil)
		return
	}

	// Validate category
	validCategories := []string{"food", "materials", "logistics", "discussion", "presentation", "other"}
	if !h.contains(validCategories, req.Item.Category) {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid category", nil)
		return
	}

	// Create item
	itemID := uuid.New()
	query := `
		INSERT INTO event_items (id, event_id, name, category, assigned_to, status, notes, created_by) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = h.db.ExecContext(r.Context(), query,
		itemID, eventID, req.Item.Name, req.Item.Category,
		req.Item.AssignedTo, "pending", req.Item.Notes, userID,
	)
	if err != nil {
		log.Printf("Error creating event item: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create item", nil)
		return
	}

	item := &models.EventItem{
		ID:         itemID,
		EventID:    eventID,
		Name:       req.Item.Name,
		Category:   req.Item.Category,
		AssignedTo: req.Item.AssignedTo,
		Status:     "pending",
		Notes:      req.Item.Notes,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
	}

	response := map[string]interface{}{
		"item": item,
	}

	w.WriteHeader(http.StatusCreated)
	h.writeSuccessResponse(w, response, "Item created successfully")
}

func (h *EventItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid event ID", nil)
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid item ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check permissions
	if !h.canManageEventItems(r.Context(), eventID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	var req models.UpdateEventItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Build update query
	setParts := []string{}
	args := []interface{}{}
	argCount := 0

	if req.Status != "" {
		validStatuses := []string{"pending", "assigned", "confirmed", "completed"}
		if h.contains(validStatuses, req.Status) {
			argCount++
			setParts = append(setParts, "status = $"+strconv.Itoa(argCount))
			args = append(args, req.Status)
		}
	}

	if req.Notes != nil {
		argCount++
		setParts = append(setParts, "notes = $"+strconv.Itoa(argCount))
		args = append(args, *req.Notes)
	}

	if len(setParts) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "No fields to update", nil)
		return
	}

	argCount++
	args = append(args, itemID)
	argCount++
	args = append(args, eventID)

	query := `UPDATE event_items SET ` + strings.Join(setParts, ", ") + `, updated_at = NOW() WHERE id = $` + strconv.Itoa(argCount-1) + ` AND event_id = $` + strconv.Itoa(argCount)

	result, err := h.db.ExecContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error updating event item: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update item", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Item not found", nil)
		return
	}

	response := map[string]interface{}{
		"item": map[string]interface{}{
			"id":        itemID,
			"updatedAt": time.Now(),
		},
	}

	if req.Status != "" {
		response["item"].(map[string]interface{})["status"] = req.Status
	}
	if req.Notes != nil {
		response["item"].(map[string]interface{})["notes"] = *req.Notes
	}

	h.writeSuccessResponse(w, response, "Item updated successfully")
}

func (h *EventItemHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid event ID", nil)
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid item ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check permissions
	if !h.canManageEventItems(r.Context(), eventID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	query := `DELETE FROM event_items WHERE id = $1 AND event_id = $2`
	result, err := h.db.ExecContext(r.Context(), query, itemID, eventID)
	if err != nil {
		log.Printf("Error deleting event item: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete item", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Item not found", nil)
		return
	}

	response := map[string]string{
		"message": "Item deleted successfully",
	}

	h.writeSuccessResponse(w, response, "Item deleted successfully")
}

// Helper methods
func (h *EventItemHandler) canAccessEvent(ctx context.Context, eventID, userID uuid.UUID) bool {
	query := `
		SELECT 1 FROM events e
		JOIN club_members cm ON e.club_id = cm.club_id
		WHERE e.id = $1 AND cm.user_id = $2 AND cm.is_active = true`

	var exists int
	err := h.db.QueryRowContext(ctx, query, eventID, userID).Scan(&exists)
	return err == nil
}

func (h *EventItemHandler) canManageEventItems(ctx context.Context, eventID, userID uuid.UUID) bool {
	query := `
		SELECT cm.role, e.created_by FROM events e
		JOIN club_members cm ON e.club_id = cm.club_id
		WHERE e.id = $1 AND cm.user_id = $2 AND cm.is_active = true`

	var role string
	var createdBy uuid.UUID
	err := h.db.QueryRowContext(ctx, query, eventID, userID).Scan(&role, &createdBy)
	if err != nil {
		return false
	}

	return role == "owner" || role == "moderator" || createdBy == userID
}

func (h *EventItemHandler) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (h *EventItemHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")

	response := models.NewAPIResponse(true, data, message)
	json.NewEncoder(w).Encode(response)
}

func (h *EventItemHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
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
