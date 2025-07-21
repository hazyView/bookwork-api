#!/bin/bash
set -e

echo "🚀 Bookwork API - Quick Staging Setup"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
DB_PASSWORD=""
JWT_SECRET=""
SETUP_TYPE="local"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --docker)
            SETUP_TYPE="docker"
            shift
            ;;
        --cloud)
            SETUP_TYPE="cloud"
            shift
            ;;
        --db-password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        --jwt-secret)
            JWT_SECRET="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --docker           Setup using Docker Compose"
            echo "  --cloud            Setup for cloud deployment"
            echo "  --db-password      Database password (will prompt if not provided)"
            echo "  --jwt-secret       JWT secret (will generate if not provided)"
            echo "  -h, --help         Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option $1"
            exit 1
            ;;
    esac
done

# Helper functions
generate_password() {
    echo $(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
}

generate_jwt_secret() {
    echo "staging-jwt-$(openssl rand -base64 32 | tr -d "=+/")-$(date +%s)"
}

check_dependencies() {
    echo -e "${YELLOW}Checking dependencies...${NC}"
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker is not installed${NC}"
        echo "Please install Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    if [[ "$SETUP_TYPE" == "docker" ]] && ! command -v docker-compose &> /dev/null; then
        echo -e "${RED}❌ Docker Compose is not installed${NC}"
        echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
        exit 1
    fi
    
    if [[ "$SETUP_TYPE" == "local" ]] && ! command -v psql &> /dev/null; then
        echo -e "${RED}❌ PostgreSQL client is not installed${NC}"
        echo "Please install PostgreSQL: https://postgresql.org/download/"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Dependencies check passed${NC}"
}

setup_environment() {
    echo -e "${YELLOW}Setting up environment configuration...${NC}"
    
    # Generate secrets if not provided
    if [[ -z "$DB_PASSWORD" ]]; then
        DB_PASSWORD=$(generate_password)
        echo -e "${YELLOW}Generated database password${NC}"
    fi
    
    if [[ -z "$JWT_SECRET" ]]; then
        JWT_SECRET=$(generate_jwt_secret)
        echo -e "${YELLOW}Generated JWT secret${NC}"
    fi
    
    # Create staging environment file
    cat > .env.staging << EOF
# Bookwork API Staging Environment
# Generated on $(date)

# Server Configuration
PORT=8001
HOST=0.0.0.0
ENVIRONMENT=staging

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=bookwork_staging
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=bookwork_staging
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=${JWT_SECRET}
JWT_ISSUER=bookwork-api-staging

# CORS Configuration
CORS_ORIGINS=http://localhost:5173,http://localhost:3000

# Logging
LOG_LEVEL=debug
EOF

    echo -e "${GREEN}✅ Environment file created: .env.staging${NC}"
}

setup_local() {
    echo -e "${YELLOW}Setting up local staging environment...${NC}"
    
    # Check if PostgreSQL is running
    if ! sudo systemctl is-active --quiet postgresql; then
        echo -e "${YELLOW}Starting PostgreSQL...${NC}"
        sudo systemctl start postgresql
    fi
    
    # Create database and user
    echo -e "${YELLOW}Creating staging database...${NC}"
    sudo -u postgres psql << EOF
DROP DATABASE IF EXISTS bookwork_staging;
DROP USER IF EXISTS bookwork_staging;
CREATE USER bookwork_staging WITH PASSWORD '${DB_PASSWORD}';
CREATE DATABASE bookwork_staging OWNER bookwork_staging;
GRANT ALL PRIVILEGES ON DATABASE bookwork_staging TO bookwork_staging;
\q
EOF
    
    # Initialize schema
    echo -e "${YELLOW}Initializing database schema...${NC}"
    PGPASSWORD=${DB_PASSWORD} psql -h localhost -U bookwork_staging -d bookwork_staging -f init.sql
    
    echo -e "${GREEN}✅ Local database setup complete${NC}"
}

setup_docker() {
    echo -e "${YELLOW}Setting up Docker staging environment...${NC}"
    
    # Create Docker Compose file for staging
    cat > docker-compose.staging.yml << EOF
version: '3.8'

services:
  db-staging:
    image: postgres:15-alpine
    container_name: bookwork-db-staging
    environment:
      POSTGRES_DB: bookwork_staging
      POSTGRES_USER: bookwork_staging
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports:
      - "5433:5432"
    volumes:
      - postgres_staging_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - bookwork-staging
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bookwork_staging"]
      interval: 10s
      timeout: 5s
      retries: 5

  api-staging:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: bookwork-api-staging
    environment:
      - PORT=8000
      - HOST=0.0.0.0
      - DB_HOST=db-staging
      - DB_PORT=5432
      - DB_USER=bookwork_staging
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=bookwork_staging
      - DB_SSLMODE=disable
      - JWT_SECRET=${JWT_SECRET}
      - JWT_ISSUER=bookwork-api-staging
      - ENVIRONMENT=staging
    ports:
      - "8001:8000"
    depends_on:
      db-staging:
        condition: service_healthy
    networks:
      - bookwork-staging
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_staging_data:

networks:
  bookwork-staging:
    name: bookwork-staging-network
EOF

    # Build and start
    echo -e "${YELLOW}Building and starting Docker containers...${NC}"
    docker-compose -f docker-compose.staging.yml up -d --build
    
    echo -e "${GREEN}✅ Docker staging environment started${NC}"
}

build_api() {
    echo -e "${YELLOW}Building API...${NC}"
    
    if command -v make &> /dev/null; then
        make build
    else
        go build -o bin/bookwork-api cmd/api/main.go
    fi
    
    echo -e "${GREEN}✅ API built successfully${NC}"
}

test_deployment() {
    echo -e "${YELLOW}Testing deployment...${NC}"
    
    # Wait for services to be ready
    echo "Waiting for API to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:8001/healthz > /dev/null 2>&1; then
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    
    # Test health endpoint
    echo -e "${YELLOW}Testing health endpoint...${NC}"
    HEALTH_RESPONSE=$(curl -s http://localhost:8001/healthz)
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}✅ Health check passed${NC}"
        echo "Response: $HEALTH_RESPONSE"
    else
        echo -e "${RED}❌ Health check failed${NC}"
        return 1
    fi
    
    # Test admin login
    echo -e "${YELLOW}Testing admin login...${NC}"
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8001/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"email": "admin@bookwork.com", "password": "admin123"}')
    
    if echo "$LOGIN_RESPONSE" | grep -q '"success":true'; then
        echo -e "${GREEN}✅ Admin login test passed${NC}"
        
        # Extract and validate token
        TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4)
        if [[ -n "$TOKEN" ]]; then
            echo -e "${GREEN}✅ JWT token generated successfully${NC}"
            
            # Test token validation
            VALIDATE_RESPONSE=$(curl -s -X POST http://localhost:8001/api/auth/validate \
                -H "Authorization: Bearer $TOKEN")
            
            if echo "$VALIDATE_RESPONSE" | grep -q '"success":true'; then
                echo -e "${GREEN}✅ Token validation test passed${NC}"
            else
                echo -e "${RED}❌ Token validation test failed${NC}"
                echo "Response: $VALIDATE_RESPONSE"
            fi
        fi
    else
        echo -e "${RED}❌ Admin login test failed${NC}"
        echo "Response: $LOGIN_RESPONSE"
        return 1
    fi
}

