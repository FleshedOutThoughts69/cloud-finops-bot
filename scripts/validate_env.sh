#!/bin/bash
# scripts/validate_env.sh

echo "🔍 Validating environment variables..."

required_vars=(
    "DRY_RUN"
    "REGIONS"
    "S3_REPORT_BUCKET"
    "ENVIRONMENT"
)

missing_vars=()
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -ne 0 ]; then
    echo "❌ Missing required environment variables:"
    printf "  - %s\n" "${missing_vars[@]}"
    exit 1
fi

echo "✅ All required environment variables are set"