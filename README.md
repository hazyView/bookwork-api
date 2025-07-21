# Bookwork API

A robust Go-based API for managing book clubs, built with enterprise-grade PostgreSQL integration, comprehensive monitoring, and production-ready database migrations.

## 🚀 Features

### Core Functionality
- **User Authentication**: JWT-based authentication with refresh tokens
- **Club Management**: Create, manage, and moderate book clubs
- **Event Management**: Schedule discussions, meetings, and book-related events
- **Availability Tracking**: Member availability for events
- **Member Management**: Role-based access control (admin, moderator, member)

### Database Features
- **PostgreSQL 15+**: Advanced database with UUID primary keys, array support, and full-text search
- **Automated Migrations**: Embedded filesystem-based migration system with rollback support
- **Connection Pooling**: Optimized database connections with PgBouncer support
- **Performance Monitoring**: Built-in database metrics and slow query monitoring
- **Health Checks**: Comprehensive API and database health monitoring endpoints
- **Security**: Row-level security, audit logging, and password policies

### Monitoring & Observability
- **Health Endpoints**: Real-time API and database health status
- **Metrics Dashboard**: Database performance, connection stats, and query analysis
- **Slow Query Detection**: Automatic identification of performance bottlenecks
- **Lock Monitoring**: Database lock detection and resolution
- **User Activity Tracking**: Detailed user session and query monitoring

## 🏗️ Architecture

### Technology Stack
- **Backend**: Go 1.21+ with Chi router
- **Database**: PostgreSQL 15+ with advanced indexing and monitoring
- **Authentication**: JWT with RS256 signing
- **Containerization**: Docker with multi-stage builds
- **Migration System**: Embedded SQL files with version control

### Database Schema
- **Users**: Authentication and profile management
- **Clubs**: Book club organization with metadata
- **Club Members**: Role-based membership with reading statistics
- **Events**: Scheduled activities with location and book details
- **Event Items**: Task and material management for events
- **Availability**: Member scheduling and attendance tracking

## 📋 Prerequisites

- **Go**: 1.21 or higher
- **PostgreSQL**: 15+ (required for advanced features)
- **Docker**: For containerized deployment (optional)
- **Make**: For build automation

## 🛠️ Installation & Setup

### 1. Clone the Repository
```bash
git clone <repository-url>
cd bookwork-api
```

### 2. Install Dependencies
```bash
make deps
```

### 3. Database Setup

#### Option A: Local PostgreSQL Installation
```bash
# Install PostgreSQL 15+
brew install postgresql@15  # macOS
# or
sudo apt-get install postgresql-15  # Ubuntu

# Create database
createdb bookwork
```

#### Option B: Docker PostgreSQL
```bash
docker run --name bookwork-postgres \
  -e POSTGRES_DB=bookwork \
  -e POSTGRES_USER=bookwork \
  -e POSTGRES_PASSWORD=your_password \
  -p 5432:5432 \
  -d postgres:15
```

### 4. Environment Configuration
Create `.env` file in the project root:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=bookwork
DB_PASSWORD=your_password
DB_NAME=bookwork
DB_SSLMODE=disable

# Database Pool Settings (Production-ready defaults)
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=300s
DB_CONN_MAX_IDLE_TIME=900s

# Optional: PgBouncer for connection pooling
DB_PGBOUNCER_ADDR=localhost:6432

# JWT Configuration
JWT_SECRET_KEY=your-super-secret-jwt-key-change-this-in-production
JWT_ISSUER=bookwork-api

# Server Configuration
SERVER_PORT=8000
SERVER_HOST=localhost
```

### 5. Database Migration
```bash
# Build migration tool and run all migrations
make migrate-up

# Check migration status
make migrate-info

# View available migration commands
make help | grep migrate
```

### 6. Build and Run
```bash
# Build and run locally
make run

# Or run in development mode (with hot reload)
make run-dev
```

## 📊 Database Management

### Migration Commands
```bash
# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Migrate to specific version
make migrate-to VERSION=003

# Show current migration status
make migrate-info

# Fresh database (DESTRUCTIVE - for development only)
make migrate-fresh
```

### Sample Data
The migration system includes comprehensive sample data:
- Default admin user (admin@bookwork.com / admin123)
- Sample book club with members
- Scheduled events and tasks
- User activity examples

### Database Monitoring
Access comprehensive monitoring endpoints:

- **Health Check**: `GET /api/health`
- **Database Metrics**: `GET /api/metrics`
- **Table Statistics**: `GET /api/metrics/tables`
- **Slow Queries**: `GET /api/metrics/slow-queries?limit=20`

## 🔧 Development

### Code Quality
```bash
# Format code
make fmt

# Run linting
make lint

# Run tests
make test

# Generate test coverage report
make test-coverage
```

### Docker Development
```bash
# Build Docker image
make docker-build

# Run in container
make docker-run

# Full Docker Compose stack
docker-compose up -d
```

## 📈 Monitoring & Maintenance

### Performance Monitoring
The API includes built-in monitoring views:

- **Database Health Summary**: Overall database status and performance
- **Slow Query Analysis**: Identify performance bottlenecks
- **Connection Monitoring**: Track database connection usage
- **Index Usage Statistics**: Optimize database performance
- **Lock Detection**: Identify and resolve database locks

### Maintenance Functions
Automated maintenance functions included:

- **Token Cleanup**: Remove expired authentication tokens
- **Health Metrics**: Calculate database performance metrics
- **Performance Analysis**: Generate optimization recommendations

### Production Deployment
```bash
# Build for production
make build-linux

# Setup staging environment
make staging-setup

# Run integration tests
make test-integration
```

## 🔐 Security

### Database Security
- Row-level security policies
- Audit logging for sensitive operations
- Password policy enforcement
- Connection encryption support

### API Security
- JWT authentication with refresh tokens
- Rate limiting (configurable)
- CORS protection
- Security headers middleware

## 📚 API Documentation

### Authentication Endpoints
```
POST /api/auth/login      - User login
POST /api/auth/refresh    - Token refresh
POST /api/auth/logout     - User logout
POST /api/auth/validate   - Token validation
```

### Monitoring Endpoints
```
GET  /api/health                    - API health status
GET  /api/metrics                   - Complete database metrics
GET  /api/metrics/tables            - Table statistics
GET  /api/metrics/slow-queries      - Slow query analysis
```

### Core API Endpoints
```
GET  /api/club/{clubId}/members     - List club members
POST /api/club/{clubId}/members     - Add club member
GET  /api/club/{clubId}/events      - List club events
POST /api/club/{clubId}/events      - Create new event
```

---
