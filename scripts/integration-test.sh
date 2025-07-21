#!/bin/bash
set -e

echo "🧪 Bookwork API - Integration Test Suite"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8001}"
FRONTEND_BASE_URL="${FRONTEND_BASE_URL:-http://localhost:5173}"
TEST_EMAIL="test@example.com"
TEST_PASSWORD="testpass123"
ADMIN_EMAIL="admin@bookwork.com"
ADMIN_PASSWORD="admin123"

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Helper functions
log_test() {
    echo -e "${BLUE}🧪 Testing: $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((TESTS_PASSED++))
}

log_failure() {
    echo -e "${RED}❌ $1${NC}"
    FAILED_TESTS+=("$1")
    ((TESTS_FAILED++))
}

log_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# API Test Functions
test_api_health() {
    log_test "API Health Check"
    
    RESPONSE=$(curl -s -w "%{http_code}" "${API_BASE_URL}/healthz" -o /tmp/health_response.json)
    HTTP_CODE="${RESPONSE: -3}"
    
    if [[ "$HTTP_CODE" == "200" ]]; then
        HEALTH_DATA=$(cat /tmp/health_response.json)
        if echo "$HEALTH_DATA" | grep -q '"status":"ok"'; then
            log_success "API health check passed"
            return 0
        else
            log_failure "API health check returned invalid response: $HEALTH_DATA"
            return 1
        fi
    else
        log_failure "API health check failed with HTTP $HTTP_CODE"
        return 1
    fi
}

test_admin_authentication() {
    log_test "Admin Authentication"
    
    LOGIN_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\"}")
    
    if echo "$LOGIN_RESPONSE" | grep -q '"success":true'; then
        ADMIN_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4)
        if [[ -n "$ADMIN_TOKEN" ]]; then
            log_success "Admin authentication successful"
            echo "$ADMIN_TOKEN" > /tmp/admin_token.txt
            return 0
        else
            log_failure "Admin authentication failed - no token received"
            return 1
        fi
    else
        log_failure "Admin authentication failed: $LOGIN_RESPONSE"
        return 1
    fi
}

test_user_registration() {
    log_test "User Registration"
    
    # Generate unique email for test
    UNIQUE_EMAIL="test_$(date +%s)@example.com"
    
    REGISTER_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"Test User\", \"email\": \"$UNIQUE_EMAIL\", \"password\": \"$TEST_PASSWORD\"}")
    
    if echo "$REGISTER_RESPONSE" | grep -q '"success":true'; then
        USER_TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"accessToken":"[^"]*"' | cut -d'"' -f4)
        if [[ -n "$USER_TOKEN" ]]; then
            log_success "User registration successful"
            echo "$USER_TOKEN" > /tmp/user_token.txt
            echo "$UNIQUE_EMAIL" > /tmp/user_email.txt
            return 0
        else
            log_failure "User registration failed - no token received"
            return 1
        fi
    else
        log_failure "User registration failed: $REGISTER_RESPONSE"
        return 1
    fi
}

test_club_creation() {
    log_test "Club Creation"
    
    if [[ ! -f /tmp/admin_token.txt ]]; then
        log_failure "Admin token not available"
        return 1
    fi
    
    ADMIN_TOKEN=$(cat /tmp/admin_token.txt)
    
    CLUB_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/clubs" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d '{"name": "Test Book Club", "description": "A test book club for integration testing"}')
    
    if echo "$CLUB_RESPONSE" | grep -q '"success":true'; then
        CLUB_ID=$(echo "$CLUB_RESPONSE" | grep -o '"id":[0-9]*' | cut -d':' -f2)
        if [[ -n "$CLUB_ID" ]]; then
            log_success "Club creation successful (ID: $CLUB_ID)"
            echo "$CLUB_ID" > /tmp/club_id.txt
            return 0
        else
            log_failure "Club creation failed - no ID received"
            return 1
        fi
    else
        log_failure "Club creation failed: $CLUB_RESPONSE"
        return 1
    fi
}

