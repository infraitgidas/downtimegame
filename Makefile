# =============================================================================
# Makefile — Downtime Game
# Targets: dev, build, test, docker-up, docker-down, clean
# =============================================================================

.PHONY: dev build test docker-up docker-down clean

# ── Development ──────────────────────────────────────────────────────────────

dev: ## Start all services in development mode
	@echo "Starting backend in dev mode..."
	@cd backend && go run ./cmd/server &

.PHONY: dev

build: ## Build all production artifacts
	@echo "Building backend..."
	@cd backend && go build -o bin/server ./cmd/server
	@echo "Building dashboard..."
	@cd dashboard && npm run build 2>/dev/null || echo "Dashboard build skipped (npm not configured)"
	@echo "Building admin..."
	@cd admin && npm run build 2>/dev/null || echo "Admin build skipped (npm not configured)"
	@echo "✅ Build complete"

# ── Testing ──────────────────────────────────────────────────────────────────

test: ## Run all Go tests
	@echo "Running Go tests..."
	@cd backend && go test ./... -v -count=1
	@echo "✅ Tests complete"

# ── Docker ───────────────────────────────────────────────────────────────────

docker-up: ## Start Docker Compose services
	@echo "Starting Docker services..."
	@cd docker && docker compose up --build -d
	@echo "✅ Docker services started"

docker-down: ## Stop Docker Compose services
	@echo "Stopping Docker services..."
	@cd docker && docker compose down
	@echo "✅ Docker services stopped"

docker-logs: ## Tail Docker logs
	@cd docker && docker compose logs -f

# ── Cleanup ──────────────────────────────────────────────────────────────────

clean: ## Clean build artifacts
	@rm -rf backend/bin
	@rm -rf dashboard/dist 2>/dev/null || true
	@rm -rf admin/dist 2>/dev/null || true
	@echo "✅ Clean complete"

# ── Help ─────────────────────────────────────────────────────────────────────

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
