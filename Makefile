# Makefile for FinOps Bot

.PHONY: setup install build build-health test test-watch coverage integration-test e2e-test \
        run-local invoke-local clean clean-all deploy destroy emergency-stop \
        tf-init tf-plan tf-apply tf-destroy tf-outputs \
        floci-start floci-stop floci-health floci-logs \
        generate-dashboard fmt lint deps help

# ──────────────────────────────────────────────────────────────
# Environment Variables
# ──────────────────────────────────────────────────────────────

export AWS_ENDPOINT_URL ?= http://localhost:4566
export AWS_ACCESS_KEY_ID ?= test
export AWS_SECRET_ACCESS_KEY ?= test
export AWS_DEFAULT_REGION ?= us-east-1
export DRY_RUN ?= true
export S3_REPORT_BUCKET ?= finops-audit-local
export ENVIRONMENT ?= dev
export LOG_LEVEL ?= debug
export LOG_FORMAT ?= text

# ──────────────────────────────────────────────────────────────
# Version Information
# ──────────────────────────────────────────────────────────────

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# ──────────────────────────────────────────────────────────────
# Development Commands
# ──────────────────────────────────────────────────────────────

setup: ## Install all dependencies
	@echo "🔧 Setting up development environment..."
	@./scripts/setup.sh
	@echo "✅ Setup complete"

install: ## Install Go dependencies
	@echo "📦 Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies installed"

build: ## Build Lambda binary with version info
	@echo "🔨 Building Lambda binary..."
	@echo "📌 Version: $(VERSION)"
	@echo "📌 Build Time: $(BUILD_TIME)"
	@echo "📌 Git Commit: $(GIT_COMMIT)"
	@GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o bootstrap cmd/main.go
	@echo "✅ Build complete: bootstrap (version: $(VERSION))"

build-health: ## Build Health Check binary with version info
	@echo "🔨 Building Health Check binary..."
	@GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o health_bootstrap cmd/health_check.go
	@echo "✅ Build complete: health_bootstrap (version: $(VERSION))"

test: ## Run unit tests
	@echo "🧪 Running unit tests..."
	@go test ./tests/unit/... -v -cover -parallel=4 -timeout=60s

test-watch: ## Run tests in watch mode (requires gow)
	@echo "🧪 Running tests in watch mode..."
	@go install github.com/mitranim/gow@latest
	@gow -c go test ./... -v

coverage: ## Generate coverage report
	@echo "📊 Generating coverage report..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

integration-test: ## Run integration tests with Floci
	@echo "🧪 Starting Floci..."
	@make floci-start
	@sleep 2
	@echo "🧪 Running integration tests..."
	@go test -tags=integration ./tests/integration/... -v -parallel=2 -timeout=300s
	@echo "🧪 Stopping Floci..."
	@make floci-stop

e2e-test: ## Run E2E tests (requires AWS credentials)
	@echo "🧪 Running E2E tests..."
	@export RUN_E2E_TESTS=true
	@go test -tags=e2e ./tests/e2e/... -v -timeout=300s

run-local: ## Run Lambda locally with Floci
	@echo "🚀 Checking Floci status..."
	@if ! curl -s http://localhost:4566/_localstack/health > /dev/null 2>&1; then \
		echo "🚀 Starting Floci..."; \
		make floci-start; \
		sleep 2; \
	else \
		echo "✅ Floci already running"; \
	fi
	@echo "🚀 Running Lambda locally..."
	@./scripts/run_local.sh

invoke-local: ## Invoke Lambda locally
	@echo "📨 Invoking Lambda..."
	@aws lambda invoke --function-name finops-cleaner-dev \
		--payload '{"dry_run":true}' \
		--endpoint-url http://localhost:4566 \
		output.json && cat output.json

# ──────────────────────────────────────────────────────────────
# Emergency Commands
# ──────────────────────────────────────────────────────────────

emergency-stop: ## Emergency stop - disable bot and set DRY_RUN=true
	@echo "🛑 EMERGENCY STOP: Disabling bot..."
	@aws events disable-rule --name finops-daily-trigger-dev --region $(AWS_DEFAULT_REGION) || true
	@aws lambda update-function-configuration --function-name finops-cleaner-dev \
		--environment "Variables={DRY_RUN=true}" --region $(AWS_DEFAULT_REGION) || true
	@echo "✅ Bot disabled. DRY_RUN=true"

