# Infraestructura — Downtime Game

> Documento oficial de infraestructura física y lógica del proyecto.
> Versión: 1.0 — 2026-05-19

---

## 1. Topología de Red

### Diagrama lógico

```
[Internet] --(wlan0, WifiDoc)-->
                                |
                        [MikroTik RB951]
                        IP: 192.168.1.253
                        Gateway de la LAN interna
                                |
                             ether1
                                |
                       [Switch TP-Link]
                       Gigalan 16 puertos (L2 no administrable)
                        /        |            \
                       /         |             \
                      /          |              \
            [Proxmox VE]   [PC Rocky Linux]   [Laptop Admin]
            192.168.1.31    192.168.1.54       DHCP
            (LXC containers) (Docker apps)    (admin del juego)
```

### Segmentación de red

| Segmento | Red | Gateway | Descripción |
|----------|-----|---------|-------------|
| LAN Juego | 192.168.1.0/24 | 192.168.1.253 | Red interna del juego |
| WAN (Internet) | DHCP (WifiDoc) | WifiDoc | Conexión a internet por WiFi |

### Mapa de puertos físicos — MikroTik RB951

| Puerto | Conectado a | Nota |
|--------|-------------|------|
| ether1 | Switch TP-Link | Tráfico LAN interna del juego |
| ether4 | Red de laboratorio | Acceso a internet por cable (solo setup) |
| wlan0 | WifiDoc (SSID) | Acceso a internet por WiFi durante la feria |

> **Importante**: Durante la feria, ether4 NO estará conectado a la red del laboratorio.
> La conexión a internet será exclusivamente por wlan0 → WifiDoc.

---

## 2. Inventario de Hardware

### 2.1 Router — MikroTik RB951-2HnD

| Especificación | Detalle |
|----------------|---------|
| Modelo | RB951-2HnD |
| Puertos LAN | 5 (ether1-ether5) |
| WiFi | 2.4 GHz, 300 Mbps |
| IP LAN | 192.168.1.253/24 |
| Usuario | admin |
| Contraseña | `{{ MIKROTIK_PASSWORD }}` |
| Rol | Router, DHCP server, firewall, NAT |
| Servicios planeados | 4 CT LXC (rojo, azul, verde, amarillo) |
| SO invitados | Rocky Linux o Alpine Linux |

### 2.4 Puesto del Jugador — PC Rocky Linux

| Especificación | Detalle |
|----------------|---------|
| IP | 192.168.1.54 |
| Usuario | infra |
| Contraseña | `{{ PCROCKY_PASSWORD }}` |
| SO | Rocky Linux 10.1 |
| Monitores | 3 (expandible a 5) |
| Roles | Dashboard de observabilidad (Docker), TUI de juego |
| Software | Docker + Docker Compose, TUI |

### 2.5 Laptop del Admin

| Especificación | Detalle |
|----------------|---------|
| Conexión | DHCP (IP asignada por MikroTik) |
| Rol | Panel de administración del juego |
| Acceso | Browser al Admin Panel (React) |

---

## 3. Credenciales

| Recurso | IP / Host | Usuario | Contraseña | Servicio |
|---------|-----------|---------|------------|----------|
| MikroTik | 192.168.1.253 | admin | `{{ MIKROTIK_PASSWORD }}` | SSH/WinBox/WebFig |
| Proxmox | 192.168.1.31 | root | `{{ PROXMOX_PASSWORD }}` | SSH/Web UI (8006) |
| PC Rocky | 192.168.1.54 | infra | `{{ PCROCKY_PASSWORD }}` | SSH |
| WiFi WifiDoc | — | — | `{{ WIFIDOC_PASSWORD }}` | WPA2 |
| Admin Panel | dashboard.downtime.local | — | — | Web (sin auth inicial) |
| Dashboard | obs.downtime.local | — | — | Web (sin auth inicial) |

---

## 4. Arquitectura del Juego

### 4.1 Componentes de Software

