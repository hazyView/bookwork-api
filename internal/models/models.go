package models

import (
	"database/sql/driver"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// StringArray is a custom type for PostgreSQL text arrays
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	return pq.Array([]string(a)).Value()
}

func (a *StringArray) Scan(value interface{}) error {
	arr := pq.StringArray{}
	if err := arr.Scan(value); err != nil {
		return err
	}
	*a = StringArray([]string(arr))
	return nil
}

// UUIDArray is a custom type for PostgreSQL UUID arrays
type UUIDArray []uuid.UUID

func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}

	uuidStrings := make([]string, len(a))
	for i, u := range a {
		uuidStrings[i] = u.String()
	}
	return pq.Array(uuidStrings).Value()
}

func (a *UUIDArray) Scan(value interface{}) error {
	arr := pq.StringArray{}
	if err := arr.Scan(value); err != nil {
		return err
	}

	*a = make(UUIDArray, len(arr))
	for i, s := range arr {
		u, err := uuid.Parse(s)
		if err != nil {
			return err
		}
		(*a)[i] = u
	}
	return nil
}

// User represents a user in the system
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Phone        *string    `json:"phone,omitempty" db:"phone"`
	Avatar       *string    `json:"avatar,omitempty" db:"avatar"`
	Role         string     `json:"role" db:"role"`
	IsActive     bool       `json:"isActive" db:"is_active"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty" db:"last_login_at"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" db:"updated_at"`
	JoinedDate   *time.Time `json:"joinedDate,omitempty"` // For API compatibility
}

// PublicUser returns user info without sensitive data
func (u *User) PublicUser() *User {
	return &User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Phone:     u.Phone,
		Avatar:    u.Avatar,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}

// Club represents a book club
type Club struct {
	ID               uuid.UUID   `json:"id" db:"id"`
	Name             string      `json:"name" db:"name"`
	Description      string      `json:"description" db:"description"`
	OwnerID          uuid.UUID   `json:"ownerId" db:"owner_id"`
	MemberCount      int         `json:"memberCount"`
	IsPublic         bool        `json:"isPublic" db:"is_public"`
	MaxMembers       *int        `json:"maxMembers,omitempty" db:"max_members"`
	MeetingFrequency *string     `json:"meetingFrequency,omitempty" db:"meeting_frequency"`
	CurrentBook      *string     `json:"currentBook,omitempty" db:"current_book"`
	Tags             StringArray `json:"tags" db:"tags"`
	Location         *string     `json:"location,omitempty" db:"location"`
	CreatedAt        time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time   `json:"updatedAt" db:"updated_at"`
}

// ClubMember represents a membership in a club
type ClubMember struct {
	ID         uuid.UUID `json:"id" db:"id"`
	ClubID     uuid.UUID `json:"clubId" db:"club_id"`
	UserID     uuid.UUID `json:"userId" db:"user_id"`
	Role       string    `json:"role" db:"role"`
	JoinedDate time.Time `json:"joinedDate" db:"joined_date"`
	BooksRead  int       `json:"booksRead" db:"books_read"`
	IsActive   bool      `json:"isActive" db:"is_active"`
	User       *User     `json:"user,omitempty"`
}

// Event represents a club event
type Event struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ClubID       uuid.UUID `json:"clubId" db:"club_id"`
	Title        string    `json:"title" db:"title"`
	Description  *string   `json:"description,omitempty" db:"description"`
	Date         string    `json:"date" db:"event_date"`
	Time         string    `json:"time" db:"event_time"`
	Location     string    `json:"location" db:"location"`
	Book         *string   `json:"book,omitempty" db:"book"`
	Type         string    `json:"type" db:"type"`
	MaxAttendees *int      `json:"maxAttendees,omitempty" db:"max_attendees"`
	IsPublic     bool      `json:"isPublic" db:"is_public"`
	CreatedBy    uuid.UUID `json:"createdBy" db:"created_by"`
	Attendees    UUIDArray `json:"attendees" db:"attendees"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

// EventItem represents a coordination item for an event
type EventItem struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	EventID    uuid.UUID  `json:"eventId" db:"event_id"`
	Name       string     `json:"name" db:"name"`
	Category   string     `json:"category" db:"category"`
	AssignedTo *uuid.UUID `json:"assignedTo,omitempty" db:"assigned_to"`
	Status     string     `json:"status" db:"status"`
	Notes      *string    `json:"notes,omitempty" db:"notes"`
	CreatedBy  uuid.UUID  `json:"createdBy" db:"created_by"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`
}

// Availability represents a user's availability for an event
type Availability struct {
	ID        uuid.UUID `json:"id" db:"id"`
	EventID   uuid.UUID `json:"eventId" db:"event_id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	Status    string    `json:"status" db:"status"`
	Notes     *string   `json:"notes,omitempty" db:"notes"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"`
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	IsRevoked bool      `json:"isRevoked" db:"is_revoked"`
}

