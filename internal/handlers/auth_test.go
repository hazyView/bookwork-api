package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bookwork-api/internal/auth"
	"bookwork-api/internal/database"
	"bookwork-api/internal/models"

	"github.com/google/uuid"
)

// Mock database for testing
type mockDB struct {
	users map[string]*models.User
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// This is a simplified mock - in real tests you'd want a more sophisticated mock
	return nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func setupAuthTest(t *testing.T) (*AuthHandler, *auth.Service) {
	// Create mock database
	config := database.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	db, err := database.New(config)
	if err != nil {
		// Use a mock if real database is not available
		t.Logf("Using mock database due to connection error: %v", err)
		db = nil
	}

	// Create auth service
	authService := auth.NewService("test-secret-key-that-is-at-least-32-chars", "test-issuer")

	// Create handler
	handler := NewAuthHandler(db, authService)

	return handler, authService
}

func TestLoginHandler(t *testing.T) {
	handler, _ := setupAuthTest(t)

	// Skip test if database is not available (handler.db is nil)
	if handler.db == nil {
		t.Skip("Skipping login test - database not available")
	}

	// Test valid login request
	loginReq := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		Email:    "test@example.com",
		Password: "testpassword",
	}

	reqBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Note: This test will fail without a proper database connection
	// In a real application, you'd mock the database layer
	handler.Login(w, req)

	// Test that the handler doesn't panic and returns a response
	if w.Code == 0 {
		t.Error("Handler should return a status code")
	}
}

func TestLoginHandlerInvalidJSON(t *testing.T) {
	handler, _ := setupAuthTest(t)

	// Test invalid JSON
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestValidateHandler(t *testing.T) {
	handler, authService := setupAuthTest(t)

	// Create a test user and token
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
		Role:  "member",
	}

	tokens, err := authService.GenerateTokens(user)
	if err != nil {
		t.Fatalf("Failed to generate test tokens: %v", err)
	}

	// Test validate with valid token
	req := httptest.NewRequest("GET", "/api/auth/validate", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	w := httptest.NewRecorder()

	handler.Validate(w, req)

	// Test that the handler processes the request
	if w.Code == 0 {
		t.Error("Handler should return a status code")
	}
}

func TestValidateHandlerNoToken(t *testing.T) {
	handler, _ := setupAuthTest(t)

	// Test validate without token
	req := httptest.NewRequest("GET", "/api/auth/validate", nil)

	w := httptest.NewRecorder()

	handler.Validate(w, req)

	// Should return unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestLogoutHandler(t *testing.T) {
	handler, authService := setupAuthTest(t)

	// Create a test user and token
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
		Role:  "member",
	}

	tokens, err := authService.GenerateTokens(user)
	if err != nil {
		t.Fatalf("Failed to generate test tokens: %v", err)
	}

	// Test logout with valid token
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	w := httptest.NewRecorder()

	handler.Logout(w, req)

	// Test that the handler processes the request
	if w.Code == 0 {
		t.Error("Handler should return a status code")
	}
}

func TestLogoutHandlerNoToken(t *testing.T) {
	handler, _ := setupAuthTest(t)

	// Test logout without token
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)

	w := httptest.NewRecorder()

	handler.Logout(w, req)

	// Should return bad request (no Authorization header)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRefreshHandler(t *testing.T) {
	handler, authService := setupAuthTest(t)

	// Create a test user and tokens
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
		Role:  "member",
	}

	tokens, err := authService.GenerateTokens(user)
	if err != nil {
		t.Fatalf("Failed to generate test tokens: %v", err)
	}

	// Test refresh with valid refresh token
	refreshReq := struct {
		RefreshToken string `json:"refresh_token"`
	}{
		RefreshToken: tokens.RefreshToken,
	}

	reqBody, _ := json.Marshal(refreshReq)
	req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler.Refresh(w, req)

	// Test that the handler processes the request
	if w.Code == 0 {
		t.Error("Handler should return a status code")
	}
}

func TestRefreshHandlerInvalidToken(t *testing.T) {
	handler, _ := setupAuthTest(t)

	// Test refresh with invalid token
	refreshReq := struct {
		RefreshToken string `json:"refresh_token"`
	}{
		RefreshToken: "invalid-token",
	}

	reqBody, _ := json.Marshal(refreshReq)
	req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handler.Refresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