# ──────────────────────────────────────────────────────────────
# Floci Commands
# ──────────────────────────────────────────────────────────────

floci-start: ## Start Floci
	@echo "🚀 Starting Floci..."
	@if command -v floci &> /dev/null; then \
		floci start; \
	else \
		docker-compose up -d; \
	fi
	@sleep 2
	@make floci-health

floci-stop: ## Stop Floci
	@echo "🛑 Stopping Floci..."
	@if command -v floci &> /dev/null; then \
		floci stop; \
	else \
		docker-compose down; \
	fi

floci-health: ## Check Floci health
	@echo "🔍 Checking Floci health..."
	@if curl -s http://localhost:4566/_localstack/health 2>/dev/null | jq -e '.services.dynamodb == "available"' > /dev/null 2>&1; then \
		echo "✅ Floci is healthy"; \
	else \
		echo "❌ Floci is not healthy (dynamodb not available)"; \
		echo "   Try: docker-compose logs or floci logs"; \
		exit 1; \
	fi

floci-logs: ## Show Floci logs
	@if command -v floci &> /dev/null; then \
		floci logs; \
	else \
		docker-compose logs; \
	fi

# ──────────────────────────────────────────────────────────────
# Terraform Commands
# ──────────────────────────────────────────────────────────────

tf-init: ## Initialize Terraform
	@echo "🔧 Initializing Terraform..."
	@cd terraform && terraform init

tf-plan: ## Plan Terraform deployment
	@echo "📋 Planning Terraform deployment..."
	@cd terraform && terraform plan -var-file=terraform.tfvars -out=tfplan

tf-apply: ## Apply Terraform deployment
	@echo "🚀 Deploying to AWS..."
	@cd terraform && terraform apply -auto-approve tfplan

tf-destroy: ## Destroy Terraform infrastructure
	@echo "💀 Destroying infrastructure..."
	@cd terraform && terraform destroy -var-file=terraform.tfvars -auto-approve

tf-outputs: ## Show Terraform outputs
	@cd terraform && terraform output

deploy: tf-init tf-plan tf-apply ## Deploy to AWS (full flow)

destroy: tf-destroy ## Destroy infrastructure

# ──────────────────────────────────────────────────────────────
# Dashboard Commands
# ──────────────────────────────────────────────────────────────

generate-dashboard: ## Generate HTML dashboard locally
	@echo "📊 Generating dashboard..."
	@go run cmd/generate_dashboard.go -output dashboard.html
	@echo "✅ Dashboard generated: dashboard.html"

# ──────────────────────────────────────────────────────────────
# Utility Commands
# ──────────────────────────────────────────────────────────────

clean: ## Clean artifacts
	@echo "🧹 Cleaning artifacts..."
	@rm -f bootstrap health_bootstrap
	@rm -f function.zip health.zip
	@rm -f output.json
	@rm -f dashboard.html
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

clean-all: ## Clean all artifacts including build cache
	@echo "🧹 Cleaning all artifacts..."
	@make clean
	@go clean -cache -testcache -modcache
	@echo "✅ All artifacts cleaned"

fmt: ## Format code
	@go fmt ./...
	@go mod tidy

lint: ## Run linters
	@echo "🔍 Running linters..."
	@golangci-lint run ./... --timeout=5m

deps: ## Show dependencies
	@go mod graph

# ──────────────────────────────────────────────────────────────
# Version Info
# ──────────────────────────────────────────────────────────────

version: ## Show version information
	@echo "📌 Version: $(VERSION)"
	@echo "📌 Build Time: $(BUILD_TIME)"
	@echo "📌 Git Commit: $(GIT_COMMIT)"
	@echo "📌 LDFLAGS: $(LDFLAGS)"

# ──────────────────────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────────────────────

help: ## Show this help
	@echo "🔧 FinOps Bot Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "📌 Version: $(VERSION)"
	@echo ""
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort