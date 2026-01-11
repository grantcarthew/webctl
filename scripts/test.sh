#!/bin/bash
# Test script for webctl
# Runs various test configurations with clear output

set -e

# Color output helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

title() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

info() {
    echo -e "${CYAN}→ $1${NC}"
}

# Parse command line arguments
COVERAGE=false
INTEGRATION_ONLY=false
UNIT_ONLY=false
RACE=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -i|--integration)
            INTEGRATION_ONLY=true
            shift
            ;;
        -u|--unit)
            UNIT_ONLY=true
            shift
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -c, --coverage       Run with coverage analysis"
            echo "  -i, --integration    Run only integration tests"
            echo "  -u, --unit          Run only unit tests (skip integration)"
            echo "  -r, --race          Run with race detection"
            echo "  -v, --verbose       Verbose output"
            echo "  -h, --help          Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                  # Run all tests (default)"
            echo "  $0 -u               # Fast unit tests only"
            echo "  $0 -c               # Run with coverage"
            echo "  $0 -r               # Run with race detection"
            echo "  $0 -i               # Integration tests only"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Banner
clear
echo ""
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                  webctl Test Suite Runner                      ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Determine which tests to run
if [ "$INTEGRATION_ONLY" = true ]; then
    title "Running Integration Tests Only"
    info "These tests require Chrome to be available"
    echo ""

    TEST_CMD="go test"
    [ "$VERBOSE" = true ] && TEST_CMD="$TEST_CMD -v"
    TEST_CMD="$TEST_CMD -run Integration ./internal/..."

    info "Command: $TEST_CMD"
    echo ""

    if eval $TEST_CMD; then
        success "Integration tests passed"
    else
        error "Integration tests failed"
        exit 1
    fi

elif [ "$UNIT_ONLY" = true ]; then
    title "Running Unit Tests Only (Fast)"
    info "Skipping integration tests that require Chrome"
    echo ""

    TEST_CMD="go test -short"
    [ "$VERBOSE" = true ] && TEST_CMD="$TEST_CMD -v"
    [ "$RACE" = true ] && TEST_CMD="$TEST_CMD -race"
    TEST_CMD="$TEST_CMD ./internal/..."

    info "Command: $TEST_CMD"
    echo ""

    if eval $TEST_CMD; then
        success "Unit tests passed"
    else
        error "Unit tests failed"
        exit 1
    fi

elif [ "$COVERAGE" = true ]; then
    title "Running Tests with Coverage Analysis"
    echo ""

    TEST_CMD="go test -coverprofile=coverage.out"
    [ "$VERBOSE" = true ] && TEST_CMD="$TEST_CMD -v"
    [ "$RACE" = true ] && TEST_CMD="$TEST_CMD -race"
    TEST_CMD="$TEST_CMD ./internal/..."

    info "Command: $TEST_CMD"
    echo ""

    if eval $TEST_CMD; then
        success "Tests passed"
        echo ""
        title "Coverage Report"
        echo ""
        go tool cover -func=coverage.out
        echo ""
        info "Full HTML report: go tool cover -html=coverage.out"
    else
        error "Tests failed"
        exit 1
    fi

else
    # Default: Run comprehensive test suite

    # 1. Unit tests (fast)
    title "1. Unit Tests (Fast)"
    info "Running unit tests without integration tests"
    echo ""

    TEST_CMD="go test -short"
    [ "$VERBOSE" = true ] && TEST_CMD="$TEST_CMD -v"
    TEST_CMD="$TEST_CMD ./internal/..."

    info "Command: $TEST_CMD"
    echo ""

    if eval $TEST_CMD; then
        success "Unit tests passed"
    else
        error "Unit tests failed"
        exit 1
    fi

    # 2. Race detection
    title "2. Race Detection"
    info "Running unit tests with race detector"
    echo ""

    TEST_CMD="go test -short -race ./internal/..."
    info "Command: $TEST_CMD"
    echo ""

    if eval $TEST_CMD; then
        success "No race conditions detected"
    else
        error "Race conditions detected"
        exit 1
    fi

    # 3. Integration tests
    title "3. Integration Tests"
    info "Running tests that require Chrome (may be slow)"
    echo ""

    TEST_CMD="go test -run Integration ./internal/..."
    [ "$VERBOSE" = true ] && TEST_CMD="$TEST_CMD -v"
    info "Command: $TEST_CMD"
    echo ""

    if eval $TEST_CMD; then
        success "Integration tests passed"
    else
        error "Integration tests failed"
        exit 1
    fi

    # 4. Coverage summary
    title "4. Coverage Summary"
    info "Generating coverage report"
    echo ""

    go test -short -coverprofile=coverage.out ./internal/... > /dev/null 2>&1

    if [ -f coverage.out ]; then
        go tool cover -func=coverage.out | grep "total:" | awk '{print "Total Coverage: " $3}'
        echo ""
        info "For detailed coverage: go tool cover -html=coverage.out"
    fi
fi

# Final summary
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                     All Tests Passed! ✓                        ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
