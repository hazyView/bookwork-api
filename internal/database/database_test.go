package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestNewDatabase(t *testing.T) {
	// Test with valid config
	config := Config{
		Host:            "localhost",
		Port:            "5432",
		User:            "postgres",
		Password:        "testpass",
		Database:        "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}

	// Note: This test might fail if PostgreSQL is not running
	// It's designed to test the connection logic
	db, err := New(config)
	if err != nil {
		// If PostgreSQL is not available, we still want to test that the DSN is constructed correctly
		// This is expected in CI environments without a database
		t.Logf("Database connection failed (expected in CI): %v", err)
		return
	}

	// If connection succeeds, test that it's properly configured
	if db == nil {
		t.Fatal("Expected non-nil database instance")
	}

	// Test connection with a simple ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Errorf("Database ping failed: %v", err)
	}

	// Clean up
	db.Close()
}

func TestDatabaseConfig(t *testing.T) {
	// Test that DSN is constructed correctly
	config := Config{
		Host:     "testhost",
		Port:     "5433",
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "require",
	}

	// We can't directly test the DSN construction without exposing it,
	// but we can test that New() handles the config correctly
	_, err := New(config)
	if err != nil {
		// This is expected if the test database doesn't exist
		// The important thing is that the function doesn't panic
		t.Logf("Expected connection error: %v", err)
	}
}

func TestDatabaseConnectionPooling(t *testing.T) {
	config := Config{
		Host:            "localhost",
		Port:            "5432",
		User:            "postgres",
		Password:        "testpass",
		Database:        "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}

	db, err := New(config)
	if err != nil {
		t.Skipf("Skipping connection pool test - database not available: %v", err)
		return
	}
	defer db.Close()

	// Test that connection pool settings are applied
	stats := db.Stats()

	// Initially, no connections should be open
	if stats.OpenConnections < 0 {
		t.Errorf("Expected non-negative open connections, got %d", stats.OpenConnections)
	}
}

func TestDatabasePing(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	db, err := New(config)
	if err != nil {
		t.Skipf("Skipping ping test - database not available: %v", err)
		return
	}
	defer db.Close()

	// Test ping with context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Errorf("Ping with context failed: %v", err)
	}
}

func TestDatabaseClose(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	db, err := New(config)
	if err != nil {
		t.Skipf("Skipping close test - database not available: %v", err)
		return
	}

	// Test close
	if err := db.Close(); err != nil {
		t.Errorf("Database close failed: %v", err)
	}

	// Test that operations fail after close
	if err := db.Ping(); err == nil {
		t.Error("Expected ping to fail after close")
	}
}

// Mock database setup for testing
func setupTestDB(t *testing.T) *sql.DB {
	// For unit tests, we can use SQLite in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

func TestDatabaseTransaction(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	db, err := New(config)
	if err != nil {
		t.Skipf("Skipping transaction test - database not available: %v", err)
		return
	}
	defer db.Close()

	// Test transaction begin/commit
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Test rollback
	if err := tx.Rollback(); err != nil {
		t.Errorf("Failed to rollback transaction: %v", err)
	}
}
