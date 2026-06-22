#!/bin/bash
# scripts/setup.sh - Fully Patched

set -e

echo "🔧 Setting up FinOps Bot development environment..."

# ──────────────────────────────────────────────────────────────
# Check Go Version
# ──────────────────────────────────────────────────────────────

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ and try again."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [[ "$(printf '%s\n' "1.21" "$GO_VERSION" | sort -V | head -n1)" != "1.21" ]]; then
    echo "❌ Go version must be 1.21 or higher. Found: $GO_VERSION"
    exit 1
fi
echo "✅ Go $GO_VERSION detected"

# ──────────────────────────────────────────────────────────────
# Check Docker
# ──────────────────────────────────────────────────────────────

if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker and try again."
    exit 1
fi
echo "✅ Docker detected"

# ──────────────────────────────────────────────────────────────
# Check/Install Floci
# ──────────────────────────────────────────────────────────────

if ! command -v floci &> /dev/null; then
    echo "📦 Installing Floci..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install floci-io/floci/floci
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        curl -sSL https://floci.io/install.sh | sh
    else
        echo "⚠️ Unsupported OS. Please install Floci manually."
        echo "  Visit: https://floci.io/aws/"
        exit 1
    fi
else
    echo "✅ Floci detected"
fi

# ──────────────────────────────────────────────────────────────
# Check/Install Terraform
# ──────────────────────────────────────────────────────────────

if command -v terraform &> /dev/null; then
    TERRAFORM_VERSION=$(terraform version -json | jq -r '.terraform_version' 2>/dev/null || echo "unknown")
    echo "✅ Terraform $TERRAFORM_VERSION detected"
else
    echo "⚠️ Terraform not found. Please install Terraform 1.5+ for deployment."
    echo "  Visit: https://developer.hashicorp.com/terraform/downloads"
fi

# ──────────────────────────────────────────────────────────────
# Install golangci-lint
# ──────────────────────────────────────────────────────────────

if ! command -v golangci-lint &> /dev/null; then
    echo "📦 Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
else
    echo "✅ golangci-lint detected"
fi

# ──────────────────────────────────────────────────────────────
# Install pre-commit
# ──────────────────────────────────────────────────────────────

if ! command -v pre-commit &> /dev/null; then
    echo "📦 Installing pre-commit..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install pre-commit
    else
        pip install pre-commit
    fi
else
    echo "✅ pre-commit detected"
fi

# ──────────────────────────────────────────────────────────────
# Install Go Dependencies
# ──────────────────────────────────────────────────────────────

echo "📦 Installing Go dependencies..."
go mod download
go mod verify
go mod tidy

# ──────────────────────────────────────────────────────────────
# Create .env from .env.example
# ──────────────────────────────────────────────────────────────

if [ ! -f .env ]; then
    echo "📝 Creating .env from .env.example..."
    cp .env.example .env
    echo "✅ Created .env. Please review and update it."
else
    echo "✅ .env already exists"
fi

# ──────────────────────────────────────────────────────────────
# Setup pre-commit hooks
# ──────────────────────────────────────────────────────────────

if [ -f .pre-commit-config.yaml ]; then
    echo "🔧 Installing pre-commit hooks..."
    pre-commit install
    pre-commit install --hook-type commit-msg
    echo "✅ Pre-commit hooks installed"
fi

# ──────────────────────────────────────────────────────────────
# Create docs directory
# ──────────────────────────────────────────────────────────────

if [ ! -d docs ]; then
    echo "📁 Creating docs directory..."
    mkdir -p docs
fi

# ──────────────────────────────────────────────────────────────
# Final Output
# ──────────────────────────────────────────────────────────────

echo ""
echo "✅ Setup complete!"
echo ""
echo "📋 Next steps:"
echo "  1. Review and update .env with your configuration"
echo "  2. Run 'make build' to build the Lambda binary"
echo "  3. Run 'make floci-start' to start Floci"
echo "  4. Run 'make floci-health' to verify Floci is healthy"
echo "  5. Run 'make test' to run unit tests"
echo ""
echo "🚀 Ready to start coding!"