// API Response structures
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

type LoginResponse struct {
	User   *User          `json:"user"`
	Tokens *TokenResponse `json:"tokens"`
}

type ValidateResponse struct {
	User  *User `json:"user"`
	Valid bool  `json:"valid"`
}

type CreateEventRequest struct {
	Title        string  `json:"title" validate:"required,min=1,max=100"`
	Description  *string `json:"description,omitempty"`
	Date         string  `json:"date" validate:"required"`
	Time         string  `json:"time" validate:"required"`
	Location     string  `json:"location" validate:"required,min=1,max=200"`
	Book         *string `json:"book,omitempty"`
	Type         string  `json:"type" validate:"required"`
	MaxAttendees *int    `json:"maxAttendees,omitempty"`
	IsPublic     bool    `json:"isPublic"`
}

type CreateEventItemRequest struct {
	Item EventItemRequest `json:"item"`
}

type EventItemRequest struct {
	Name       string     `json:"name" validate:"required"`
	Category   string     `json:"category" validate:"required"`
	AssignedTo *uuid.UUID `json:"assignedTo,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
}

type UpdateEventItemRequest struct {
	Status string  `json:"status,omitempty"`
	Notes  *string `json:"notes,omitempty"`
}

type AvailabilityRequest struct {
	UserID uuid.UUID `json:"userId" validate:"required"`
	Status string    `json:"status" validate:"required"`
	Notes  *string   `json:"notes,omitempty"`
}

type AvailabilitySummary struct {
	Available   int `json:"available"`
	Maybe       int `json:"maybe"`
	Unavailable int `json:"unavailable"`
	Total       int `json:"total"`
}

type AvailabilityResponse struct {
	Availability map[string]*Availability `json:"availability"`
	Summary      *AvailabilitySummary     `json:"summary"`
}

type AddMemberRequest struct {
	UserID uuid.UUID `json:"userId" validate:"required"`
	Role   string    `json:"role" validate:"required"`
}

type UpdateMemberRequest struct {
	Role     *string `json:"role,omitempty"`
	IsActive *bool   `json:"isActive,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

func NewAPIResponse(success bool, data interface{}, message string) *APIResponse {
	return &APIResponse{
		Success:   success,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

func NewErrorResponse(code, message string, details map[string]interface{}) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC(),
	}
}

// Frontend-Compatible Response Models

// FrontendLoginResponse matches the frontend authentication response format
type FrontendLoginResponse struct {
	Token     string `json:"token"`
	User      *User  `json:"user"`
	ExpiresAt string `json:"expiresAt"`
}

// FrontendRefreshResponse matches the frontend token refresh response format
type FrontendRefreshResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

// FrontendErrorResponse matches the frontend error handling format
type FrontendErrorResponse struct {
	Error      string      `json:"error"`
	Message    string      `json:"message"`
	StatusCode int         `json:"statusCode"`
	Details    interface{} `json:"details,omitempty"`
	Timestamp  string      `json:"timestamp,omitempty"`
}

// FrontendClubMember matches the frontend club member format with flattened user data
type FrontendClubMember struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Avatar      *string  `json:"avatar,omitempty"`
	Role        string   `json:"role"`
	JoinDate    string   `json:"joinDate"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
}

// FrontendEvent matches the frontend event format with combined datetime
type FrontendEvent struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Date        string  `json:"date"` // ISO 8601 combined datetime
	Location    *string `json:"location,omitempty"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	OrganizerID string  `json:"organizerId"`
}