test_club_membership() {
    log_test "Club Membership"
    
    if [[ ! -f /tmp/user_token.txt ]] || [[ ! -f /tmp/club_id.txt ]]; then
        log_failure "User token or club ID not available"
        return 1
    fi
    
    USER_TOKEN=$(cat /tmp/user_token.txt)
    CLUB_ID=$(cat /tmp/club_id.txt)
    
    MEMBER_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/clubs/${CLUB_ID}/members" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $USER_TOKEN")
    
    if echo "$MEMBER_RESPONSE" | grep -q '"success":true'; then
        log_success "Club membership successful"
        return 0
    else
        log_failure "Club membership failed: $MEMBER_RESPONSE"
        return 1
    fi
}

test_event_creation() {
    log_test "Event Creation"
    
    if [[ ! -f /tmp/admin_token.txt ]] || [[ ! -f /tmp/club_id.txt ]]; then
        log_failure "Admin token or club ID not available"
        return 1
    fi
    
    ADMIN_TOKEN=$(cat /tmp/admin_token.txt)
    CLUB_ID=$(cat /tmp/club_id.txt)
    
    # Future date for event
    FUTURE_DATE=$(date -d "+7 days" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v+7d '+%Y-%m-%dT%H:%M:%SZ')
    
    EVENT_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/clubs/${CLUB_ID}/events" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d "{\"title\": \"Test Event\", \"description\": \"Integration test event\", \"date\": \"$FUTURE_DATE\", \"location\": \"Online\"}")
    
    if echo "$EVENT_RESPONSE" | grep -q '"success":true'; then
        EVENT_ID=$(echo "$EVENT_RESPONSE" | grep -o '"id":[0-9]*' | cut -d':' -f2)
        if [[ -n "$EVENT_ID" ]]; then
            log_success "Event creation successful (ID: $EVENT_ID)"
            echo "$EVENT_ID" > /tmp/event_id.txt
            return 0
        else
            log_failure "Event creation failed - no ID received"
            return 1
        fi
    else
        log_failure "Event creation failed: $EVENT_RESPONSE"
        return 1
    fi
}

test_cors_headers() {
    log_test "CORS Headers"
    
    CORS_RESPONSE=$(curl -s -I -X OPTIONS "${API_BASE_URL}/api/auth/login" \
        -H "Origin: http://localhost:5173" \
        -H "Access-Control-Request-Method: POST" \
        -H "Access-Control-Request-Headers: Content-Type")
    
    if echo "$CORS_RESPONSE" | grep -q "Access-Control-Allow-Origin"; then
        if echo "$CORS_RESPONSE" | grep -q "Access-Control-Allow-Methods"; then
            log_success "CORS headers properly configured"
            return 0
        else
            log_failure "CORS Allow-Methods header missing"
            return 1
        fi
    else
        log_failure "CORS Allow-Origin header missing"
        return 1
    fi
}

test_frontend_connection() {
    log_test "Frontend Connection"
    
    # Check if frontend is running
    FRONTEND_RESPONSE=$(curl -s -w "%{http_code}" "$FRONTEND_BASE_URL" -o /dev/null)
    
    if [[ "$FRONTEND_RESPONSE" == "200" ]]; then
        log_success "Frontend is accessible"
        
        # Test API call from frontend context
        API_FROM_FRONTEND=$(curl -s -w "%{http_code}" \
            -H "Origin: $FRONTEND_BASE_URL" \
            "${API_BASE_URL}/healthz" -o /dev/null)
        
        if [[ "$API_FROM_FRONTEND" == "200" ]]; then
            log_success "API accessible from frontend origin"
            return 0
        else
            log_failure "API not accessible from frontend origin (HTTP $API_FROM_FRONTEND)"
            return 1
        fi
    else
        log_info "Frontend not running on $FRONTEND_BASE_URL (HTTP $FRONTEND_RESPONSE)"
        log_info "This is OK if you haven't started your SvelteKit frontend yet"
        return 0
    fi
}

# Performance Tests
test_api_performance() {
    log_test "API Performance"
    
    # Test response times for key endpoints
    HEALTH_TIME=$(curl -s -w "%{time_total}" "${API_BASE_URL}/healthz" -o /dev/null)
    
    if (( $(echo "$HEALTH_TIME < 1.0" | bc -l 2>/dev/null || echo "1") )); then
        log_success "API response time acceptable (${HEALTH_TIME}s)"
        return 0
    else
        log_failure "API response time too slow (${HEALTH_TIME}s)"
        return 1
    fi
}