show_summary() {
    echo ""
    echo -e "${GREEN}🎉 Staging deployment completed successfully!${NC}"
    echo "======================================"
    echo ""
    echo -e "${YELLOW}📋 Deployment Summary:${NC}"
    echo "• API URL: http://localhost:8001"
    echo "• Health Check: http://localhost:8001/healthz"
    echo "• Environment: staging"
    echo "• Database: bookwork_staging"
    
    if [[ "$SETUP_TYPE" == "docker" ]]; then
        echo "• pgAdmin: http://localhost:8081"
    fi
    
    echo ""
    echo -e "${YELLOW}🔐 Credentials:${NC}"
    echo "• Admin Email: admin@bookwork.com"
    echo "• Admin Password: admin123"
    echo "• Database User: bookwork_staging"
    echo "• Database Password: ${DB_PASSWORD}"
    
    echo ""
    echo -e "${YELLOW}🛠️  Management Commands:${NC}"
    
    if [[ "$SETUP_TYPE" == "docker" ]]; then
        echo "• View logs: docker-compose -f docker-compose.staging.yml logs -f"
        echo "• Stop services: docker-compose -f docker-compose.staging.yml down"
        echo "• Restart API: docker-compose -f docker-compose.staging.yml restart api-staging"
    else
        echo "• Start API: ENV_FILE=.env.staging ./bin/bookwork-api"
        echo "• Check database: psql -h localhost -U bookwork_staging -d bookwork_staging"
    fi
    
    echo ""
    echo -e "${YELLOW}🔗 Integration Testing:${NC}"
    echo "Update your SvelteKit frontend .env file:"
    echo "VITE_API_BASE=http://localhost:8001/api"
    echo ""
    echo -e "${GREEN}Ready for integration testing with your SvelteKit frontend!${NC}"
}

cleanup_on_error() {
    echo -e "${RED}❌ Setup failed. Cleaning up...${NC}"
    
    if [[ "$SETUP_TYPE" == "docker" ]]; then
        docker-compose -f docker-compose.staging.yml down -v 2>/dev/null || true
    fi
    
    exit 1
}

# Main execution
main() {
    echo -e "${YELLOW}Setup type: $SETUP_TYPE${NC}"
    echo ""
    
    # Set up error handling
    trap cleanup_on_error ERR
    
    # Run setup steps
    check_dependencies
    setup_environment
    
    case $SETUP_TYPE in
        "local")
            setup_local
            build_api
            # Start API in background for testing
            export $(cat .env.staging | xargs)
            ./bin/bookwork-api &
            API_PID=$!
            sleep 5
            ;;
        "docker")
            setup_docker
            ;;
        "cloud")
            echo -e "${YELLOW}For cloud deployment, please refer to the STAGING_DEPLOYMENT_GUIDE.md${NC}"
            exit 0
            ;;
    esac
    
    # Test deployment
    test_deployment
    
    # Kill background API if running locally
    if [[ "$SETUP_TYPE" == "local" && -n "$API_PID" ]]; then
        kill $API_PID 2>/dev/null || true
    fi
    
    # Show summary
    show_summary
}

# Run main function
main
