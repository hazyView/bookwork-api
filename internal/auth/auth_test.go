package auth

import (
	"bookwork-api/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTTokenGeneration(t *testing.T) {
	service := NewService("test-secret", "test-issuer")

	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Role:  "member",
		Name:  "Test User",
	}

	tokens, err := service.GenerateTokens(user)
	if err != nil {
		t.Fatalf("Failed to generate tokens: %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("Generated access token is empty")
	}

	if tokens.RefreshToken == "" {
		t.Error("Generated refresh token is empty")
	}

	// Validate access token can be parsed
	claims, err := service.ValidateToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate generated token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}

	if claims.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, claims.Email)
	}

	if claims.Role != user.Role {
		t.Errorf("Expected Role %s, got %s", user.Role, claims.Role)
	}

	if claims.Type != "access" {
		t.Errorf("Expected Type 'access', got %s", claims.Type)
	}
}

func TestJWTTokenValidation(t *testing.T) {
	service := NewService("test-secret", "test-issuer")

	// Test invalid token
	_, err := service.ValidateToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}

	// Test empty token
	_, err = service.ValidateToken("")
	if err == nil {
		t.Error("Expected error for empty token, got nil")
	}
}

func TestJWTTokenExpiration(t *testing.T) {
	service := NewService("test-secret", "test-issuer")

	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Role:  "member",
		Name:  "Test User",
	}

	tokens, err := service.GenerateTokens(user)
	if err != nil {
		t.Fatalf("Failed to generate tokens: %v", err)
	}

	// Validate token is currently valid
	claims, err := service.ValidateToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("Token should be valid: %v", err)
	}

	// Check expiration time is in the future
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("Token should have future expiration time")
	}
}

func TestHashPassword(t *testing.T) {
	service := NewService("test-secret", "test-issuer")
	password := "testpassword123"

	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal original password")
	}
}

func TestVerifyPassword(t *testing.T) {
	service := NewService("test-secret", "test-issuer")
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	if !service.VerifyPassword(hash, password) {
		t.Error("Password verification should succeed for correct password")
	}

	// Test wrong password
	if service.VerifyPassword(hash, wrongPassword) {
		t.Error("Password verification should fail for wrong password")
	}

	// Test empty password
	if service.VerifyPassword(hash, "") {
		t.Error("Password verification should fail for empty password")
	}
}