# Database Connectivity Test
test_database_connectivity() {
    log_test "Database Connectivity"
    
    # Try to register a user (which requires DB write)
    TEST_DB_EMAIL="db_test_$(date +%s)@example.com"
    
    DB_TEST_RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"DB Test User\", \"email\": \"$TEST_DB_EMAIL\", \"password\": \"dbtest123\"}")
    
    if echo "$DB_TEST_RESPONSE" | grep -q '"success":true'; then
        log_success "Database connectivity confirmed"
        return 0
    else
        log_failure "Database connectivity issue: $DB_TEST_RESPONSE"
        return 1
    fi
}

# Cleanup function
cleanup_test_data() {
    log_info "Cleaning up test data..."
    
    # Delete test club if created
    if [[ -f /tmp/club_id.txt ]] && [[ -f /tmp/admin_token.txt ]]; then
        CLUB_ID=$(cat /tmp/club_id.txt)
        ADMIN_TOKEN=$(cat /tmp/admin_token.txt)
        
        curl -s -X DELETE "${API_BASE_URL}/api/clubs/${CLUB_ID}" \
            -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null
    fi
    
    # Clean up temp files
    rm -f /tmp/admin_token.txt /tmp/user_token.txt /tmp/user_email.txt
    rm -f /tmp/club_id.txt /tmp/event_id.txt /tmp/health_response.json
}

# Main test execution
run_integration_tests() {
    echo -e "${YELLOW}Starting integration tests...${NC}"
    echo ""
    
    # Core API Tests
    test_api_health
    test_database_connectivity
    test_admin_authentication
    test_user_registration
    test_club_creation
    test_club_membership
    test_event_creation
    
    # Integration Tests
    test_cors_headers
    test_frontend_connection
    test_api_performance
    
    echo ""
    echo -e "${YELLOW}🧹 Cleaning up test data...${NC}"
    cleanup_test_data
    
    # Results Summary
    echo ""
    echo "====================================="
    echo -e "${BLUE}📊 Test Results Summary${NC}"
    echo "====================================="
    echo -e "${GREEN}✅ Tests Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}❌ Tests Failed: $TESTS_FAILED${NC}"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo ""
        echo -e "${RED}Failed Tests:${NC}"
        for test in "${FAILED_TESTS[@]}"; do
            echo -e "${RED}  • $test${NC}"
        done
        echo ""
        echo -e "${YELLOW}💡 Troubleshooting Tips:${NC}"
        echo "• Check that the API is running on $API_BASE_URL"
        echo "• Verify database connection and schema"
        echo "• Check CORS configuration for frontend integration"
        echo "• Review API logs for detailed error messages"
        
        return 1
    else
        echo ""
        echo -e "${GREEN}🎉 All tests passed! Your staging environment is ready for development.${NC}"
        echo ""
        echo -e "${YELLOW}🚀 Next Steps:${NC}"
        echo "1. Start your SvelteKit frontend"
        echo "2. Update frontend .env file: VITE_API_BASE=$API_BASE_URL/api"
        echo "3. Test user flows in the frontend application"
        echo "4. Run frontend tests against this staging API"
        
        return 0
    fi
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Integration Test Suite for Bookwork API"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --api-url URL      API base URL (default: http://localhost:8001)"
        echo "  --frontend-url URL Frontend URL (default: http://localhost:5173)"
        echo "  --help, -h         Show this help"
        echo ""
        echo "Environment Variables:"
        echo "  API_BASE_URL       Override default API URL"
        echo "  FRONTEND_BASE_URL  Override default frontend URL"
        exit 0
        ;;
    --api-url)
        API_BASE_URL="$2"
        shift 2
        ;;
    --frontend-url)
        FRONTEND_BASE_URL="$2"
        shift 2
        ;;
esac

echo -e "${BLUE}Configuration:${NC}"
echo "• API Base URL: $API_BASE_URL"
echo "• Frontend URL: $FRONTEND_BASE_URL"
echo ""

# Run the tests
run_integration_tests
