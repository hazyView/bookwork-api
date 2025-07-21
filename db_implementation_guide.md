# PostgreSQL 15+ Implementation Guide for Bookwork API

**Document Version**: 1.0  
**Created**: July 21, 2025  
**Author**: Principal Software Engineer & API Architect  
**Target Audience**: Software Engineers, Database Engineers, DevOps Engineers  

---

## 📋 Table of Contents

1. [Project Overview & Architecture](#project-overview--architecture)
2. [Prerequisites & Environment Setup](#prerequisites--environment-setup)
3. [Database Schema Implementation](#database-schema-implementation)
4. [Performance Optimization](#performance-optimization)
5. [Security Configuration](#security-configuration)
6. [Connection Management](#connection-management)
7. [Deployment Strategies](#deployment-strategies)
8. [Monitoring & Maintenance](#monitoring--maintenance)
9. [Troubleshooting Guide](#troubleshooting-guide)
10. [Production Checklist](#production-checklist)

---

## 🏗️ Project Overview & Architecture

### **System Architecture**
The Bookwork API is a Go-based REST API for book club management featuring:
- **Backend**: Go 1.21+ with Chi router
- **Database**: PostgreSQL 15+ with UUID primary keys
- **Authentication**: JWT-based with refresh tokens
- **Frontend**: SvelteKit (separate repository)
- **Deployment**: Docker Compose ready

### **Database Interaction Patterns**
```go
// The API uses these patterns extensively:
db.QueryRowContext(ctx, query, args...).Scan(...)
db.QueryContext(ctx, query, args...)
db.ExecContext(ctx, query, args...)
db.BeginTx(ctx, nil) // For transactions
```

### **Data Model Overview**
```
┌─────────────────────────────────────────────────────────────┐
│                    DATABASE RELATIONSHIPS                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  USERS ←1:N→ CLUBS ←1:N→ EVENTS ←1:N→ EVENT_ITEMS           │
│    ↓                       ↓                                │
│  REFRESH_TOKENS        AVAILABILITY                         │
│    ↓                                                        │
│  CLUB_MEMBERS                                               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔧 Prerequisites & Environment Setup

### **1. PostgreSQL Installation**

#### **Linux (Ubuntu/Debian)**
```bash
# Install PostgreSQL 15
sudo apt update
sudo apt install -y postgresql-15 postgresql-contrib-15

# Start and enable service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Verify installation
sudo -u postgres psql -c "SELECT version();"
```

#### **Linux (CentOS/RHEL)**
```bash
# Install PostgreSQL 15
sudo dnf install -y postgresql15-server postgresql15-contrib

# Initialize database
sudo postgresql-15-setup initdb

# Start and enable service
sudo systemctl start postgresql-15
sudo systemctl enable postgresql-15
```

#### **macOS**
```bash
# Using Homebrew
brew install postgresql@15
brew services start postgresql@15

# Or using Postgres.app
# Download from https://postgresapp.com/
```

#### **Windows**
```powershell
# Download installer from https://www.postgresql.org/download/windows/
# Or use Chocolatey
choco install postgresql15 --params '/Password:yourpassword'
```

### **2. Docker Setup (Recommended for Development)**
```bash
# Clone the repository
git clone <repository-url>
cd bookwork-api

# Start PostgreSQL with Docker
docker run --name bookwork-postgres \
  -e POSTGRES_DB=bookwork \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=yourpassword \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  -d postgres:15-alpine

# Verify container is running
docker ps
```

### **3. Environment Variables**
Create `.env` file in project root:
```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=bookwork
DB_SSLMODE=disable

# Application Configuration
PORT=8000
HOST=localhost
JWT_SECRET=your-super-secret-jwt-key-must-be-at-least-32-characters-long
JWT_ISSUER=bookwork-api
ENVIRONMENT=development
```

---

## 🗄️ Database Schema Implementation

### **Step 1: Connect to PostgreSQL**
```bash
# Connect to PostgreSQL
psql -h localhost -U postgres

# Or with Docker
docker exec -it bookwork-postgres psql -U postgres -d bookwork
```

### **Step 2: Enable Required Extensions**
```sql
-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enable array operations (optional but recommended)
CREATE EXTENSION IF NOT EXISTS "intarray";

-- Verify extensions
\dx
```

### **Step 3: Create Database Schema**

#### **Users Table (Core Authentication)**
```sql
-- Users Table - Primary authentication and user management
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    avatar VARCHAR(500),
    role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('admin', 'moderator', 'member', 'guest')),
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add constraints and indexes
CREATE UNIQUE INDEX idx_users_email_unique ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;
CREATE INDEX idx_users_last_login ON users(last_login_at);

-- Add email validation constraint
ALTER TABLE users ADD CONSTRAINT check_email_format 
    CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

-- Add updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

#### **Clubs Table (Book Club Management)**
```sql
-- Clubs Table - Book club management
CREATE TABLE clubs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_public BOOLEAN DEFAULT false,
    max_members INTEGER CHECK (max_members > 0),
    meeting_frequency VARCHAR(50),
    current_book VARCHAR(255),
    location VARCHAR(255),
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_clubs_owner ON clubs(owner_id);
CREATE INDEX idx_clubs_public ON clubs(is_public) WHERE is_public = true;
CREATE INDEX idx_clubs_name ON clubs(name);
CREATE INDEX idx_clubs_tags ON clubs USING GIN(tags);

-- Updated_at trigger
CREATE TRIGGER update_clubs_updated_at 
    BEFORE UPDATE ON clubs 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

#### **Club Members Table (Membership Management)**
```sql
-- Club Members Table - Manages club membership and roles
CREATE TABLE club_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id UUID REFERENCES clubs(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('admin', 'moderator', 'member', 'guest')),
    joined_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    books_read INTEGER DEFAULT 0 CHECK (books_read >= 0),
    is_active BOOLEAN DEFAULT true,
    
    -- Ensure unique membership
    UNIQUE(club_id, user_id)
);

-- Performance indexes
CREATE INDEX idx_club_members_club_id ON club_members(club_id);
CREATE INDEX idx_club_members_user_id ON club_members(user_id);
CREATE INDEX idx_club_members_active ON club_members(club_id, is_active) WHERE is_active = true;
CREATE INDEX idx_club_members_role ON club_members(club_id, role);
```

#### **Events Table (Event Management)**
```sql
-- Events Table - Book club events and meetings
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id UUID REFERENCES clubs(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    event_date DATE NOT NULL,
    event_time TIME NOT NULL,
    location VARCHAR(255) NOT NULL,
    book VARCHAR(255),
    type VARCHAR(50) DEFAULT 'discussion' CHECK (type IN ('discussion', 'meeting', 'social', 'planning', 'other')),
    max_attendees INTEGER CHECK (max_attendees > 0),
    is_public BOOLEAN DEFAULT false,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    attendees UUID[] DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure event is in the future (optional constraint)
    CONSTRAINT check_future_date CHECK (event_date >= CURRENT_DATE)
);

-- Performance indexes
CREATE INDEX idx_events_club_id ON events(club_id);
CREATE INDEX idx_events_date ON events(event_date);
CREATE INDEX idx_events_club_date ON events(club_id, event_date);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_created_by ON events(created_by);
CREATE INDEX idx_events_attendees ON events USING GIN(attendees);

-- Updated_at trigger
CREATE TRIGGER update_events_updated_at 
    BEFORE UPDATE ON events 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

#### **Event Items Table (Task Management)**
```sql
-- Event Items Table - Event coordination and task management
CREATE TABLE event_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(50) DEFAULT 'other' CHECK (category IN ('agenda', 'task', 'material', 'note', 'other')),
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled')),
    notes TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance indexes
CREATE INDEX idx_event_items_event_id ON event_items(event_id);
CREATE INDEX idx_event_items_assigned_to ON event_items(assigned_to);
CREATE INDEX idx_event_items_status ON event_items(status);
CREATE INDEX idx_event_items_category ON event_items(category);
CREATE INDEX idx_event_items_event_status ON event_items(event_id, status);

-- Updated_at trigger
CREATE TRIGGER update_event_items_updated_at 
    BEFORE UPDATE ON event_items 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

#### **Availability Table (RSVP Management)**
```sql
-- Availability Table - User availability for events
CREATE TABLE availability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('available', 'unavailable', 'maybe')),
    notes TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique availability per user per event
    UNIQUE(event_id, user_id)
);

-- Performance indexes
CREATE INDEX idx_availability_event_id ON availability(event_id);
CREATE INDEX idx_availability_user_id ON availability(user_id);
CREATE INDEX idx_availability_status ON availability(status);
CREATE INDEX idx_availability_event_status ON availability(event_id, status);
```

#### **Refresh Tokens Table (JWT Management)**
```sql
-- Refresh Tokens Table - JWT refresh token management
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_revoked BOOLEAN DEFAULT false,
    
    -- Ensure future expiration
    CONSTRAINT check_future_expiration CHECK (expires_at > CURRENT_TIMESTAMP)
);

-- Performance indexes
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id, expires_at, is_revoked) 
    WHERE is_revoked = false;

-- Cleanup expired tokens (run periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM refresh_tokens 
    WHERE expires_at < CURRENT_TIMESTAMP OR is_revoked = true;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
```

### **Step 4: Initial Data Setup**
```sql
-- Insert default admin user
INSERT INTO users (
    id, 
    name, 
    email, 
    password_hash, 
    role
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Admin User',
    'admin@bookwork.com',
    '$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGLcZEiGDMVr5yUP1KUOYTa', -- password: admin123
    'admin'
) ON CONFLICT (id) DO NOTHING;

-- Create sample club for testing
INSERT INTO clubs (
    id,
    name,
    description,
    owner_id,
    is_public
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Sample Book Club',
    'A sample book club for testing the API',
    '00000000-0000-0000-0000-000000000001',
    true
) ON CONFLICT (id) DO NOTHING;
```

---

## ⚡ Performance Optimization

### **1. Connection Pool Configuration**
```go
// internal/database/database.go enhancement
func New(config Config) (*DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        config.Host, config.Port, config.User, config.Password, 
        config.Database, config.SSLMode,
    )

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Optimized connection pool settings
    db.SetMaxOpenConns(25)                 // Max concurrent connections
    db.SetMaxIdleConns(10)                 // Keep connections alive
    db.SetConnMaxLifetime(5 * time.Minute) // Rotate connections
    db.SetConnMaxIdleTime(2 * time.Minute) // Close idle connections

    // Test connection with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &DB{db}, nil
}
```

### **2. Query Optimization**
```sql
-- Compound indexes for common query patterns
CREATE INDEX CONCURRENTLY idx_club_members_active_role 
    ON club_members(club_id, is_active, role) 
    WHERE is_active = true;

CREATE INDEX CONCURRENTLY idx_events_club_date_type 
    ON events(club_id, event_date, type);

CREATE INDEX CONCURRENTLY idx_availability_event_summary 
    ON availability(event_id, status) 
    INCLUDE (user_id);

-- Partial indexes for common filters
CREATE INDEX CONCURRENTLY idx_users_active_email 
    ON users(email) 
    WHERE is_active = true;

CREATE INDEX CONCURRENTLY idx_events_future 
    ON events(event_date, club_id) 
    WHERE event_date >= CURRENT_DATE;
```

### **3. PostgreSQL Configuration (postgresql.conf)**
```ini
# Memory Settings (adjust based on available RAM)
shared_buffers = 256MB                    # 25% of RAM for dedicated server
work_mem = 4MB                            # Per connection memory
maintenance_work_mem = 64MB               # For maintenance operations
effective_cache_size = 1GB                # OS cache size estimate

# Connection Settings
max_connections = 200                     # Adjust based on load
superuser_reserved_connections = 3

# Write-Ahead Logging (WAL)
wal_level = replica                       # For replication support
max_wal_size = 1GB
min_wal_size = 80MB
checkpoint_completion_target = 0.9

# Query Planning
default_statistics_target = 100           # More detailed statistics
random_page_cost = 1.1                    # For SSD storage
effective_io_concurrency = 200            # For SSD storage

# Logging (for production monitoring)
log_destination = 'csvlog'
logging_collector = on
log_directory = 'pg_log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_statement = 'mod'                     # Log modifications
log_duration = on
log_min_duration_statement = 1000         # Log slow queries (1 second)
```

### **4. Database Maintenance Scripts**
```sql
-- Create maintenance functions
CREATE OR REPLACE FUNCTION perform_maintenance()
RETURNS TEXT AS $$
DECLARE
    result TEXT := '';
    rec RECORD;
BEGIN
    -- Update table statistics
    FOR rec IN SELECT schemaname, tablename FROM pg_tables 
               WHERE schemaname = 'public' LOOP
        EXECUTE 'ANALYZE ' || quote_ident(rec.schemaname) || '.' || quote_ident(rec.tablename);
        result := result || 'Analyzed ' || rec.tablename || E'\n';
    END LOOP;
    
    -- Clean up expired refresh tokens
    PERFORM cleanup_expired_tokens();
    result := result || 'Cleaned up expired refresh tokens' || E'\n';
    
    -- Vacuum analyze main tables
    VACUUM ANALYZE users;
    VACUUM ANALYZE clubs;
    VACUUM ANALYZE events;
    result := result || 'Vacuumed main tables' || E'\n';
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Schedule maintenance (run daily via cron)
-- 0 2 * * * psql -d bookwork -c "SELECT perform_maintenance();"
```

---

## 🔐 Security Configuration

### **1. Database User Management**
```sql
-- Create application-specific user
CREATE USER bookwork_app WITH PASSWORD 'secure_app_password_here';

-- Grant minimal required permissions
GRANT CONNECT ON DATABASE bookwork TO bookwork_app;
GRANT USAGE ON SCHEMA public TO bookwork_app;

-- Grant table permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO bookwork_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO bookwork_app;

-- Grant function execution permissions
GRANT EXECUTE ON FUNCTION cleanup_expired_tokens() TO bookwork_app;
GRANT EXECUTE ON FUNCTION perform_maintenance() TO bookwork_app;

-- Create read-only user for reporting/monitoring
CREATE USER bookwork_readonly WITH PASSWORD 'secure_readonly_password_here';
GRANT CONNECT ON DATABASE bookwork TO bookwork_readonly;
GRANT USAGE ON SCHEMA public TO bookwork_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO bookwork_readonly;
```

### **2. SSL/TLS Configuration**
```bash
# Generate SSL certificates (self-signed for development)
sudo openssl req -new -x509 -days 365 -nodes -text \
  -out /etc/ssl/certs/server.crt \
  -keyout /etc/ssl/private/server.key \
  -subj "/CN=localhost"

sudo chown postgres:postgres /etc/ssl/private/server.key
sudo chmod 600 /etc/ssl/private/server.key
```

```ini
# postgresql.conf SSL settings
ssl = on
ssl_cert_file = '/etc/ssl/certs/server.crt'
ssl_key_file = '/etc/ssl/private/server.key'
ssl_protocols = 'TLSv1.2,TLSv1.3'
ssl_ciphers = 'HIGH:MEDIUM:+3DES:!aNULL'
ssl_prefer_server_ciphers = on
```

### **3. pg_hba.conf Security**
```bash
# pg_hba.conf - Host-based authentication
# TYPE  DATABASE        USER            ADDRESS                 METHOD

# "local" is for Unix domain socket connections only
local   all             postgres                                peer
local   all             bookwork_app                            md5
local   all             bookwork_readonly                       md5

# IPv4 local connections:
host    bookwork        bookwork_app    127.0.0.1/32            md5
host    bookwork        bookwork_readonly 127.0.0.1/32          md5

# IPv6 local connections:
host    bookwork        bookwork_app    ::1/128                 md5

# SSL connections for remote access (production)
hostssl bookwork        bookwork_app    0.0.0.0/0               md5
hostssl bookwork        bookwork_readonly 0.0.0.0/0             md5

# Deny all other connections
host    all             all             0.0.0.0/0               reject
```

### **4. Row Level Security (RLS)**
```sql
-- Enable RLS on sensitive tables (optional but recommended)
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE clubs ENABLE ROW LEVEL SECURITY;
ALTER TABLE club_members ENABLE ROW LEVEL SECURITY;

-- Create policies (example for club_members)
CREATE POLICY club_members_policy ON club_members
    FOR ALL TO bookwork_app
    USING (
        user_id = current_setting('app.current_user_id')::UUID OR
        EXISTS (
            SELECT 1 FROM club_members cm 
            WHERE cm.club_id = club_members.club_id 
            AND cm.user_id = current_setting('app.current_user_id')::UUID
            AND cm.role IN ('admin', 'moderator')
        )
    );

-- Set user context in application (Go code)
-- db.ExecContext(ctx, "SET app.current_user_id = $1", userID)
```

---

## 🔄 Connection Management

### **1. Connection Pooling with PgBouncer (Production)**
```ini
# /etc/pgbouncer/pgbouncer.ini
[databases]
bookwork = host=localhost port=5432 dbname=bookwork user=bookwork_app

[pgbouncer]
listen_port = 6432
listen_addr = 127.0.0.1
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt

# Connection pooling settings
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 50
max_db_connections = 100
reserve_pool_size = 10

# Timeouts
server_reset_query = DISCARD ALL
server_check_delay = 30
server_check_query = SELECT 1
server_lifetime = 3600
server_idle_timeout = 600

# Logging
admin_users = pgbouncer_admin
log_connections = 1
log_disconnections = 1
log_pooler_errors = 1
```

```bash
# /etc/pgbouncer/userlist.txt
"bookwork_app" "md5hashed_password_here"
"pgbouncer_admin" "md5hashed_admin_password_here"
```

### **2. Application Connection Configuration**
```go
// internal/config/config.go - Enhanced database config
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
    PgBouncerAddr   string // For production with PgBouncer
}

func Load() (*Config, error) {
    config := &Config{
        Database: DatabaseConfig{
            Host:            getEnv("DB_HOST", "localhost"),
            Port:            getEnv("DB_PORT", "5432"),
            User:            getEnv("DB_USER", "bookwork_app"),
            Password:        getEnv("DB_PASSWORD", ""),
            Database:        getEnv("DB_NAME", "bookwork"),
            SSLMode:         getEnv("DB_SSLMODE", "require"),
            MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
            MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
            ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
            ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", "2m"),
            PgBouncerAddr:   getEnv("PGBOUNCER_ADDR", ""),
        },
    }
    return config, nil
}
```

---

## 🚀 Deployment Strategies

### **1. Development Setup**
```bash
# Using Docker Compose (from project root)
docker-compose up -d

# Or using the provided script
chmod +x scripts/setup-staging.sh
./scripts/setup-staging.sh --docker
```

### **2. Production Deployment (AWS RDS)**
```bash
# Create RDS instance
aws rds create-db-instance \
    --db-instance-identifier bookwork-prod \
    --db-instance-class db.t3.medium \
    --engine postgres \
    --engine-version 15.4 \
    --master-username bookwork_admin \
    --master-user-password 'SecurePassword123!' \
    --allocated-storage 100 \
    --storage-type gp2 \
    --vpc-security-group-ids sg-xxxxxxxxx \
    --db-subnet-group-name bookwork-subnet-group \
    --backup-retention-period 7 \
    --storage-encrypted \
    --monitoring-interval 60 \
    --enable-performance-insights \
    --performance-insights-retention-period 7
```

### **3. Production Environment Variables**
```bash
# Production .env file
DB_HOST=bookwork-prod.xxxxxxxxx.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_USER=bookwork_app
DB_PASSWORD=SecureProductionPassword123!
DB_NAME=bookwork
DB_SSLMODE=require
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME=10m
DB_CONN_MAX_IDLE_TIME=5m

JWT_SECRET=ultra-secure-jwt-secret-key-for-production-environment-change-this
JWT_ISSUER=bookwork-api-production
ENVIRONMENT=production
```

### **4. Database Migration Strategy**
```sql
-- Create migrations table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Example migration structure
-- migrations/001_initial_schema.sql
-- migrations/002_add_user_preferences.sql
-- migrations/003_add_club_settings.sql
```

```go
// internal/migrations/migrate.go
package migrations

import (
    "database/sql"
    "fmt"
    "io/fs"
    "path/filepath"
    "sort"
    "strings"
)

func RunMigrations(db *sql.DB, migrationsFS fs.FS) error {
    // Create migrations table if not exists
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR(50) PRIMARY KEY,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return fmt.Errorf("failed to create migrations table: %w", err)
    }

    // Get applied migrations
    applied := make(map[string]bool)
    rows, err := db.Query("SELECT version FROM schema_migrations")
    if err != nil {
        return fmt.Errorf("failed to query migrations: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var version string
        if err := rows.Scan(&version); err != nil {
            return err
        }
        applied[version] = true
    }

    // Find and sort migration files
    var files []string
    fs.WalkDir(migrationsFS, ".", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if strings.HasSuffix(path, ".sql") {
            files = append(files, path)
        }
        return nil
    })
    sort.Strings(files)

    // Apply unapplied migrations
    for _, file := range files {
        version := strings.TrimSuffix(filepath.Base(file), ".sql")
        if applied[version] {
            continue
        }

        content, err := fs.ReadFile(migrationsFS, file)
        if err != nil {
            return fmt.Errorf("failed to read migration %s: %w", file, err)
        }

        tx, err := db.Begin()
        if err != nil {
            return fmt.Errorf("failed to begin transaction: %w", err)
        }

        _, err = tx.Exec(string(content))
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("failed to apply migration %s: %w", file, err)
        }

        _, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("failed to record migration %s: %w", file, err)
        }

        if err := tx.Commit(); err != nil {
            return fmt.Errorf("failed to commit migration %s: %w", file, err)
        }

        fmt.Printf("Applied migration: %s\n", version)
    }

    return nil
}
```

---

## 📊 Monitoring & Maintenance

### **1. Database Health Monitoring**
```sql
-- Create monitoring views
CREATE OR REPLACE VIEW db_health_summary AS
SELECT 
    'connections' as metric,
    COUNT(*) as value,
    (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_value
FROM pg_stat_activity
WHERE state = 'active'
UNION ALL
SELECT 
    'database_size_mb' as metric,
    pg_database_size(current_database()) / 1024 / 1024 as value,
    NULL as max_value
UNION ALL
SELECT 
    'active_users_24h' as metric,
    COUNT(DISTINCT user_id) as value,
    NULL as max_value
FROM users 
WHERE last_login_at > NOW() - INTERVAL '24 hours'
UNION ALL
SELECT 
    'events_this_month' as metric,
    COUNT(*) as value,
    NULL as max_value
FROM events 
WHERE created_at >= date_trunc('month', NOW());

-- Query performance monitoring
CREATE OR REPLACE VIEW slow_queries AS
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 20;
```

### **2. Automated Backup Strategy**
```bash
#!/bin/bash
# backup_database.sh
set -e

DB_NAME="bookwork"
DB_USER="bookwork_app" 
BACKUP_DIR="/var/backups/postgresql"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/bookwork_${DATE}.sql"
S3_BUCKET="bookwork-db-backups"

# Create backup directory
mkdir -p $BACKUP_DIR

# Create database backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME --no-password > $BACKUP_FILE

# Compress backup
gzip $BACKUP_FILE
BACKUP_FILE="${BACKUP_FILE}.gz"

# Upload to S3 (if configured)
if [ ! -z "$S3_BUCKET" ]; then
    aws s3 cp $BACKUP_FILE s3://$S3_BUCKET/daily/
fi

# Keep only last 7 days locally
find $BACKUP_DIR -name "bookwork_*.sql.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE"
```

```bash
# Add to crontab for daily backups at 2 AM
# crontab -e
0 2 * * * /path/to/backup_database.sh
```

### **3. Performance Monitoring Queries**
```sql
-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public' 
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes 
ORDER BY idx_scan DESC;

-- Check for unused indexes
SELECT 
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) as size
FROM pg_stat_user_indexes 
WHERE idx_scan = 0 
ORDER BY pg_relation_size(indexrelid) DESC;