```
┌─────────────────────────────────────────────────────────┐
│                    PC Rocky Linux (192.168.1.54)          │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │   Dashboard   │  │      TUI     │  │  Admin Panel    │  │
│  │   (React)     │  │  (Bubbletea) │  │  (React)        │  │
│  │   Monitor 1   │  │  Monitor 2   │  │  Laptop Admin   │  │
│  └──────┬───────┘  └──────┬───────┘  └───────┬─────────┘  │
│         │                 │                   │             │
│         └────────────┬────┘───────────────────┘             │
│                      │  WebSocket                           │
│              ┌───────┴────────┐                             │
│              │    Backend      │                             │
│              │   (Go + WS)    │                             │
│              │   Docker       │                             │
│              └───────┬────────┘                             │
│                      │                                      │
└──────────────────────┼──────────────────────────────────────┘
                       │ HTTP health checks / ping
                       │
┌──────────────────────┼──────────────────────────────────────┐
│              ┌───────┴────────┐                             │
│              │   Proxmox VE   │  (192.168.1.31)              │
│              │                │                              │
│  ┌───────────┴───────────┐   │                              │
│  │  LXC Rojo  │  LXC Azul│   │                              │
│  │  LXC Verde │  LXC Amar│   │                              │
│  └───────────────────────┘   │                              │
└──────────────────────────────┘                              │
                                                              │
┌─────────────────────────────────────────────────────────────┐
│                    MikroTik RB951 (192.168.1.253)            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐ │
│  │ DHCP     │ │ Firewall │ │   NAT    │ │  DNS forwarder  │ │
│  └──────────┘ └──────────┘ └──────────┘ └────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Servicios por Color (LXC en Proxmox)

| Servicio | CT LXC | VMID | SO | IP | Puerto HTTP | Endpoints | Estado |
|----------|--------|------|----|-----|-------------|-----------|--------|
| 🔴 Rojo | sg-rojo | 200 | Rocky Linux 10 | 192.168.1.200 | 8080 | `/` (UI), `/health`, `/status` | ✅ Activo |
| 🔵 Azul | sg-azul | 201 | Rocky Linux 10 | 192.168.1.204 | 8084 | `/` (UI), `/health`, `/status` | ✅ Activo |
| 🟢 Verde | sg-verde | 202 | Rocky Linux 10 | 192.168.1.202 | 8082 | `/` (UI), `/health`, `/status` | ✅ Activo |
| 🟡 Amarillo | sg-amarillo | 203 | Rocky Linux 10 | 192.168.1.203 | 8083 | `/` (UI), `/health`, `/status` | ✅ Activo |

Cada servicio sirve:
- **`/`**: Página HTML con UI/UX atractiva (fondo animado con gradiente, grid, scanlines, glow orbs, uptime en vivo)
- **`/health`**: JSON con status, service, color, timestamp, uptime (para el backend del juego)
- **`/status`**: JSON detallado con color_hex, port, started_at (para el dashboard)

El backend del juego consulta `/health` periódicamente para detectar downtimes.

**Nota**: El template de Rocky Linux 10 en Proxmox puede no asignar la IP estática automáticamente
en algunos casos. El script `services/deploy.sh` maneja el deploy completo del servicio HTTP.

### 4.3 Flujo del Juego

```
1. Admin inicia sesión de juego desde el Admin Panel
   ↓
2. Jugador ingresa su nombre en la TUI (PC Rocky, monitor central)
   ↓
3. El backend elige un escenario aleatorio
   ↓
4. Se genera un downtime en uno de los 4 servicios LXC
   - Ej: "Corte de red en servicio Rojo" → iptables drop en lxc-red
   - Ej: "Servicio caído" → kill del proceso HTTP en lxc-azul
   - Ej: "DNS roto" → /etc/resolv.conf inválido en lxc-verde
   ↓
5. El dashboard (monitor 1) muestra:
   - Alarma visual + sonora escandalosa
   - Cronómetro grande en ROJO
   - Qué servicio cayó
   ↓
6. El jugador diagnostica y resuelve:
   - Usa la terminal (monitor 2 o 3) para SSH a Proxmox/LXC
   - Identifica la causa raíz
   - Aplica la solución
   ↓
7. El backend detecta la recuperación del health check
   ↓
8. Se guarda: nombre del jugador + tiempo de resolución
   ↓
9. Dashboard muestra pantalla de éxito + tiempo
   ↓
