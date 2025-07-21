# Bookwork API

A complete RESTful API for the Bookwork book club management platform, built with Go.

## 🚀 Quick Start Guide

### Option 1: Docker (Recommended)
```bash
# 1. Quick setup with Docker
./scripts/setup-staging.sh --docker

# 2. Test the deployment
./scripts/integration-test.sh

# 3. Update your SvelteKit .env file
echo "VITE_API_BASE=http://localhost:8001/api" >> frontend/.env
```

### Option 2: Local Development
```bash
# 1. Setup local environment
./scripts/setup-staging.sh

# 2. Start the API
ENV_FILE=.env.staging ./bin/bookwork-api

# 3. Test in another terminal
./scripts/integration-test.sh
```

## 🔗 Integration with SvelteKit

Your SvelteKit app should connect to:
- **API Base URL**: `http://localhost:8001/api`
- **Health Check**: `http://localhost:8001/healthz`

### Example Frontend Configuration

**`.env`:**
```bash
VITE_API_BASE=http://localhost:8001/api
```

**API Service Example:**
```javascript
// src/lib/api.js
const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8001/api';

export async function apiCall(endpoint, options = {}) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  });
  
  return response.json();
}

// Login example
export async function login(email, password) {
  return apiCall('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}
```

## 🧪 Testing Your Integration

```bash
# Run comprehensive integration tests
./scripts/integration-test.sh

# Test specific API endpoint
curl http://localhost:8001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@bookwork.com", "password": "admin123"}'
```

## 📋 Default Credentials

- **Admin**: admin@bookwork.com / admin123
- **Database**: bookwork_staging (password auto-generated)

## Features

- **Authentication**: JWT-based authentication with refresh tokens
- **Club Management**: Club member management with role-based permissions
- **Event Management**: Create, update, and manage club events
- **Event Coordination**: Item assignment and availability tracking
- **Security**: Password hashing, input validation, SQL injection protection
- **Performance**: Database connection pooling, proper indexing

## Getting Started (Manual Setup)

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd bookwork-api
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your database credentials and JWT secret
```

4. Create and initialize the database:
```bash
# Create database
createdb bookwork

# Run the initialization script
psql -d bookwork -f init.sql
```

5. Run the application:
```bash
go run cmd/api/main.go
```

The API will be available at `http://localhost:8000/api`

## 🛠️ Common Commands

```bash
# View API logs (Docker)
docker-compose -f docker-compose.staging.yml logs -f api-staging

# Restart API (Docker)
docker-compose -f docker-compose.staging.yml restart api-staging

# Stop everything (Docker)
docker-compose -f docker-compose.staging.yml down

# Access database
psql -h localhost -p 5433 -U bookwork_staging -d bookwork_staging
```

## 🆘 Troubleshooting

| Problem | Solution |
|---------|----------|
| API not starting | Check `docker-compose logs` or local environment variables |
| CORS errors | Verify `CORS_ORIGINS` includes your frontend URL |
| Database connection failed | Ensure PostgreSQL is running and credentials are correct |
| Tests failing | Run `./scripts/integration-test.sh` for detailed diagnostics |

## 📚 What's Included

✅ Complete Go API with all endpoints from specification  
✅ PostgreSQL database with sample data  
✅ JWT authentication system  
✅ CORS configured for SvelteKit  
✅ Docker containerization  
✅ Integration test suite  
✅ Health monitoring  

## 🎯 Next Steps

1. **Start Development**: Your API is ready for SvelteKit integration
2. **Test Endpoints**: Use the integration test suite to verify functionality
3. **Build Frontend**: Connect your SvelteKit app to `http://localhost:8001/api`
4. **Deploy**: When ready, follow `STAGING_DEPLOYMENT_GUIDE.md` for cloud deployment

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8000` |
| `HOST` | Server host | `localhost` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | (required) |
| `DB_NAME` | Database name | `bookwork` |
| `DB_SSLMODE` | SSL mode | `disable` |
| `JWT_SECRET` | JWT secret key | (required, min 32 chars) |
| `JWT_ISSUER` | JWT issuer | `bookwork-api` |

## API Endpoints

### Authentication

- `POST /api/auth/login` - User login
- `POST /api/auth/refresh` - Refresh access token
- `POST /api/auth/validate` - Validate token (protected)
- `POST /api/auth/logout` - Logout (protected)

### Club Management

- `GET /api/club/{clubId}/members` - Get club members (protected)
- `POST /api/club/{clubId}/members` - Add club member (protected)
- `PUT /api/club/{clubId}/members/{memberId}` - Update member (protected)
- `DELETE /api/club/{clubId}/members/{memberId}` - Remove member (protected)

### Event Management

- `GET /api/club/{clubId}/events` - Get club events (protected)
- `POST /api/club/{clubId}/events` - Create event (protected)
- `PUT /api/events/{eventId}` - Update event (protected)
- `DELETE /api/events/{eventId}` - Delete event (protected)

### Event Coordination

- `GET /api/events/{eventId}/items` - Get event items (protected)
- `POST /api/events/{eventId}/items` - Create event item (protected)
- `PUT /api/events/{eventId}/items/{itemId}` - Update item (protected)
- `DELETE /api/events/{eventId}/items/{itemId}` - Delete item (protected)

### Availability Management

- `GET /api/events/{eventId}/availability` - Get availability (protected)
- `POST /api/events/{eventId}/availability` - Update availability (protected)

### Health Check

- `GET /healthz` - Health check endpoint

## Project Structure

```
bookwork-api/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── auth/
│   │   └── auth.go              # JWT authentication service
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── database/
│   │   └── database.go          # Database connection
│   ├── handlers/
│   │   ├── auth.go              # Authentication handlers
│   │   ├── availability.go      # Availability handlers
│   │   ├── club.go              # Club management handlers
│   │   ├── event_items.go       # Event items handlers
│   │   └── events.go            # Event management handlers
│   └── models/
│       └── models.go            # Data models and types
├── init.sql                     # Database schema
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── .env.example                 # Environment variables template
└── README.md                    # This file
```

## Testing

### Default Admin User

For testing purposes, a default admin user is created:
- Email: `admin@bookwork.com`
- Password: `admin123`
- Role: `admin`

### Sample API Requests

#### Login
```bash
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@bookwork.com",
    "password": "admin123"
  }'
```

#### Validate Token
```bash
curl -X POST http://localhost:8000/api/auth/validate \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## Security Features

- **Password Security**: Bcrypt hashing with salt
- **JWT Security**: HS256 algorithm, configurable expiration
- **SQL Injection Protection**: Parameterized queries
- **Input Validation**: Server-side validation for all inputs
- **Role-Based Access**: Owner, moderator, and member roles
- **Token Management**: Refresh token rotation and revocation

## Performance Optimizations

- Database connection pooling
- Proper database indexes
- Query optimization
- Response caching headers
- Request timeouts

## Production Deployment

1. Set secure environment variables:
   - Use a strong JWT secret (min 32 characters)
   - Enable SSL mode for database
   - Set appropriate CORS origins

2. Database optimizations:
   - Use connection pooling
   - Regular VACUUM and ANALYZE
   - Monitor query performance

3. Security checklist:
   - Use HTTPS in production
   - Set secure CORS policies
   - Implement rate limiting
   - Monitor for suspicious activity

## Contributing

1. Follow Go conventions and best practices
2. Write tests for new features
3. Update documentation for API changes
4. Use descriptive commit messages

## License

This project is licensed under the MIT License.
