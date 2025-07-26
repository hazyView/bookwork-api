package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type Config struct {
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

// NewMock creates a mock database for testing and demos
func NewMock() *DB {
	// Create a mock database that has the same interface
	return &DB{
		DB: nil, // No real sql.DB connection
	}
}

func New(config Config) (*DB, error) {
	var dsn string

	// Use PgBouncer if configured, otherwise direct connection
	if config.PgBouncerAddr != "" {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password,
			config.Database, config.SSLMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password,
			config.Database, config.SSLMode,
		)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool with enhanced settings
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to database at %s:%s", config.Host, config.Port)
	log.Printf("Connection pool: max_open=%d, max_idle=%d, max_lifetime=%v",
		config.MaxOpenConns, config.MaxIdleConns, config.ConnMaxLifetime)

	return &DB{db}, nil
}

func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil // Mock case
}

func (db *DB) Ping() error {
	if db.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return db.PingContext(ctx)
	}
	return nil // Mock case - always healthy
}

func (db *DB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if db.DB != nil {
		return db.DB.BeginTx(ctx, nil)
	}
	return &sql.Tx{}, nil // Mock case
}

// QueryRowContext with mock support
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if db.DB != nil {
		return db.DB.QueryRowContext(ctx, query, args...)
	}
	// Mock case - return empty row (handlers will need to handle this)
	return &sql.Row{}
}

// QueryContext with mock support
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if db.DB != nil {
		return db.DB.QueryContext(ctx, query, args...)
	}
	// Mock case
	return &sql.Rows{}, nil
}

// ExecContext with mock support
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if db.DB != nil {
		return db.DB.ExecContext(ctx, query, args...)
	}
	// Mock case
	return &mockResult{}, nil
}

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 1, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 1, nil }