-- Connection activity
SELECT 
    state,
    COUNT(*) as connections,
    AVG(EXTRACT(epoch FROM (NOW() - query_start))) as avg_duration_seconds
FROM pg_stat_activity 
WHERE state IS NOT NULL
GROUP BY state;
```

---

## 🔧 Troubleshooting Guide

### **Common Issues and Solutions**

#### **1. Connection Issues**
```bash
# Problem: "could not connect to server"
# Solution: Check PostgreSQL service status
sudo systemctl status postgresql
sudo systemctl start postgresql

# Problem: "password authentication failed"
# Solution: Check pg_hba.conf and user permissions
sudo -u postgres psql -c "SELECT rolname FROM pg_roles WHERE rolname = 'bookwork_app';"

# Problem: "too many connections"
# Solution: Check current connections and adjust max_connections
SELECT COUNT(*) FROM pg_stat_activity;
```

#### **2. Performance Issues**
```sql
-- Identify slow queries
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    (total_time/sum(total_time) OVER()) * 100 as percent_total
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;

-- Check for missing indexes
SELECT 
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    seq_tup_read / seq_scan as avg_tup_read
FROM pg_stat_user_tables 
WHERE seq_scan > 0 
ORDER BY seq_tup_read DESC 
LIMIT 10;

-- Analyze table statistics
ANALYZE VERBOSE;
```

#### **3. Storage Issues**
```bash
# Check disk space
df -h

