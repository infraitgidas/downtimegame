# Backlog — Downtime Game

> Progreso general del proyecto, dividido por fases.
> ✅ Completada · 🔄 En progreso · ⏳ Pendiente · ❌ Cancelada

---

## Fase 0: Concepto y especificación ✅

- [x] Definir concepto del juego (SysAdmin/SRE gamificado con hardware real)
- [x] Documentar mecánica del juego en [`downtime-game.md`](downtime-game.md)
- [x] Documentar infraestructura de red en [`infrastructure.md`](infrastructure.md)
- [x] Documentar servicios HTTP por color en [`services.md`](services.md)
- [x] Crear SDD artifacts: spec, design, tasks

---

## Fase 1: Foundation — Backend, frontends e infraestructura ✅

### Backend Core
- [x] Crear `backend/go.mod` con dependencias (chi, gorilla/websocket, go-sqlite3, cors)
- [x] Implementar WebSocket hub (`internal/hub/hub.go`) — register, unregister, broadcast, rooms
- [x] Implementar WebSocket client (`internal/hub/client.go`) — read/write pumps
- [x] Escribir tests del hub (`internal/hub/hub_test.go`) — 3 escenarios
- [x] Crear servidor HTTP con chi router + endpoints `/health`, `/ws`, `/api/games`, `/api/services`
- [x] Graceful shutdown con SIGTERM/SIGINT
- [x] Verificar `go build ./...` y `go test ./...` pasan

### Docker & Orquestación
- [x] Crear backend Dockerfile (multi-stage Go build)
- [x] Crear nginx reverse proxy config
- [x] Crear docker-compose.yml (backend + dashboard + admin + nginx)
- [x] Crear Makefile con targets: dev, build, test, docker-up, docker-down, clean
- [x] Crear .gitignore

### Dashboard (React + Vite + TypeScript)
- [x] Scaffold con Vite + React + TypeScript
- [x] Layout con grid de 4 service cards
- [x] Componente ServiceCard con health check polling cada 5s
- [x] Hook useWebSocket para conexión WS
- [x] Indicador de estado de conexión WebSocket
- [x] Estilo monospace oscuro con glows por color

### Admin Panel (React + Vite + TypeScript)
- [x] Scaffold con Vite + React + TypeScript
- [x] Placeholder de controles de juego (iniciar/detener/escenario/estadísticas)
- [x] Hook useWebSocket para conexión WS
- [x] Visor de logs de WebSocket
- [x] Dockerfile multi-stage

### Servicios HTTP (Python) para LXC
- [x] Script `http-service.py` — sirve `/`, `/health`, `/status`
- [x] Script `deploy.sh` — deploy a CTs via Proxmox `pct`
- [x] Configuraciones de MikroTik (DNS, WiFi, firewall) en `.rsc`

### Configuración del Repositorio
- [x] Sanitizar credenciales en todos los archivos
- [x] Crear README.md con descripción del proyecto
- [x] Inicializar git, commit inicial, push a GitHub
- [x] Configurar remote: `github.com/infraitgidas/downtimegame.git`

### Verificación
- [x] `make build` compila
- [x] `make test` pasa todos los tests de Go
- [ ] `make dev` inicia todos los servicios (pendiente de verificar con LXC real)

---

## Fase 2: Escenarios de downtime + lógica de juego 🔄

> ⬇️ Arrancamos ahora mismo. Detalle abajo.

### 2.1 Modelo de datos (SQLite) ✅
- [x] Esquema: games, incidents, leaderboard con migración automática
- [x] Store en Go con `database/sql` + `modernc.org/sqlite` (sin CGo)
- [x] CRUD completo de juegos, incidentes y leaderboard

### 2.2 Sistema de escenarios ✅
- [x] Catálogo de 8 escenarios de downtime embebido en Go
- [x] Escenarios: crash, hang, latencia, DNS failure — distribuidos entre los 4 servicios
- [x] Cada escenario con: descripción, pistas, fix_hint, difficulty, time_limit

### 2.3 Motor de juego ✅
- [x] Crear partida (POST /api/games)
- [x] Iniciar partida (POST /api/games/{id}/start) — elige escenario aleatorio
- [x] Resolver incidente (POST /api/games/{id}/resolve) — calcula score
- [x] Abandonar partida (POST /api/games/{id}/abandon)
- [x] Temporizador con cuenta regresiva + timeout automático
- [x] Sistema de puntuación: base (1000) + bonus de tiempo + bonus de dificultad
- [x] Leaderboard persistente (GET /api/leaderboard)

### 2.4 Sistema de incidentes ✅
- [x] Trigger de downtime: servicio pasa a offline + incident=true
- [x] Simulación completa (sin LXC real)
- [x] Health check toggle vía estado interno del engine
- [x] Restauración automática al resolver

### 2.5 WebSocket — Eventos de juego ✅
- [x] Eventos: game_created, game_started, game_completed, game_abandoned, game_timeout
- [x] Eventos: incident_active, incident_resolved, timer_tick, service_status
- [x] Broadcast automático a todos los clientes conectados

### 2.6 Admin Panel — Controles funcionales ⏳
- [ ] Botón "Iniciar Partida" funcional (pide nombre del jugador)
- [ ] Botón "Detener Partida"
- [ ] Visualización de escenario activo
- [ ] Historial de partidas/results

### 2.7 Dashboard — Alarma y cronómetro ⏳
- [ ] Tarjeta de alarma roja cuando hay un incidente activo
- [ ] Cronómetro de resolución visible (grande, rojo, pulsante)
- [ ] Indicador de servicio afectado
- [ ] Mensaje de bienvenida al iniciar partida

---

## Fase 3: Dashboard con métricas y alarmas reales ⏳

- [ ] Historial de uptime por servicio (gráfico de línea temporal)
- [ ] Dashboard de latencia (ping times a cada LXC)
- [ ] Alertas configurables (thresholds de latencia, timeout, etc.)
- [ ] Visualización de incidentes históricos
- [ ] Modo kiosk para Monitor 1 (pantalla completa)
- [ ] Auto-reconexión WebSocket robusta

---

## Fase 4: Integración con hardware real (LXC, red) ⏳

- [ ] Script de deploy de servicios a 4 CTs LXC
- [ ] Health checks reales contra IPs de LXC
- [ ] Trigger de downtime real vía SSH a LXC (systemctl stop)
- [ ] Restauración automática de servicios al finalizar partida
- [ ] Verificación de conectividad de red MikroTik
- [ ] Modo "offline" (simulación) vs "online" (LXC real)

---

## Fase 5: Testing, pulido, deploy en feria ⏳

- [ ] Tests de integración del backend (WebSocket + HTTP)
- [ ] Tests de componentes frontend (si aplica)
- [ ] Script de deploy a PC Rocky Linux (Docker Compose + configs)
- [ ] Guía rápida de operación para voluntarios de la feria
- [ ] Checklist pre-feria (verificar red, LXC, Docker, servicios)
- [ ] Modo demo/atractor (loop automático para mostrar el juego)
- [ ] Prueba de carga: múltiples rondas seguidas

---

## Fase 6: Mejoras post-feria (nice to have) ⏳

- [ ] TUI en Bubbletea (Go) para monitor 2
- [ ] Sonidos y efectos visuales en dashboard
- [ ] Múltiples jugadores simultáneos (misma partida)
- [ ] Modo "tutorial" con guía paso a paso
- [ ] Estadísticas exportables (CSV/JSON)
- [ ] Temas visuales (dark/light/cyberpunk)

---

> **Última actualización**: 2026-05-20
> **Nota**: Todo lo marcado con 🔄 está en progreso activo.
