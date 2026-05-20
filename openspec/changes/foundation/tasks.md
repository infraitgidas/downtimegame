# Tasks: Foundation

## Phase 1: Backend Core

- [x] 1.1 Create `backend/go.mod` with dependencies
- [x] 1.2 Create `backend/internal/hub/hub.go` — WebSocket hub
- [x] 1.3 Create `backend/internal/hub/client.go` — WebSocket client
- [x] 1.4 Write hub tests in `backend/internal/hub/hub_test.go`
- [x] 1.5 Create `backend/internal/server/server.go` — HTTP server + chi routes
- [x] 1.6 Create `backend/cmd/server/main.go` — entry point
- [x] 1.7 Verify `go build ./...` and `go test ./...` pass

## Phase 2: Docker Infrastructure

- [x] 2.1 Create `backend/Dockerfile` — multi-stage Go build
- [x] 2.2 Create `docker/nginx.conf` — reverse proxy config
- [x] 2.3 Create `docker/docker-compose.yml` — multi-service orchestration
- [x] 2.4 Create `Makefile` — dev, build, test, docker targets
- [x] 2.5 Create `.gitignore` — Go + Node ignores

## Phase 3: Dashboard Skeleton

- [x] 3.1 Create `dashboard/` with Vite + React + TypeScript scaffold
- [x] 3.2 Create `dashboard/src/App.tsx` — main dashboard layout with 4 service cards
- [x] 3.3 Create `dashboard/src/components/ServiceCard.tsx` — service status card
- [x] 3.4 Create `dashboard/src/hooks/useWebSocket.ts` — WS connection hook
- [x] 3.5 Create `dashboard/Dockerfile` — multi-stage Node build

## Phase 4: Admin Panel Skeleton

- [x] 4.1 Create `admin/` with Vite + React + TypeScript scaffold
- [x] 4.2 Create `admin/src/App.tsx` — admin page placeholder
- [x] 4.3 Create `admin/Dockerfile` — multi-stage Node build

## Phase 5: Final Verification

- [x] 5.1 Verify `make build` compiles all services
- [x] 5.2 Verify `make test` passes all Go tests
- [ ] 5.3 Verify `make dev` starts all services
