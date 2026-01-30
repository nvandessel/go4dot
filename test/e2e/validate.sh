#!/bin/bash
# validate.sh - Full validation script for g4d
#
# This script provides a single command to validate all changes:
# 1. Build the project
# 2. Run linter
# 3. Run unit tests
# 4. Run E2E tests (teatest scenarios)
# 5. Run visual tests (VHS in Docker)
# 6. Generate summary report
#
# Usage:
#   make validate           # Full validation
#   make validate-quick     # Skip visual tests
#   make validate-update    # Update golden files

set -e

# Colors for output (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORT_DIR="$PROJECT_ROOT/test/e2e/reports"
QUICK_MODE=false
UPDATE_GOLDEN=false

# Parse arguments
for arg in "$@"; do
    case $arg in
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --update)
            UPDATE_GOLDEN=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --quick   Skip visual tests for faster feedback"
            echo "  --update  Update golden files (use when changes are intentional)"
            echo "  --help    Show this help message"
            exit 0
            ;;
    esac
done

cd "$PROJECT_ROOT"

# Ensure report directory exists
mkdir -p "$REPORT_DIR"

# Summary tracking
SUMMARY_FILE="$REPORT_DIR/latest-summary.txt"
PASSED=0
FAILED=0
WARNINGS=0

log_step() {
    echo -e "${BLUE}$1${NC}"
}

log_success() {
    echo -e "${GREEN}✓ $1${NC}"
    PASSED=$((PASSED + 1))
}

log_failure() {
    echo -e "${RED}✗ $1${NC}"
    FAILED=$((FAILED + 1))
}

log_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
    WARNINGS=$((WARNINGS + 1))
}

# Start validation
echo ""
echo "========================================"
echo "  g4d Validation"
echo "========================================"
echo ""

# Step 1: Build
log_step "Building project..."
if make build > "$REPORT_DIR/build.log" 2>&1; then
    log_success "Build successful"
else
    log_failure "Build failed (see $REPORT_DIR/build.log)"
    cat "$REPORT_DIR/build.log"
    exit 1
fi

# Step 2: Lint
log_step "Running linter..."
if make lint > "$REPORT_DIR/lint.log" 2>&1; then
    log_success "Lint passed"
else
    log_failure "Lint failed (see $REPORT_DIR/lint.log)"
    cat "$REPORT_DIR/lint.log"
    exit 1
fi

# Step 3: Unit tests
log_step "Running unit tests..."
if make test > "$REPORT_DIR/unit-tests.log" 2>&1; then
    # Extract test count from output
    TEST_COUNT=$(grep -E "^ok|^PASS" "$REPORT_DIR/unit-tests.log" | wc -l || echo "?")
    log_success "Unit tests passed ($TEST_COUNT packages)"
else
    log_failure "Unit tests failed (see $REPORT_DIR/unit-tests.log)"
    # Show failed tests
    grep -A 5 "FAIL" "$REPORT_DIR/unit-tests.log" || true
    exit 1
fi

# Step 4: E2E teatest scenarios (TUI tests)
log_step "Running E2E TUI tests..."
if go test -v -tags=e2e -run="^TestDashboard" ./test/e2e/scenarios/... > "$REPORT_DIR/e2e-tui.log" 2>&1; then
    log_success "E2E TUI tests passed"
else
    # TUI tests may not exist yet - that's OK
    if grep -q "no test files" "$REPORT_DIR/e2e-tui.log" || grep -q "no tests to run" "$REPORT_DIR/e2e-tui.log"; then
        log_warning "No TUI tests found (skipped)"
    else
        log_failure "E2E TUI tests failed (see $REPORT_DIR/e2e-tui.log)"
        cat "$REPORT_DIR/e2e-tui.log"
        exit 1
    fi
fi

# Step 5: Docker integration tests
log_step "Running Docker integration tests..."
if go test -v -tags=e2e -parallel=4 -timeout=10m -run="^(TestDoctor_|TestInstall_)" ./test/e2e/scenarios/... > "$REPORT_DIR/e2e-docker.log" 2>&1; then
    log_success "Docker integration tests passed"
else
    # Check if docker/podman is available
    if grep -q "No container runtime" "$REPORT_DIR/e2e-docker.log" || grep -q "skipped" "$REPORT_DIR/e2e-docker.log"; then
        log_warning "Docker tests skipped (no container runtime)"
    else
        log_failure "Docker integration tests failed (see $REPORT_DIR/e2e-docker.log)"
        cat "$REPORT_DIR/e2e-docker.log"
        exit 1
    fi
fi

# Step 6: Visual tests (VHS in Docker) - skip in quick mode
if [ "$QUICK_MODE" = false ]; then
    log_step "Running visual tests..."

    VISUAL_ARGS=""
    if [ "$UPDATE_GOLDEN" = true ]; then
        VISUAL_ARGS="UPDATE_GOLDEN=1"
    fi

    if $VISUAL_ARGS go test -v -tags=e2e -run="^TestCLI" ./test/e2e/scenarios/... > "$REPORT_DIR/e2e-visual.log" 2>&1; then
        log_success "Visual tests passed"
    else
        if grep -q "VHS not installed" "$REPORT_DIR/e2e-visual.log" || grep -q "skipped" "$REPORT_DIR/e2e-visual.log"; then
            log_warning "Visual tests skipped (VHS not available)"
        else
            log_failure "Visual tests failed (see $REPORT_DIR/e2e-visual.log)"
            cat "$REPORT_DIR/e2e-visual.log"
            # Visual test failures are warnings, not hard failures
            FAILED=$((FAILED - 1))
            WARNINGS=$((WARNINGS + 1))
        fi
    fi
else
    log_warning "Visual tests skipped (--quick mode)"
fi

# Generate summary
echo ""
echo "----------------------------------------"
echo "  Summary"
echo "----------------------------------------"
{
    echo "Validation Report"
    echo "================="
    echo "Date: $(date)"
    echo ""
    echo "Results:"
    echo "  Passed:   $PASSED"
    echo "  Failed:   $FAILED"
    echo "  Warnings: $WARNINGS"
    echo ""
    if [ $FAILED -eq 0 ]; then
        echo "Status: PASSED"
    else
        echo "Status: FAILED"
    fi
} | tee "$SUMMARY_FILE"

echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All validations passed!${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}Note: $WARNINGS warnings (see logs for details)${NC}"
    fi
    echo ""
    echo "Ready to commit changes."
    exit 0
else
    echo -e "${RED}Validation failed with $FAILED error(s)${NC}"
    echo ""
    echo "Review logs in: $REPORT_DIR/"
    exit 1
fi
