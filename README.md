# Downtime Game

> Experiencia gamificada de SysAdmin/SRE para ferias universitarias.
> Diagnostica y resuelve incidentes en infraestructura real.

---

## Concepto

El **Downtime Game** es un juego presencial donde un participante asume el rol de SysAdmin/SRE.
Debe diagnosticar y resolver incidentes (downtime) que ocurren aleatoriamente sobre servicios
ejecutándose en infraestructura real (Proxmox LXC, red MikroTik, Docker).

## Stack

| Componente | Tecnología |
|------------|-----------|
| Backend | Go 1.26+ — WebSocket (gorilla/websocket) + chi router + SQLite |
| Dashboard | React + Vite + TypeScript |
| Admin Panel | React + Vite + TypeScript |
| TUI (futuro) | Bubbletea (Go) |
| Contenedores LXC | Rocky Linux 10 en Proxmox VE |
| Orquestación | Docker Compose |
| Proxy reverso | nginx |
| Infraestructura | MikroTik RB951, Proxmox VE, PC Rocky Linux |

## Servicios

| Color | IP | Puerto | Health | Status UI |
|-------|-----|--------|--------|-----------|
| 🔴 Rojo | 192.168.1.200 | 8080 | `/health` | `/` |
| 🔵 Azul | 192.168.1.204 | 8084 | `/health` | `/` |
| 🟢 Verde | 192.168.1.202 | 8082 | `/health` | `/` |
| 🟡 Amarillo | 192.168.1.203 | 8083 | `/health` | `/` |

## Arquitectura

```
Browser ──→ nginx (:80)
               ├── /api/* ──→ Go Backend (:8080)
               ├── /admin/* ──→ Admin Panel (:5174)
               └── /* ──→ Dashboard (:5173)

Backend ──WS──→ Clients (Dashboard, Admin)
Backend ──HTTP→ LXC Services (health checks / ping)
```

## Desarrollo

```bash
# Prerequisitos
go 1.26+
node 24+
docker + docker compose

# Backend
cd backend
go run ./cmd/server

# Dashboard
cd dashboard
npm install && npm run dev

# Admin Panel
cd admin
npm install && npm run dev

# Docker (todo junto)
make docker-up

# Tests
make test
```

## Proximos Pasos

- [ ] Fase 2: Escenarios de downtime + lógica de juego
- [ ] Fase 3: Dashboard con métricas y alarmas reales
- [ ] Fase 4: Integración con hardware real (LXC, red)
- [ ] Fase 5: Tests, pulido, deploy en feria

## Estructura del Proyecto

```
downtime-game/
├── backend/         ← Go module (WebSocket hub, HTTP server, API)
├── dashboard/       ← React+Vite (monitor de servicios)
├── admin/           ← React+Vite (panel de administración)
├── docker/          ← Docker Compose + nginx
├── docs/            ← Documentación
├── services/        ← Servicios HTTP Python para LXC
├── openspec/        ← SDD artifacts (specs, design, tasks)
├── Makefile
└── README.md
```

## Licencia

Proyecto interno — Feria Universitaria de Laboratorios Científicos.