# Check database size
psql -d bookwork -c "SELECT pg_size_pretty(pg_database_size('bookwork'));"

# Clean up old data
psql -d bookwork -c "SELECT cleanup_expired_tokens();"
```

#### **4. Application Integration Issues**
```go
// Enable query logging for debugging
import "log"

func debugQuery(query string, args ...interface{}) {
    if os.Getenv("DEBUG_SQL") == "true" {
        log.Printf("SQL: %s, Args: %v", query, args)
    }
}

// Add to handler methods
debugQuery(query, args...)
rows, err := h.db.QueryContext(ctx, query, args...)
```

---

## ✅ Production Checklist

### **Pre-Deployment Checklist**

#### **Database Configuration**
- [ ] PostgreSQL 15+ installed and running
- [ ] Database users created with minimal permissions
- [ ] SSL/TLS enabled and configured
- [ ] Connection pooling configured (PgBouncer)
- [ ] Backup strategy implemented
- [ ] Monitoring and alerting configured

#### **Security**
- [ ] pg_hba.conf properly configured
- [ ] Firewall rules applied
- [ ] SSL certificates valid and configured
- [ ] Database credentials stored securely
- [ ] Row Level Security policies applied (if needed)

#### **Performance**
- [ ] All indexes created and optimized
- [ ] postgresql.conf tuned for environment
- [ ] Connection pool settings optimized
- [ ] Query performance tested under load
- [ ] Auto-vacuum configured

#### **Operations**
- [ ] Backup and restore procedures tested
- [ ] Migration strategy documented
- [ ] Monitoring dashboards created
- [ ] Alerting thresholds configured
- [ ] Log rotation configured

#### **Application Integration**
- [ ] Environment variables properly set
- [ ] Database connection tested
- [ ] All API endpoints tested
- [ ] Error handling verified
- [ ] Performance requirements met (<200ms)

### **Go Live Checklist**
1. **Final backup of development data**
2. **Deploy database schema to production**
3. **Run initial data migration**
4. **Deploy application with production config**
5. **Verify all endpoints functionality**
6. **Monitor performance and error rates**
7. **Document any issues and resolutions**

---

## 📞 Support and Additional Resources

### **Documentation Links**
- [PostgreSQL 15 Documentation](https://www.postgresql.org/docs/15/)
- [Go PostgreSQL Driver (lib/pq)](https://github.com/lib/pq)
- [pgx - Alternative Go Driver](https://github.com/jackc/pgx)
- [PgBouncer Documentation](https://www.pgbouncer.org/)

### **Monitoring Tools**
- **pgAdmin**: Web-based PostgreSQL administration
- **pg_stat_statements**: Query performance statistics
- **Prometheus + Grafana**: Comprehensive monitoring
- **Datadog/New Relic**: Application Performance Monitoring

### **Best Practices**
1. **Always use connection pooling in production**
2. **Monitor query performance regularly**
3. **Keep statistics up to date with ANALYZE**
4. **Use prepared statements for frequently executed queries**
5. **Implement proper error handling and retry logic**
6. **Regular security audits and updates**
7. **Test backup and recovery procedures**

---

**End of Document**

This comprehensive guide provides everything needed to implement PostgreSQL 15+ for the Bookwork API project. Follow each section carefully, and refer back to this document for ongoing maintenance and troubleshooting.

For questions or issues not covered in this guide, consult the PostgreSQL documentation or seek assistance from database administrators familiar with PostgreSQL.