// FrontendEventItem matches the frontend event item format
type FrontendEventItem struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`                 // Maps from "name"
	Description *string `json:"description,omitempty"` // Maps from "notes"
	Type        string  `json:"type"`                  // Maps from "category"
	Status      string  `json:"status"`
	AssigneeID  *string `json:"assigneeId,omitempty"`
	DueDate     *string `json:"dueDate,omitempty"`
}

// FrontendAvailability matches the frontend availability format
type FrontendAvailability struct {
	UserID    string  `json:"userId"`
	Status    string  `json:"status"`
	Note      *string `json:"note,omitempty"`
	UpdatedAt string  `json:"updatedAt"`
}

// Utility functions for permissions based on role
func getPermissionsForRole(role string) []string {
	switch role {
	case "admin":
		return []string{"read", "write", "delete"}
	case "moderator":
		return []string{"read", "write"}
	case "member":
		return []string{"read", "write"}
	case "guest":
		return []string{"read"}
	default:
		return []string{"read"}
	}
}

// Conversion methods to transform models to frontend format

// ToFrontendFormat converts a ClubMember to frontend-compatible format
func (cm *ClubMember) ToFrontendFormat() *FrontendClubMember {
	status := "active"
	if !cm.IsActive {
		status = "inactive"
	}

	permissions := getPermissionsForRole(cm.Role)

	return &FrontendClubMember{
		ID:          cm.User.ID.String(),
		Name:        cm.User.Name,
		Email:       cm.User.Email,
		Avatar:      cm.User.Avatar,
		Role:        cm.Role,
		JoinDate:    cm.JoinedDate.UTC().Format(time.RFC3339),
		Status:      status,
		Permissions: permissions,
	}
}

// ToFrontendFormat converts an Event to frontend-compatible format
func (e *Event) ToFrontendFormat() *FrontendEvent {
	// Combine date and time into ISO 8601 format
	datetime, err := time.Parse("2006-01-02 15:04:05", e.Date+" "+e.Time)
	if err != nil {
		// Fallback to just the date if time parsing fails
		datetime, _ = time.Parse("2006-01-02", e.Date)
	}

	// Determine status (adding basic logic for event status)
	status := "scheduled"
	now := time.Now()
	if datetime.Before(now) {
		status = "completed"
	}

	return &FrontendEvent{
		ID:          e.ID.String(),
		Title:       e.Title,
		Description: e.Description,
		Date:        datetime.UTC().Format(time.RFC3339),
		Location:    &e.Location,
		Type:        e.Type,
		Status:      status,
		OrganizerID: e.CreatedBy.String(),
	}
}

// ToFrontendFormat converts an EventItem to frontend-compatible format
func (ei *EventItem) ToFrontendFormat() *FrontendEventItem {
	var assigneeID *string
	if ei.AssignedTo != nil {
		id := ei.AssignedTo.String()
		assigneeID = &id
	}

	var dueDate *string
	if !ei.CreatedAt.IsZero() {
		// For now, use creation date as due date; in a real implementation,
		// you might have a separate due_date field
		date := ei.UpdatedAt.UTC().Format(time.RFC3339)
		dueDate = &date
	}

	return &FrontendEventItem{
		ID:          ei.ID.String(),
		Title:       ei.Name,     // Map "name" to "title"
		Description: ei.Notes,    // Map "notes" to "description"
		Type:        ei.Category, // Map "category" to "type"
		Status:      ei.Status,
		AssigneeID:  assigneeID,
		DueDate:     dueDate,
	}
}

// ToFrontendFormat converts Availability to frontend-compatible format
func (a *Availability) ToFrontendFormat() *FrontendAvailability {
	return &FrontendAvailability{
		UserID:    a.UserID.String(),
		Status:    a.Status,
		Note:      a.Notes,
		UpdatedAt: a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
