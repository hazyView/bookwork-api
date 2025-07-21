package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"bookwork-api/internal/auth"
	"bookwork-api/internal/database"
	"bookwork-api/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db   *database.DB
	auth *auth.Service
}

func NewAuthHandler(db *database.DB, authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		db:   db,
		auth: authService,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and password are required", nil)
		return
	}

	// Get user from database
	user, err := h.getUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid credentials", nil)
			return
		}
		log.Printf("Error getting user: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Verify password
	if !h.auth.VerifyPassword(user.PasswordHash, req.Password) {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid credentials", nil)
		return
	}

	// Check if user is active
	if !user.IsActive {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Account is deactivated", nil)
		return
	}

	// Generate tokens
	tokens, err := h.auth.GenerateTokens(user)
	if err != nil {
		log.Printf("Error generating tokens: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate tokens", nil)
		return
	}

	// Store refresh token in database
	if err := h.storeRefreshToken(r.Context(), user.ID, tokens.RefreshToken); err != nil {
		log.Printf("Error storing refresh token: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store refresh token", nil)
		return
	}

	// Update last login
	if err := h.updateLastLogin(r.Context(), user.ID); err != nil {
		log.Printf("Error updating last login: %v", err)
		// Don't fail the request for this
	}

	// Calculate expiration time (30 minutes from now)
	expiresAt := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)

	response := &models.FrontendLoginResponse{
		Token:     tokens.AccessToken,
		User:      user.PublicUser(),
		ExpiresAt: expiresAt,
	}

	h.writeSuccessResponse(w, response, "Login successful")
}

func (h *AuthHandler) Validate(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found in context", nil)
		return
	}

	user, err := h.getUserByID(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found", nil)
			return
		}
		log.Printf("Error getting user: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	response := map[string]interface{}{
		"valid": true,
		"user":  user.PublicUser(),
	}

	h.writeSuccessResponse(w, response, "Token is valid")
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	if req.RefreshToken == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Refresh token is required", nil)
		return
	}

	// Validate refresh token
	claims, err := h.auth.ValidateToken(req.RefreshToken)
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid refresh token", nil)
		return
	}

	if claims.Type != "refresh" {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token type", nil)
		return
	}

	// Check if refresh token exists and is not revoked
	exists, err := h.isRefreshTokenValid(r.Context(), claims.UserID, req.RefreshToken)
	if err != nil {
		log.Printf("Error checking refresh token: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if !exists {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Refresh token not found or revoked", nil)
		return
	}

	// Get user
	user, err := h.getUserByID(r.Context(), claims.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not found", nil)
			return
		}
		log.Printf("Error getting user: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Generate new access token
	newAccessToken, err := h.auth.GenerateTokens(user)
	if err != nil {
		log.Printf("Error generating new access token: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate new token", nil)
		return
	}

	// Calculate expiration time (30 minutes from now)
	expiresAt := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)

	response := &models.FrontendRefreshResponse{
		Token:     newAccessToken.AccessToken,
		ExpiresAt: expiresAt,
	}

	h.writeSuccessResponse(w, response, "Token refreshed successfully")
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req models.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON format", nil)
		return
	}

	if req.RefreshToken == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Refresh token is required", nil)
		return
	}

	// Validate and revoke refresh token
	claims, err := h.auth.ValidateToken(req.RefreshToken)
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid refresh token", nil)
		return
	}

	// Revoke the refresh token
	if err := h.revokeRefreshToken(r.Context(), claims.UserID, req.RefreshToken); err != nil {
		log.Printf("Error revoking refresh token: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke token", nil)
		return
	}

	response := map[string]string{
		"message": "Successfully logged out",
	}

	h.writeSuccessResponse(w, response, "Logout successful")
}

// Database helper methods
func (h *AuthHandler) getUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, phone, avatar, role, is_active, 
		       last_login_at, created_at, updated_at 
		FROM users 
		WHERE email = $1 AND is_active = true`

	var user models.User
	err := h.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash,
		&user.Phone, &user.Avatar, &user.Role, &user.IsActive,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)

	return &user, err
}

func (h *AuthHandler) getUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, phone, avatar, role, is_active, 
		       last_login_at, created_at, updated_at 
		FROM users 
		WHERE id = $1 AND is_active = true`

	var user models.User
	err := h.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash,
		&user.Phone, &user.Avatar, &user.Role, &user.IsActive,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)

	return &user, err
}

func (h *AuthHandler) storeRefreshToken(ctx context.Context, userID uuid.UUID, token string) error {
	// Hash the token before storing
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at) 
		VALUES ($1, $2, $3)`

	_, err = h.db.ExecContext(ctx, query, userID, string(hashedToken), time.Now().Add(7*24*time.Hour))
	return err
}

func (h *AuthHandler) isRefreshTokenValid(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	query := `
		SELECT token_hash 
		FROM refresh_tokens 
		WHERE user_id = $1 AND expires_at > NOW() AND is_revoked = false`

	rows, err := h.db.QueryContext(ctx, query, userID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var hashedToken string
		if err := rows.Scan(&hashedToken); err != nil {
			continue
		}

		if bcrypt.CompareHashAndPassword([]byte(hashedToken), []byte(token)) == nil {
			return true, nil
		}
	}

	return false, nil
}

func (h *AuthHandler) revokeRefreshToken(ctx context.Context, userID uuid.UUID, token string) error {
	// First, find the token
	query := `
		UPDATE refresh_tokens 
		SET is_revoked = true 
		WHERE user_id = $1 AND expires_at > NOW() AND is_revoked = false`

	_, err := h.db.ExecContext(ctx, query, userID)
	return err
}

func (h *AuthHandler) updateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	_, err := h.db.ExecContext(ctx, query, userID)
	return err
}

// Response helper methods
func (h *AuthHandler) writeSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := models.NewAPIResponse(true, data, message)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
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
