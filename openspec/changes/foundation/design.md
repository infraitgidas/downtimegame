# Design: Foundation

## Technical Approach

Backend-first with frontend scaffolds. Implement Go WebSocket hub with tests, then build Docker Compose infrastructure alongside React+Vite skeletons for Dashboard and Admin Panel.

---

## Architecture Decisions

### AD1: WebSocket Library

| Option | Tradeoff | Decision |
|--------|----------|----------|
| gorilla/websocket | Mature, widely adopted, maintained community fork | ✅ Chosen |
| nhooyr.io/websocket | Newer, context-based API | ❌ Less ecosystem support |

**Rationale**: gorilla/websocket is the de facto standard for Go WebSocket. The community fork is actively maintained.

### AD2: HTTP Router

| Option | Tradeoff | Decision |
|--------|----------|----------|
| chi | Lightweight, stdlib-compatible, middleware chaining | ✅ Chosen |
| gorilla/mux | Stable but less idiomatic middleware | ❌ Unmaintained |
| net/http (stdlib) | No routing patterns | ❌ Too basic |

### AD3: Database

| Option | Tradeoff | Decision |
|--------|----------|----------|
| SQLite via mattn/go-sqlite3 | Embedded, zero config, perfect for single node | ✅ Chosen |
| PostgreSQL | Overkill for local-only game | ❌ Too heavy |

**Note**: DB schema deferred to Fase 2. Foundation only stubs the model.

---

## Data Flow

```
Browser/Client
    │
    ├── HTTP ──→ nginx (:80)
    │               ├── /api/* ──→ backend:8080
    │               ├── /admin/* ──→ admin:5174
    │               └── /* ──→ dashboard:5173
    │
    └── WS ──→ backend:8080/ws ──→ WebSocket Hub
                                         ├── Register
                                         ├── Unregister
                                         ├── Broadcast
                                         └── JoinRoom
```

---

## Project Structure

```
downtime-game/
├── backend/
│   ├── cmd/server/main.go         ← Entry point
│   ├── internal/
│   │   ├── hub/
│   │   │   ├── hub.go             ← WebSocket hub
│   │   │   └── client.go          ← WebSocket client
│   │   └── server/
│   │       └── server.go          ← HTTP server + routes
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── dashboard/
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/ServiceCard.tsx
│   │   └── hooks/useWebSocket.ts
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── Dockerfile
├── admin/
│   ├── src/
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── Dockerfile
├── docker/
│   ├── nginx.conf
│   └── docker-compose.yml
├── Makefile
├── .gitignore
└── README.md
```

---

## Interfaces

### WebSocket Message Format

```json
{
  "type": "welcome" | "broadcast" | "service_update" | "game_event",
  "client_id": "uuid-string",
  "payload": {}
}
```

### Go Hub API

```go
type Hub struct {
    clients    map[*Client]bool
    register   chan *Client
    unregister chan *Client
    broadcast  chan []byte
    rooms      map[string]map[*Client]bool
}

func NewHub() *Hub
func (h *Hub) Run()
func (h *Hub) Broadcast(msg []byte)
func (h *Hub) JoinRoom(client *Client, room string)
```

---

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | WebSocket hub | gorilla/websocket test server + concurrent client simulation |
| Unit | HTTP handlers | net/http/httptest |
| Integration | Docker Compose | Manual verification via curl + browser |

---

## Open Questions

- [ ] None — all decisions documented