10. Ready para siguiente jugador
```

### 4.4 Stack Tecnológico (Planificado)

| Componente | Tecnología | Estado |
|------------|-----------|--------|
| Backend | Go 1.26+ — WebSocket (gorilla/websocket) + chi router | 📝 Planeado |
| TUI (PC Rocky) | Bubbletea (Go) — monitor 2 | 📝 Planeado |
| Dashboard (PC Rocky) | React + Vite + TypeScript — monitor 1 | 📝 Planeado |
| Admin Panel | React + Vite + TypeScript — laptop admin | 📝 Planeado |
| Contenedores LXC | Alpine Linux o Rocky Linux en Proxmox | 📝 Planeado |
| Orquestación | Docker Compose (backend, dashboard, admin) | 📝 Planeado |
| Proxy reverso | nginx (dentro de Docker Compose) | 📝 Planeado |
| Base de datos | SQLite (backend Go, embedded) | 📝 Planeado |

---

## 5. Limitaciones y Supuestos

- ✅ **100% local**: Todo el stack se ejecuta en el hardware del laboratorio
- ✅ **Sin internet**: El juego debe funcionar sin conectividad externa
- ✅ **Multi-nivel**: Jugadores con experiencia nula o avanzada deben poder jugar
- ❌ **Sin cloud**: No se depende de servicios externos, APIs, ni SaaS
- ⚠️ **Sin auth inicial**: El MVP no requiere autenticación (entorno controlado de feria)

---

## 6. Estado del Proyecto

### 6.1 Línea de tiempo

| Fase | Descripción | Estado |
|------|-------------|--------|
| **Fase 0** | Concepto y especificación | ✅ COMPLETADA |
| **Fase 1** | Foundation — estructura, backend, scaffolds | 📝 SDD Propuesta creada |
| **Fase 2** | Escenarios de downtime + lógica de juego | ⏳ Pendiente |
| **Fase 3** | Dashboard con métricas y alarmas | ⏳ Pendiente |
| **Fase 4** | Integración con hardware real (LXC, red) | ⏳ Pendiente |
| **Fase 5** | Tests, pulido, deploy en feria | ⏳ Pendiente |

### 6.2 Lo que existe hoy

| Artefacto | Ruta |
|-----------|------|
| Especificación del juego | `docs/downtime-game.md` |
| Documento de infraestructura | `docs/infrastructure.md` ← este archivo |
| Documento de servicios HTTP | `docs/services.md` |
| Servicio HTTP Python (código fuente) | `services/http-service.py` |
| Script de deploy a CTs | `services/deploy.sh` |
| Configuración MikroTik (export format) | `mikrotik-feria-config.rsc` |
| Registros DNS estáticos | `mikrotik-dns.rsc` |
| SDD Proposal — Foundation | Engram memory (sdd/foundation/proposal) |
| Skill registry (AI) | `.atl/skill-registry.md` |
| **Documentación técnica de servicios** | `docs/services.md` §§4-6 (nuevas secciones) |
| **4 CTs LXC con http-service.py actualizado** | ✅ Los 4 responden `/health`, `/status`, UI completa |

### 6.3 Lo que NO existe (próximos pasos)

- ❌ Código fuente (backend Go, frontends, TUI)
- ❌ Dockerfiles ni Docker Compose
- ❌ Configuración de red en MikroTik
- ❌ Tests automatizados

---

## 7. Puertos y Servicios (Mapeo)

| Servicio | Puerto Host | Protocolo | Descripción |
|----------|-------------|-----------|-------------|
| Backend API/WS | 8080 | TCP | WebSocket + REST API |
| Dashboard | 5173 | TCP | React/Vite dev server |
| Admin Panel | 5174 | TCP | React/Vite dev server |
| nginx (prod) | 80 | TCP | Reverse proxy unificado |
| Proxmox WEB | 8006 | TCP | Interfaz web Proxmox |
| Proxmox SSH | 22 | TCP | Acceso a hypervisor |
| LXC Rojo | 80 | TCP | Servicio web |
| LXC Azul | 8080 | TCP | API secundaria |
| LXC Verde | 3000 | TCP | App interna |
| LXC Amarillo | 9090 | TCP | Servicio monitoreo |
| MikroTik SSH | 22 | TCP | Administración |
| MikroTik WinBox | 8291 | TCP | Administración GUI |

---

## 8. Notas de Configuración

### MikroTik — Configuración crítica

- **Gateway LAN**: 192.168.1.253/24
- **DHCP Server**: 192.168.1.100-192.168.1.200 (para laptops admin y dispositivos)
- **NAT**: Masquerade desde ether1 hacia wlan0 (salida a internet)
- **DNS**: Forwarder a 8.8.8.8 (o cache-only si no hay internet)
- **Firewall**: Permitir tráfico interno, bloquear WAN innecesario

### Proxmox — Configuración crítica

- **Bridge**: vmbr0 en 192.168.1.31/24
- **Gateway**: 192.168.1.253
- **CTs**: 4 contenedores LXC con IPs estáticas 192.168.1.100-103
- **Template**: Alpine Linux 3.21+ o Rocky Linux 10.1

### PC Rocky Linux — Configuración crítica

- **Docker**: Docker Engine + Docker Compose plugin
- **Monitores**: 3 displays conectados a la misma GPU
- **Dashboard**: Monitor 1 (izquierdo) — pantalla completa en Chromium kiosk mode
- **TUI**: Monitor 2 (central) — terminal a pantalla completa
- **Terminal**: Monitor 3 (derecho) — SSH a Proxmox/LXC para debugging

---

> Este documento se actualiza a medida que avanza el proyecto.
> Próxima revisión: al completar Fase 1 (Foundation).
