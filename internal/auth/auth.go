package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bookwork-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	secretKey []byte
	issuer    string
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Type   string    `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func NewService(secretKey, issuer string) *Service {
	return &Service{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

func (s *Service) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

func (s *Service) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func (s *Service) GenerateTokens(user *models.User) (*models.TokenResponse, error) {
	// Generate access token (30 minutes)
	accessToken, err := s.generateToken(user, "access", 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (7 days)
	refreshToken, err := s.generateToken(user, "refresh", 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    1800, // 30 minutes in seconds
	}, nil
}

func (s *Service) generateToken(user *models.User, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *Service) GenerateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Middleware for JWT authentication
func (s *Service) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			s.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format", nil)
			return
		}

		claims, err := s.ValidateToken(parts[1])
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			s.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token", nil)
			return
		}

		if claims.Type != "access" {
			s.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token type", nil)
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_role", claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Create frontend-compatible error response
	jsonResponse := fmt.Sprintf(`{"error":"%s","message":"%s","statusCode":%d,"timestamp":"%s"}`,
		code, message, statusCode, time.Now().UTC().Format(time.RFC3339))

	w.Write([]byte(jsonResponse))
}

// Helper functions to extract user info from context
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

func GetUserRoleFromContext(ctx context.Context) (string, error) {
	role, ok := ctx.Value("user_role").(string)
	if !ok {
		return "", fmt.Errorf("user role not found in context")
	}
	return role, nil
}
