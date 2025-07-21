package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"bookwork-api/internal/auth"
	"bookwork-api/internal/database"
	"bookwork-api/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ClubHandler struct {
	db *database.DB
}

func NewClubHandler(db *database.DB) *ClubHandler {
	return &ClubHandler{db: db}
}

func (h *ClubHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
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

	role := r.URL.Query().Get("role")
	activeParam := r.URL.Query().Get("active")

	offset := (page - 1) * limit

	// Build query
	query := `
		SELECT cm.id, cm.club_id, cm.user_id, cm.role, cm.joined_date, cm.books_read, cm.is_active,
		       u.id, u.name, u.email, u.phone, u.avatar
		FROM club_members cm
		JOIN users u ON cm.user_id = u.id
		WHERE cm.club_id = $1`

	args := []interface{}{clubID}
	argCount := 1

	if role != "" {
		argCount++
		query += ` AND cm.role = $` + strconv.Itoa(argCount)
		args = append(args, role)
	}

	if activeParam != "" {
		active, _ := strconv.ParseBool(activeParam)
		argCount++
		query += ` AND cm.is_active = $` + strconv.Itoa(argCount)
		args = append(args, active)
	}

	query += ` ORDER BY cm.joined_date DESC LIMIT $` + strconv.Itoa(argCount+1) + ` OFFSET $` + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error querying members: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get members", nil)
		return
	}
	defer rows.Close()

	var members []models.ClubMember
	for rows.Next() {
		var member models.ClubMember
		var user models.User

		err := rows.Scan(
			&member.ID, &member.ClubID, &member.UserID, &member.Role,
			&member.JoinedDate, &member.BooksRead, &member.IsActive,
			&user.ID, &user.Name, &user.Email, &user.Phone, &user.Avatar,
		)
		if err != nil {
			log.Printf("Error scanning member: %v", err)
			continue
		}

		member.User = &user
		members = append(members, member)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM club_members WHERE club_id = $1`
	countArgs := []interface{}{clubID}

	if role != "" {
		countQuery += ` AND role = $2`
		countArgs = append(countArgs, role)
	}

	var total int
	h.db.QueryRowContext(r.Context(), countQuery, countArgs...).Scan(&total)

	totalPages := (total + limit - 1) / limit

	// Transform members to frontend format
	var frontendMembers []*models.FrontendClubMember
	for _, member := range members {
		frontendMembers = append(frontendMembers, member.ToFrontendFormat())
	}

	response := map[string]interface{}{
		"members": frontendMembers,
		"pagination": models.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	h.writeSuccessResponse(w, response, "Members retrieved successfully")
}

func (h *ClubHandler) AddMember(w http.ResponseWriter, r *http.Request) {
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

	// Check if user has permission to add members (owner or moderator)
	if !h.canManageMembers(r.Context(), clubID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	var req models.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Check if user already is a member
	if h.isClubMember(r.Context(), clubID, req.UserID) {
		h.writeErrorResponse(w, http.StatusConflict, "CONFLICT", "User is already a member", nil)
		return
	}

	// Add member
	memberID := uuid.New()
	query := `
		INSERT INTO club_members (id, club_id, user_id, role) 
		VALUES ($1, $2, $3, $4)`

	_, err = h.db.ExecContext(r.Context(), query, memberID, clubID, req.UserID, req.Role)
	if err != nil {
		log.Printf("Error adding member: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add member", nil)
		return
	}

	member := &models.ClubMember{
		ID:         memberID,
		ClubID:     clubID,
		UserID:     req.UserID,
		Role:       req.Role,
		JoinedDate: time.Now(),
		BooksRead:  0,
		IsActive:   true,
	}

	response := map[string]interface{}{
		"member": member,
	}

	w.WriteHeader(http.StatusCreated)
	h.writeSuccessResponse(w, response, "Member added successfully")
}

func (h *ClubHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "clubId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid club ID", nil)
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "memberId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid member ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check permissions
	if !h.canManageMembers(r.Context(), clubID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	var req models.UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Build update query
	setParts := []string{}
	args := []interface{}{}
	argCount := 0

	if req.Role != nil {
		argCount++
		setParts = append(setParts, "role = $"+strconv.Itoa(argCount))
		args = append(args, *req.Role)
	}

	if req.IsActive != nil {
		argCount++
		setParts = append(setParts, "is_active = $"+strconv.Itoa(argCount))
		args = append(args, *req.IsActive)
	}

	if len(setParts) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "No fields to update", nil)
		return
	}

	argCount++
	args = append(args, memberID)

	query := `UPDATE club_members SET ` + join(setParts, ", ") + ` WHERE id = $` + strconv.Itoa(argCount)

	_, err = h.db.ExecContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("Error updating member: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update member", nil)
		return
	}

	response := map[string]interface{}{
		"member": map[string]interface{}{
			"id":        memberID,
			"updatedAt": time.Now(),
		},
	}

	if req.Role != nil {
		response["member"].(map[string]interface{})["role"] = *req.Role
	}
	if req.IsActive != nil {
		response["member"].(map[string]interface{})["isActive"] = *req.IsActive
	}

	h.writeSuccessResponse(w, response, "Member updated successfully")
}

func (h *ClubHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "clubId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid club ID", nil)
		return
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "memberId"))
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid member ID", nil)
		return
	}

	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	// Check permissions
	if !h.canManageMembers(r.Context(), clubID, userID) {
		h.writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", nil)
		return
	}

	query := `DELETE FROM club_members WHERE id = $1 AND club_id = $2`
	result, err := h.db.ExecContext(r.Context(), query, memberID, clubID)
	if err != nil {
		log.Printf("Error removing member: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove member", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Member not found", nil)
		return
	}

	response := map[string]string{
		"message": "Member removed successfully",
	}

	h.writeSuccessResponse(w, response, "Member removed successfully")
}

// Helper methods
func (h *ClubHandler) isClubMember(ctx context.Context, clubID, userID uuid.UUID) bool {
	query := `SELECT 1 FROM club_members WHERE club_id = $1 AND user_id = $2 AND is_active = true`
	var exists int
	err := h.db.QueryRowContext(ctx, query, clubID, userID).Scan(&exists)
	return err == nil
}

func (h *ClubHandler) canManageMembers(ctx context.Context, clubID, userID uuid.UUID) bool {
	query := `SELECT role FROM club_members WHERE club_id = $1 AND user_id = $2 AND is_active = true`
	var role string
	err := h.db.QueryRowContext(ctx, query, clubID, userID).Scan(&role)
	if err != nil {
		return false
	}
	return role == "owner" || role == "moderator"
}

func (h *ClubHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")

	response := models.NewAPIResponse(true, data, message)
	json.NewEncoder(w).Encode(response)
}

func (h *ClubHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
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

// String join helper
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	var result string
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
