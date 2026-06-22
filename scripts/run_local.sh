#!/bin/bash
# scripts/run_local.sh - Fully Patched

set -e

# ──────────────────────────────────────────────────────────────
# Colors
# ──────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ──────────────────────────────────────────────────────────────
# Logging Functions
# ──────────────────────────────────────────────────────────────

log_info() { echo -e "${GREEN}✅${NC} $*"; }
log_warn() { echo -e "${YELLOW}⚠️${NC} $*"; }
log_error() { echo -e "${RED}❌${NC} $*"; }
log_debug() { 
    if [ "$VERBOSE" = "true" ]; then
        echo -e "${BLUE}🔍${NC} $*"
    fi
}
log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }

# ──────────────────────────────────────────────────────────────
# Environment Loading
# ──────────────────────────────────────────────────────────────

load_env() {
    local env_file="${1:-.env}"
    if [ ! -f "$env_file" ]; then
        log_warn ".env file not found. Using defaults."
        return 0
    fi
    
    # Check permissions
    if [[ "$OSTYPE" == "darwin"* ]]; then
        PERMS=$(stat -f "%Lp" "$env_file" 2>/dev/null || echo "unknown")
    else
        PERMS=$(stat -c "%a" "$env_file" 2>/dev/null || echo "unknown")
    fi
    if [ "$PERMS" != "600" ] && [ "$PERMS" != "640" ] && [ "$PERMS" != "unknown" ]; then
        log_warn ".env file has permissions $PERMS (recommended: 600)"
    fi
    
    log_info "Loading .env file"
    while IFS= read -r line || [ -n "$line" ]; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" ]] && continue
        
        # Remove leading/trailing whitespace
        line=$(echo "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
        
        # Parse key=value (handle quotes)
        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)= ]]; then
            key="${BASH_REMATCH[1]}"
            value="${line#*=}"
            
            # Remove quotes if present
            value=$(echo "$value" | sed -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//")
            
            export "$key"="$value"
            log_debug "Loaded: $key=$value"
        fi
    done < "$env_file"
}

# ──────────────────────────────────────────────────────────────
# Validation Functions
# ──────────────────────────────────────────────────────────────

validate_env() {
    local required_vars=("DRY_RUN" "REGIONS" "S3_REPORT_BUCKET" "ENVIRONMENT")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        log_error "Missing required environment variables:"
        printf "  - %s\n" "${missing_vars[@]}"
        exit 1
    fi
}

check_prerequisites() {
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        echo "   Please install Go 1.21+"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    if [[ "$(printf '%s\n' "1.21" "$GO_VERSION" | sort -V | head -n1)" != "1.21" ]]; then
        log_error "Go version must be 1.21 or higher. Found: $GO_VERSION"
        exit 1
    fi
    log_debug "Go version: $GO_VERSION"
    
    # Check if go.mod exists
    if [ ! -f "go.mod" ]; then
        log_error "go.mod not found. Run 'make setup' first."
        exit 1
    fi
}

check_floci() {
    echo ""
    log_debug "Checking Floci health..."
    if curl -s http://localhost:4566/_localstack/health > /dev/null 2>&1; then
        log_info "Floci is running"
        return 0
    else
        log_error "Floci is not running"
        echo "   Run: make floci-start"
        exit 1
    fi
}

# ──────────────────────────────────────────────────────────────
# Cleanup
# ──────────────────────────────────────────────────────────────

cleanup() {
    echo ""
    log_info "Shutting down..."
    # Add any cleanup logic here
    exit 0
}

trap cleanup SIGINT SIGTERM

# ──────────────────────────────────────────────────────────────
# Run Lambda
# ──────────────────────────────────────────────────────────────

run_lambda() {
    log_info "Building and running with go run..."
    echo ""
    go run cmd/main.go
}

# ──────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────

main() {
    echo ""
    echo "🚀 Running FinOps Bot locally..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Load environment variables
    load_env .env
    load_env .env.local 2>/dev/null || true
    
    # Set default values
    export AWS_ENDPOINT_URL=${AWS_ENDPOINT_URL:-http://localhost:4566}
    export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-test}
    export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-test}
    export AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-us-east-1}
    export DRY_RUN=${DRY_RUN:-true}
    export LOG_LEVEL=${LOG_LEVEL:-debug}
    export LOG_FORMAT=${LOG_FORMAT:-text}
    export ENVIRONMENT=${ENVIRONMENT:-dev}
    
    # Validate
    check_prerequisites
    validate_env
    check_floci
    
    # Show configuration
    echo ""
    echo "📋 Configuration:"
    echo "  DRY_RUN: $DRY_RUN"
    echo "  LOG_LEVEL: $LOG_LEVEL"
    echo "  LOG_FORMAT: $LOG_FORMAT"
    echo "  ENVIRONMENT: $ENVIRONMENT"
    echo "  AWS_ENDPOINT_URL: $AWS_ENDPOINT_URL"
    echo ""
    
    # Dry-run mode for script
    if [ "${DRY_RUN_SCRIPT:-false}" = "true" ]; then
        echo "🔍 Dry-run mode: would run with these settings"
        exit 0
    fi
    
    # Run the Lambda
    echo "🚀 Starting Lambda..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    run_lambda
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Lambda execution completed"
}

# ──────────────────────────────────────────────────────────────
# Script Entry
# ──────────────────────────────────────────────────────────────

main "$@"