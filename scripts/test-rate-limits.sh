#!/bin/bash

# Rate Limit Testing Script
# Tests all rate limits in the nginx configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${1:-http://localhost:3000}"
API_BASE="$BASE_URL/v1"

# Test counters
PASSED=0
FAILED=0

function print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

function print_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((PASSED++))
}

function print_failure() {
    echo -e "${RED}❌ $1${NC}"
    ((FAILED++))
}

function print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

function test_rate_limit() {
    local name="$1"
    local url="$2"
    local method="${3:-GET}"
    local data="${4:-}"
    local expected_limit="$5"
    local burst="$6"

    print_header "Testing $name rate limit ($expected_limit r/s)"

    # Count successful requests before hitting limit
    local success_count=0
    local rate_limited_count=0

    # Make requests rapidly to hit the rate limit
    for i in {1..50}; do
        if [ "$method" = "POST" ]; then
            response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$url" \
                -H "Content-Type: application/json" \
                -d "$data" 2>/dev/null || echo "000")
        else
            response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
        fi

        if [ "$response" = "429" ]; then
            rate_limited_count=$((rate_limited_count + 1))
        elif [ "$response" = "000" ]; then
            print_failure "Connection failed - make sure your app is running on $BASE_URL"
            return 1
        else
            success_count=$((success_count + 1))
        fi

        # Small delay to ensure we can count the requests properly
        sleep 0.05
    done

    # Calculate expected success count (rate limit + burst)
    local expected_success=$((expected_limit + burst))

    echo "  Requests made: 50"
    echo "  Successful: $success_count"
    echo "  Rate limited: $rate_limited_count"
    echo "  Expected successful: ~$expected_success"

    # Check if we hit rate limiting (should have some 429s)
    if [ $rate_limited_count -gt 0 ]; then
        print_success "Rate limiting working for $name"
    else
        print_failure "No rate limiting detected for $name"
    fi

    # Wait for rate limit to reset
    echo "  Waiting 2 seconds for rate limit to reset..."
    sleep 2
}

function test_auth_rate_limit() {
    print_header "Testing Authentication Rate Limits"

    # Test login endpoint
    test_rate_limit "Login" "$API_BASE/auth/login" "POST" '{"username":"test","password":"test"}' 5 10

    # Test signup endpoint
    test_rate_limit "Signup" "$API_BASE/auth/signup" "POST" '{"username":"test","password":"test","email":"test@example.com"}' 5 10
}

function test_quiz_rate_limit() {
    print_header "Testing Quiz Rate Limits"

    # Test quiz question endpoint
    test_rate_limit "Quiz Question" "$API_BASE/quiz/question" "GET" "" 10 20

    # Test quiz answer endpoint
    test_rate_limit "Quiz Answer" "$API_BASE/quiz/answer" "POST" '{"question_id":1,"user_answer":"test"}' 10 20
}

function test_api_rate_limit() {
    print_header "Testing General API Rate Limits"

    # Test settings endpoint
    test_rate_limit "Settings" "$API_BASE/settings" "GET" "" 10 15

    # Test user profile endpoint
    test_rate_limit "User Profile" "$API_BASE/userz/profile" "GET" "" 10 15
}

function test_default_rate_limit() {
    print_header "Testing Default Rate Limits"

    # Test SPA routes
    test_rate_limit "SPA Route (Login)" "$BASE_URL/login" "GET" "" 10 20

    # Test static assets
    test_rate_limit "Static Assets" "$BASE_URL/assets/index.css" "GET" "" 10 30

    # Test root path
    test_rate_limit "Root Path" "$BASE_URL/" "GET" "" 10 15

    # Test 404 routes
    test_rate_limit "404 Routes" "$BASE_URL/nonexistent" "GET" "" 10 10
}

function test_concurrent_requests() {
    print_header "Testing Concurrent Requests"

    print_info "Making 10 concurrent requests to auth endpoint..."

    # Make concurrent requests
    for i in {1..10}; do
        curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE/auth/login" \
            -H "Content-Type: application/json" \
            -d '{"username":"test","password":"test"}' &
    done

    # Wait for all requests to complete
    wait

    print_success "Concurrent request test completed"
}

function show_summary() {
    print_header "Test Summary"
    echo "Passed: $PASSED"
    echo "Failed: $FAILED"
    echo "Total: $((PASSED + FAILED))"

    if [ $FAILED -eq 0 ]; then
        echo -e "\n${GREEN}🎉 All rate limit tests passed!${NC}"
    else
        echo -e "\n${RED}⚠️  Some tests failed. Check the configuration.${NC}"
    fi
}

function show_usage() {
    echo "Usage: $0 [base_url]"
    echo ""
    echo "Examples:"
    echo "  $0                          # Test http://localhost:3000 (default)"
    echo "  $0 http://localhost:3000    # Test http://localhost:3000"
    echo "  $0 http://myapp.com:3000    # Test http://myapp.com:3000"
    echo ""
    echo "Make sure your application is running before testing!"
}

# Main script
echo -e "${BLUE}🚀 Rate Limit Testing Script${NC}"
echo "Testing rate limits for: $BASE_URL"

# Check if curl is available
if ! command -v curl &>/dev/null; then
    echo -e "${RED}❌ curl is required but not installed${NC}"
    exit 1
fi

# Run all tests
print_header "Starting Rate Limit Tests"
test_auth_rate_limit
test_quiz_rate_limit
test_api_rate_limit
test_default_rate_limit
test_concurrent_requests

# Show summary
show_summary

exit $([ $FAILED -eq 0 ] && echo 0 || echo 1)
