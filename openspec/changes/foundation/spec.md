# Spec: Foundation

## Purpose

Establish the technical foundation of the Downtime Game project: Go backend with WebSocket hub, Docker Compose multi-service orchestration, and skeletons for Dashboard and Admin Panel frontends.

---

## Requirements

### R1: Go Module with WebSocket Hub

The system MUST provide a Go module (`backend/`) with a WebSocket hub that supports:

- Client registration and unregistration
- Broadcast messages to all connected clients
- Room-based messaging per game session
- Graceful shutdown on SIGTERM/SIGINT

#### Scenario: Client connects and receives welcome message

- GIVEN a WebSocket hub is running
- WHEN a client connects via WebSocket
- THEN the client receives a `{"type": "welcome", "client_id": "..."}` message

#### Scenario: Broadcast sends message to all clients

- GIVEN 3 clients are connected to the hub
- WHEN a broadcast message is sent
- THEN all 3 clients receive the same message

#### Scenario: Client disconnect removes from hub

- GIVEN a client is connected to the hub
- WHEN the client disconnects
- THEN the hub unregisters the client
- AND remaining clients are unaffected

### R2: REST API Endpoints

The backend MUST serve REST API endpoints via chi router on port 8080:

- `GET /health` — returns `{"status": "ok"}`
- `GET /api/games` — returns empty list of games
- `GET /api/services` — returns service configurations
- `GET /ws` — WebSocket upgrade endpoint

#### Scenario: Health check returns OK

- GIVEN the server is running
- WHEN a GET request is sent to `/health`
- THEN the response status is 200
- AND the body is `{"status": "ok"}`

#### Scenario: WebSocket upgrade works

- GIVEN the server is running
- WHEN a WebSocket upgrade request is sent to `/ws`
- THEN the connection upgrades successfully to WebSocket

### R3: Dashboard Skeleton

The system MUST provide a React+Vite+TypeScript dashboard (`dashboard/`) that:

- Displays a title "Downtime Game — Dashboard"
- Shows a grid of 4 service placeholders (rojo, azul, verde, amarillo)
- Connects to the backend WebSocket on page load
- Renders without external CDN dependencies

#### Scenario: Dashboard loads and shows services

- GIVEN the dashboard is served
- WHEN a browser loads the page
- THEN it shows "Downtime Game — Dashboard" as the title
- AND a grid of 4 service cards is rendered

### R4: Admin Panel Skeleton

The system MUST provide a React+Vite+TypeScript admin panel (`admin/`) that:

- Displays a title "Downtime Game — Admin"
- Shows placeholder content for game control
- Connects to the backend WebSocket on page load
- Renders without external CDN dependencies

#### Scenario: Admin panel loads

- GIVEN the admin panel is served
- WHEN a browser loads the page
- THEN it shows "Downtime Game — Admin" as the title

### R5: Docker Compose Multi-Service

The system MUST provide Docker Compose orchestration (`docker-compose.yml`) that:

- Runs the Go backend on port 8080
- Runs the Dashboard on port 5173
- Runs the Admin Panel on port 5174
- Uses nginx reverse proxy on port 80
- All services communicate via a shared Docker network

#### Scenario: All services start

- GIVEN Docker Compose is running
- WHEN `docker compose up` completes
- THEN the backend responds on port 8080
- AND the dashboard responds on port 5173
- AND nginx responds on port 80

### R6: Makefile

The project MUST provide a Makefile with targets:

- `make dev` — starts all services in development mode
- `make build` — builds all production artifacts
- `make test` — runs Go tests
- `make docker-up` — starts Docker Compose
- `make docker-down` — stops Docker Compose

#### Scenario: Make test runs Go tests

- GIVEN the project is set up with Go
- WHEN `make test` is executed
- THEN Go tests run and pass
