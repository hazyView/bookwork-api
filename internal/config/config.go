package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	CORS     CORSConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port           string
	Host           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	AllowedOrigins []string
}

type SecurityConfig struct {
	EnableHSTS      bool
	HSTSMaxAge      int
	EnableHTTPSOnly bool
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	MaxAge           int
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PgBouncerAddr   string
}

type JWTConfig struct {
	SecretKey string
	Issuer    string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		Server: ServerConfig{
			Port:           getEnv("PORT", "8000"),
			Host:           getEnv("HOST", "localhost"),
			ReadTimeout:    getEnvAsDuration("READ_TIMEOUT", "30s"),
			WriteTimeout:   getEnvAsDuration("WRITE_TIMEOUT", "30s"),
			AllowedOrigins: getEnvAsStringArray("ALLOWED_ORIGINS", []string{"http://localhost:5173", "http://localhost:3000"}),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			Database:        getEnv("DB_NAME", "bookwork"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", "2m"),
			PgBouncerAddr:   getEnv("PGBOUNCER_ADDR", ""),
		},
		JWT: JWTConfig{
			SecretKey: getJWTSecret(),
			Issuer:    getEnv("JWT_ISSUER", "bookwork-api"),
		},
		CORS: CORSConfig{
			AllowedOrigins:   getEnvAsStringArray("ALLOWED_ORIGINS", []string{"http://localhost:5173", "http://localhost:3000"}),
			AllowCredentials: getEnvAsBool("ALLOW_CREDENTIALS", true),
			MaxAge:           getEnvAsInt("CORS_MAX_AGE", 300),
		},
		Security: SecurityConfig{
			EnableHSTS:      getEnvAsBool("ENABLE_HSTS", true),
			HSTSMaxAge:      getEnvAsInt("HSTS_MAX_AGE", 31536000),
			EnableHTTPSOnly: getEnvAsBool("ENABLE_HTTPS_ONLY", false),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		log.Printf("Warning: Invalid boolean value for %s: %s, using default: %t", key, value, defaultValue)
	}
	return defaultValue
}

func getEnvAsStringArray(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	if defaultDuration, err := time.ParseDuration(defaultValue); err == nil {
		log.Printf("Warning: Invalid duration value for %s: %s, using default: %s", key, value, defaultValue)
		return defaultDuration
	}
	log.Printf("Error: Invalid default duration value: %s, using 5m", defaultValue)
	return 5 * time.Minute
}

// getJWTSecret returns the JWT secret key with production validation
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")

	// If no JWT secret is provided
	if secret == "" {
		// In production environment, JWT_SECRET is required
		if env := os.Getenv("ENV"); env == "production" || env == "prod" {
			log.Fatal("JWT_SECRET environment variable is required in production")
		}

		// For development, warn about using default and provide a secure default
		log.Println("Warning: JWT_SECRET not set, using development default. SET JWT_SECRET for production!")
		return "dev-jwt-secret-change-this-in-production-environments-use-at-least-32-characters"
	}

	// Validate secret length (minimum 32 characters for security)
	if len(secret) < 32 {
		log.Printf("Warning: JWT_SECRET should be at least 32 characters (current: %d)", len(secret))
		if env := os.Getenv("ENV"); env == "production" || env == "prod" {
			log.Fatal("JWT_SECRET must be at least 32 characters in production")
		}
	}

	// Check if using the old default value
	if secret == "your-super-secret-jwt-key-change-this-in-production" {
		log.Println("Warning: You are using the default JWT_SECRET. Please change it for security!")
		if env := os.Getenv("ENV"); env == "production" || env == "prod" {
			log.Fatal("Cannot use default JWT_SECRET in production")
		}
	}

	return secret
}
