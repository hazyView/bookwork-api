package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

type Migration struct {
	Version   int
	Name      string
	SQL       string
	AppliedAt *time.Time
}

type Migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// RunMigrations executes all pending migrations
func (m *Migrator) RunMigrations() error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Load all available migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedSet := make(map[int]bool)
	for _, version := range applied {
		appliedSet[version] = true
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if !appliedSet[migration.Version] {
			if err := m.applyMigration(migration); err != nil {
				return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
			}
			log.Printf("Applied migration %03d_%s", migration.Version, migration.Name)
		}
	}

	return nil
}

// RollbackMigration rolls back the last applied migration
func (m *Migrator) RollbackMigration() error {
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Get the last applied migration
	sort.Sort(sort.Reverse(sort.IntSlice(applied)))
	lastVersion := applied[0]

	// Remove from migrations table
	_, err = m.db.Exec("DELETE FROM schema_migrations WHERE version = $1", lastVersion)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	log.Printf("Rolled back migration %03d", lastVersion)
	log.Println("WARNING: Schema changes were not automatically reversed.")
	log.Println("Manual rollback may be required for data integrity.")

	return nil
}

// MigrateTo migrates to a specific version
func (m *Migrator) MigrateTo(targetVersion int) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Load all available migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Validate target version exists
	var targetExists bool
	for _, migration := range migrations {
		if migration.Version == targetVersion {
			targetExists = true
			break
		}
	}
	if !targetExists {
		return fmt.Errorf("migration version %d does not exist", targetVersion)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedSet := make(map[int]bool)
	for _, version := range applied {
		appliedSet[version] = true
	}

	// Apply migrations up to target version
	for _, migration := range migrations {
		if migration.Version <= targetVersion && !appliedSet[migration.Version] {
			if err := m.applyMigration(migration); err != nil {
				return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
			}
			log.Printf("Applied migration %03d_%s", migration.Version, migration.Name)
		}
	}

	return nil
}

// GetMigrationStatus returns applied and pending migrations
func (m *Migrator) GetMigrationStatus() ([]Migration, []Migration, error) {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return nil, nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Load all available migrations
	allMigrations, err := m.loadMigrations()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations with timestamps
	rows, err := m.db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	appliedMap := make(map[int]time.Time)
	for rows.Next() {
		var version int
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		appliedMap[version] = appliedAt
	}

	var applied, pending []Migration
	for _, migration := range allMigrations {
		if appliedAt, exists := appliedMap[migration.Version]; exists {
			migration.AppliedAt = &appliedAt
			applied = append(applied, migration)
		} else {
			pending = append(pending, migration)
		}
	}

	return applied, pending, nil
}

func (m *Migrator) createMigrationsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) loadMigrations() ([]Migration, error) {
	entries, err := sqlFiles.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filename := entry.Name()
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) != 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(parts[1], ".sql")

		content, err := sqlFiles.ReadFile(filepath.Join("sql", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m *Migrator) getAppliedMigrations() ([]int, error) {
	rows, err := m.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, nil
}

func (m *Migrator) applyMigration(migration Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	_, err = tx.Exec(migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}
