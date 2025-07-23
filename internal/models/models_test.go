package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStringArrayValue(t *testing.T) {
	// Test empty array
	empty := StringArray{}
	val, err := empty.Value()
	if err != nil {
		t.Errorf("Empty array Value() failed: %v", err)
	}
	if val == nil {
		t.Error("Empty array should not return nil value")
	}

	// Test array with values
	arr := StringArray{"hello", "world", "test"}
	val, err = arr.Value()
	if err != nil {
		t.Errorf("Array Value() failed: %v", err)
	}
	if val == nil {
		t.Error("Array should not return nil value")
	}
}

func TestStringArrayScan(t *testing.T) {
	// Test scanning nil
	var arr StringArray
	err := arr.Scan(nil)
	if err != nil {
		t.Errorf("Scanning nil failed: %v", err)
	}

	// Test scanning empty array
	err = arr.Scan("{}")
	if err != nil {
		t.Errorf("Scanning empty array failed: %v", err)
	}

	// Test scanning array with values
	err = arr.Scan(`{"hello","world","test"}`)
	if err != nil {
		t.Errorf("Scanning array failed: %v", err)
	}

	expected := []string{"hello", "world", "test"}
	if len(arr) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(arr))
	}

	for i, exp := range expected {
		if i >= len(arr) || arr[i] != exp {
			t.Errorf("Expected arr[%d] = %s, got %s", i, exp, arr[i])
		}
	}
}

func TestUUIDArrayValue(t *testing.T) {
	// Test empty array
	empty := UUIDArray{}
	val, err := empty.Value()
	if err != nil {
		t.Errorf("Empty UUID array Value() failed: %v", err)
	}
	expectedEmpty := "{}"
	if val != expectedEmpty {
		t.Errorf("Expected empty array value '%s', got '%v'", expectedEmpty, val)
	}

	// Test array with UUIDs
	id1 := uuid.New()
	id2 := uuid.New()
	arr := UUIDArray{id1, id2}
	val, err = arr.Value()
	if err != nil {
		t.Errorf("UUID array Value() failed: %v", err)
	}
	if val == nil {
		t.Error("UUID array should not return nil value")
	}
}

func TestUUIDArrayScan(t *testing.T) {
	// Test scanning nil
	var arr UUIDArray
	err := arr.Scan(nil)
	if err != nil {
		t.Errorf("Scanning nil failed: %v", err)
	}

	// Test scanning empty array
	err = arr.Scan("{}")
	if err != nil {
		t.Errorf("Scanning empty array failed: %v", err)
	}

	// Test scanning array with UUIDs
	id1 := uuid.New()
	id2 := uuid.New()
	uuidStr := `{"` + id1.String() + `","` + id2.String() + `"}`

	err = arr.Scan(uuidStr)
	if err != nil {
		t.Errorf("Scanning UUID array failed: %v", err)
	}

	if len(arr) != 2 {
		t.Errorf("Expected length 2, got %d", len(arr))
	}

	if arr[0] != id1 {
		t.Errorf("Expected arr[0] = %s, got %s", id1, arr[0])
	}

	if arr[1] != id2 {
		t.Errorf("Expected arr[1] = %s, got %s", id2, arr[1])
	}
}

func TestUserModel(t *testing.T) {
	// Test user creation
	user := &User{
		ID:        uuid.New(),
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      "member",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if user.ID == uuid.Nil {
		t.Error("User ID should not be nil")
	}

	if user.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got %s", user.Name)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", user.Email)
	}

	if user.Role != "member" {
		t.Errorf("Expected role 'member', got %s", user.Role)
	}

	if !user.IsActive {
		t.Error("User should be active")
	}
}

func TestTokenResponse(t *testing.T) {
	// Test token response creation
	response := &TokenResponse{
		AccessToken:  "access_token_123",
		RefreshToken: "refresh_token_456",
		ExpiresIn:    3600,
	}

	if response.AccessToken != "access_token_123" {
		t.Errorf("Expected access token 'access_token_123', got %s", response.AccessToken)
	}

	if response.RefreshToken != "refresh_token_456" {
		t.Errorf("Expected refresh token 'refresh_token_456', got %s", response.RefreshToken)
	}

	if response.ExpiresIn != 3600 {
		t.Errorf("Expected expires in 3600, got %d", response.ExpiresIn)
	}
}

func TestAPIResponse(t *testing.T) {
	// Test success response
	successData := map[string]string{"message": "success"}
	response := &APIResponse{
		Success: true,
		Data:    successData,
		Message: "Operation completed successfully",
	}

	if !response.Success {
		t.Error("Response should be successful")
	}

	if response.Message != "Operation completed successfully" {
		t.Errorf("Expected message 'Operation completed successfully', got %s", response.Message)
	}

	// Test error response
	errorResponse := &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid input data",
		},
	}

	if errorResponse.Success {
		t.Error("Error response should not be successful")
	}

	if errorResponse.Error == nil {
		t.Error("Error response should have error details")
	}

	if errorResponse.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected error code 'VALIDATION_ERROR', got %s", errorResponse.Error.Code)
	}
}
