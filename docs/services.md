# Servicios HTTP por Color — Downtime Game

> Servicios web que se ejecutan en los 4 contenedores LXC de Proxmox.
> Cada uno sirve una página de estado con su color y un endpoint `/health` para monitoreo.

---

## 1. Stack

| Componente | Tecnología |
|------------|-----------|
| Lenguaje | Python 3 |
| Servidor HTTP | `http.server` (stdlib — sin dependencias externas) |
| Init | systemd (servicio `sg-<color>`) |
| Endpoints | `/` (UI), `/health` (JSON), `/status` (JSON detallado) |
| Puerto | 8080 (rojo), 8084 (azul), 8082 (verde), 8083 (amarillo) |
| Sin internet | La UI no depende de CDNs ni Google Fonts |

---

## 2. Ubicación de Archivos

### En cada CT

| Archivo | Ruta |
|---------|------|
| Script Python | `/opt/http-service.py` |
| Systemd unit | `/etc/systemd/system/sg-<color>.service` |

### En este repositorio

| Archivo | Descripción |
|---------|-------------|
| `services/http-service.py` | Script del servidor HTTP (multi-color vía env vars) |
| `services/deploy.sh` | Script de deploy a CTs via Proxmox (`pct`) |
| `mikrotik-dns.rsc` | Registros DNS estáticos para MikroTik |

---

## 3. Endpoints

### `GET /` — Página de estado

HTML con:
- Nombre y emoji del color
- Indicador ONLINE con animación pulsante
- Grid con uptime, puerto, timestamp, estado
- Scanlines + grid background con glow orbs
- Auto-actualización via JS cada 2 segundos
- Sin dependencias externas (font-family nativa monospace)

### `GET /health` — Health check simple

```json
{
  "status": "ok",
  "service": "sg-rojo",
  "color": "rojo",
  "timestamp": "2026-05-20T02:36:31.839644",
  "uptime": "00:15:22",
  "uptime_seconds": 922
}
```

### `GET /status` — Health check detallado

```json
{
  "status": "ok",
  "service": "sg-rojo",
  "color": "rojo",
  "color_hex": "#DC143C",
  "port": 8080,
  "started_at": "2026-05-20T02:21:09+00:00",
  "uptime": "00:15:22",
  "uptime_seconds": 922,
  "healthy": true
}
```

---

## 4. Descripción Técnica

### 4.1 Arquitectura del Script

El servicio es un servidor HTTP implementado en Python puro usando la biblioteca estándar `http.server`. No tiene dependencias externas — ni pip, ni requirements.txt, nada.

```
http.server.HTTPServer (0.0.0.0:{PORT})
  └── BaseHTTPRequestHandler.do_GET()
        ├── /        → HTML_PAGE (f-string con config del color)
        ├── /health  → JSON {status, service, color, timestamp, uptime}
        └── /status  → JSON detallado {color_hex, port, started_at, healthy}
```

### 4.2 Sistema Multi-Color

Un solo script sirve a los 4 colores. La configuración se inyecta vía variables de entorno:

| Variable | Función |
|----------|---------|
| `SERVICE_COLOR` | Clave en `COLOR_CONFIG` — define nombre, emoji, paleta CSS |
| `SERVICE_PORT` | Puerto TCP donde escucha |
| `SERVICE_ID` | Identificador del servicio (`sg-rojo`, `sg-azul`, etc.) |

El diccionario `COLOR_CONFIG` mapea cada color a:
- `css`: color principal (hex)
- `css_bg`, `css_grad_from`, `css_grad_to`: degradado de fondo
- `name`: nombre capitalizado
- `emoji`: emoji del color

### 4.3 Manejo de Señales

El script captura `SIGTERM` e `SIGINT` para hacer shutdown graceful del server HTTP. El systemd unit usa `SIGTERM` para detener el servicio, y `Restart=always` lo revive automáticamente si falla.

```
SIGTERM/SIGINT → server.shutdown() → exit(0)
Proceso muerto → systemd (RestartSec=5) → restart automático
```

### 4.4 Endpoints

#### `GET /` — Página de estado
HTML renderizado con CSS inline, sin dependencias externas. Incluye JS que hace `fetch('/health')` cada 2 segundos para actualizar uptime y timestamp en vivo.

#### `GET /health` — Health check simple
```json
{
  "status": "ok",
  "service": "sg-rojo",
  "color": "rojo",
  "timestamp": "2026-05-20T02:36:31.839644",
  "uptime": "00:15:22",
  "uptime_seconds": 922
}
```

#### `GET /status` — Health check detallado
```json
{
  "status": "ok",
  "service": "sg-rojo",
  "color": "rojo",
  "color_hex": "#DC143C",
  "port": 8080,
  "started_at": "2026-05-20T02:21:09+00:00",
  "uptime": "00:15:22",
  "uptime_seconds": 922,
  "healthy": true
}
```

### 4.5 Logs

Salida por stderr con formato:
```
[H:M:S] METHOD PATH STATUS
```
Ejemplo:
```
[14:23:01] GET /health 200
[14:23:01] GET / 200
```

Los logs se capturan via journald gracias al `StandardOutput=journal` y `StandardError=journal` del systemd unit.

---

## 5. Descripción de Diseño

### 5.1 Estética General

El diseño de los servicios usa una estética **cyberpunk/sysadmin** con:
- **Scanlines CRT**: overlays de líneas horizontales semitransparentes que simulan un monitor viejo
- **Grid background**: rejilla tipo "matrix" con líneas sutiles de 40px
- **Glow orbs**: esferas de luz desenfocada que flotan lentamente animadas con CSS
- **Monospace**: tipografía nativa `Courier New / Liberation Mono` sin dependencias externas

### 5.2 Paleta de Colores

Cada servicio tiene su propia identidad visual:

| Color | Hex | Gradiente (from → to) | Glow |
|-------|-----|----------------------|------|
| 🔴 Rojo | `#DC143C` | `#2d0a14 → #1a0a0f` | Rojo intenso |
| 🔵 Azul | `#1E90FF` | `#0a142d → #0a0f1a` | Azul dodger |
| 🟢 Verde | `#00FF7F` | `#0a2d14 → #0a1a0f` | Verde primavera |
| 🟡 Amarillo | `#FFD700` | `#2d2d0a → #1a1a0a` | Dorado |

Fondo general: degradado oscuro + negro sólido en la esquina inferior derecha.

### 5.3 Animaciones CSS

| Animación | Elemento | Comportamiento |
|-----------|----------|----------------|
| `pulse` | Emoji del color | Escala 1 → 1.05 en 2s, loop infinito |
| `dotPulse` | Punto ONLINE | Opacidad 1 → 0.5, escala 1 → 0.85 en 1.5s |
| `orbFloat` | Glow orbs | Translate + scale, alternando dirección 8-10s |

### 5.4 Layout

```
┌─────────────────────────────────────────┐
│  (grid background + scanlines overlay)   │
│  ┌─────────────────────────────────────┐ │
│  │          🔴 (emoji pulse)           │ │
│  │          ROJO (glow text)           │ │
│  │          sg-rojo                    │ │
│  │                                     │ │
│  │    ● ONLINE                         │ │
│  │                                     │ │
│  │  ┌──────────┬──────────┐            │ │
│  │  │ Uptime   │ Puerto   │            │ │
│  │  │ 10:23:45 │ 8080     │            │ │
│  │  ├──────────┼──────────┤            │ │
│  │  │Timestamp │ Estado   │            │ │
│  │  │ 11:30    │ Saludable│            │ │
│  │  └──────────┴──────────┘            │ │
│  │                                     │ │
│  │  [/]  [/health]  [/status]          │ │
│  └─────────────────────────────────────┘ │
│  (glow orb 1)      (glow orb 2)          │
└─────────────────────────────────────────┘
```

### 5.5 Responsive

A partir de 600px de ancho, el layout se adapta:
- Tamaño de fuente reducido (service-name de 4.5rem → 2.5rem)
- Grid de 2 columnas → 1 columna
- Emoji más chico (4rem → 2.5rem)

### 5.6 Principios de Diseño

- **Zero dependencias externas**: no se carga nada de CDNs, Google Fonts, ni librerías externas
- **Funciona sin internet**: todo el CSS y JS es inline en el HTML
- **Auto-actualización**: el frontend se actualiza solo via JS polling cada 2s
- **Identidad por color**: cada servicio se distingue al instante por su paleta cromática

---

## 6. Descripción Operativa

### 6.1 Ciclo de Vida del Servicio

```
systemctl start sg-rojo
  → systemd ejecuta /usr/bin/python3 /opt/http-service.py
  → HTTPServer bindea en 0.0.0.0:{PORT}
  → Servicio responde requests HTTP

systemctl stop sg-rojo
  → systemd envía SIGTERM al proceso
  → El script ejecuta server.shutdown() y sale limpio
  → systemd espera que el proceso termine

Falla inesperada
  → El proceso muere (exit code != 0)
  → systemd detecta la salida
  → Espera 5 segundos (RestartSec=5)
  → Reinicia automáticamente (Restart=always)
```

### 6.2 Monitoreo

El backend del juego (Go) consulta periódicamente `/health` de cada servicio para detectar downtimes. El dashboard puede consultar `/status` para obtener información más detallada.

| Endpoint | Propósito | Consumidor |
|----------|-----------|------------|
| `/health` | Detectar si el servicio está vivo | Backend Go del juego |
| `/status` | Obtener métricas detalladas | Dashboard de observabilidad |
| `/` | UI visual para humanos | Navegador del operador |

### 6.3 Deploy

El deploy se realiza **desde Proxmox VE** usando `pct` (Proxmox Container Toolkit).

**Secuencia correcta de update** (aprendida de la práctica):

```bash
# 1. Detener y deshabilitar el servicio VIEJO
pct exec {VMID} -- systemctl stop sg-{color}
pct exec {VMID} -- systemctl disable sg-{color}
pct exec {VMID} -- systemctl reset-failed sg-{color}

# 2. Matar cualquier proceso que aún tenga el puerto
pct exec {VMID} -- fuser -k {PORT}/tcp
sleep 2

# 3. Limpiar archivos viejos (pueden estar en rutas alternativas)
pct exec {VMID} -- rm -rf /opt/downtime-game 2>/dev/null
pct exec {VMID} -- rm -f /etc/systemd/system/sg-{color}.service
pct exec {VMID} -- systemctl daemon-reload

# 4. Copiar el nuevo script y unit file
pct push {VMID} http-service.py /opt/http-service.py --perms 755
pct push {VMID} sg-{color}.service /etc/systemd/system/sg-{color}.service --perms 644

# 5. Iniciar el nuevo servicio
pct exec {VMID} -- systemctl daemon-reload
pct exec {VMID} -- systemctl enable sg-{color}
pct exec {VMID} -- systemctl start sg-{color}

# 6. Verificar
sleep 2
pct exec {VMID} -- systemctl is-active sg-{color}
pct exec {VMID} -- curl -s http://127.0.0.1:{PORT}/health
```

> **⚠️ Gotcha importante**: Si el CT tenía una versión anterior del servicio en `/opt/downtime-game/server.py`, el systemd unit VIEJO revive el proceso al instante después de `fuser -k` gracias a `Restart=always`. Siempre hacer `systemctl stop` + `systemctl disable` ANTES de matar el proceso.

### 6.4 Troubleshooting

#### Servicio no arranca
```bash
# Dentro del CT
systemctl status sg-rojo --no-pager
journalctl -u sg-rojo -n 50 --no-pager

# Verificar que Python 3 existe
which python3

# Probar el script manualmente
SERVICE_COLOR=rojo SERVICE_PORT=8080 python3 /opt/http-service.py
```

#### Puerto no responde
```bash
# Verificar que el servicio está escuchando
ss -tlnp | grep 8080

# Verificar que NO hay un proceso zombie en ruta alternativa
ps aux | grep python

# Verificar firewall dentro del CT
iptables -L -n
```

#### CT no reachable
```bash
# Desde Proxmox
pct enter <vmid>
ip addr show
ip route show
ping 192.168.1.253
```

#### El proceso se restaura solo después de matarlo
```bash
# Secuencia correcta (no al revés):
systemctl stop sg-rojo          # 1. Parar systemd
systemctl disable sg-rojo       # 2. Deshabilitar
fuser -k 8080/tcp               # 3. Matar proceso
rm -rf /opt/downtime-game       # 4. Limpiar scripts viejos
# ... update files ...
systemctl start sg-rojo         # 5. Iniciar nuevo
```

---

## 7. Configuración por Color

| Color | Env `SERVICE_COLOR` | Puerto | Hex | CT VMID | IP |
|-------|---------------------|--------|-----|---------|-----|
| Rojo | `rojo` | 8080 | `#DC143C` | 200 | 192.168.1.200 |
| Azul | `azul` | 8084 | `#1E90FF` | 201 | 192.168.1.204 |
| Verde | `verde` | 8082 | `#00FF7F` | 202 | 192.168.1.202 |
| Amarillo | `amarillo` | 8083 | `#FFD700` | 203 | 192.168.1.203 |

Variables de entorno que acepta el script:

| Variable | Default | Descripción |
|----------|---------|-------------|
| `SERVICE_COLOR` | `rojo` | Color del servicio |
| `SERVICE_PORT` | `8080` | Puerto HTTP |
| `SERVICE_ID` | `sg-<color>` | ID del servicio (para health checks) |

---

## 8. Deploy

### Prerrequisitos

- Ejecutar **desde Proxmox VE** (192.168.1.31)
- Los CTs deben existir y estar configurados con Rocky Linux
- Python 3 instalado en cada CT (`pct exec <vmid> -- which python3`)

### Comandos

```bash
# Deploy a TODOS los CTs
sudo bash services/deploy.sh

# Deploy solo a algunos
sudo bash services/deploy.sh rojo verde

# Deploy a uno solo
sudo bash services/deploy.sh azul
```

### Qué hace el deploy script

1. Verifica que el CT existe y está corriendo
2. Copia `http-service.py` a `/opt/http-service.py` en el CT
3. Crea el archivo de systemd `sg-<color>.service`
4. Detiene el servicio anterior (si existe)
5. Habilita e inicia el nuevo servicio
6. Verifica que el servicio esté activo
7. Prueba el endpoint `/health`

---

## 9. DNS (MikroTik)

Una vez que el MikroTik esté configurado, importar los registros DNS:

```bash
curl -u admin:{{ MIKROTIK_PASSWORD }} -T mikrotik-dns.rsc ftp://192.168.1.253/
```
```rsc
import file=mikrotik-dns.rsc
```

Esto agrega:

| Hostname | IP |
|----------|-----|
| `rojo.downtime.game` (alias: `rojo`) | 192.168.1.200 |
| `azul.downtime.game` (alias: `azul`) | 192.168.1.204 |
| `verde.downtime.game` (alias: `verde`) | 192.168.1.202 |
| `amarillo.downtime.game` (alias: `amarillo`) | 192.168.1.203 |
| `proxmox.downtime.game` (alias: `proxmox`) | 192.168.1.31 |
| `pcrocky.downtime.game` (alias: `pcrocky`) | 192.168.1.54 |
| `router.downtime.game` (alias: `router`) | 192.168.1.253 |

---

## 10. Verificación

```bash
# Probar health check de cada servicio
curl -s http://192.168.1.200:8080/health | python3 -m json.tool
curl -s http://192.168.1.204:8084/health | python3 -m json.tool
curl -s http://192.168.1.202:8082/health | python3 -m json.tool
curl -s http://192.168.1.203:8083/health | python3 -m json.tool

# Ver página web
curl -s http://192.168.1.200:8080/ | head -5

# Ver logs del servicio (dentro del CT)
journalctl -u sg-rojo --no-pager -n 20
```

---

## 11. Troubleshooting

### Servicio no arranca

```bash
# Dentro del CT
systemctl status sg-rojo --no-pager
journalctl -u sg-rojo -n 50 --no-pager

# Verificar que Python 3 existe
which python3

# Probar el script manualmente
SERVICE_COLOR=rojo SERVICE_PORT=8080 python3 /opt/http-service.py
```

### Puerto no responde

```bash
# Verificar que el servicio está escuchando
ss -tlnp | grep 8080

# Verificar firewall dentro del CT
iptables -L -n
```

### CT no reachable

```bash
# Desde Proxmox
pct enter <vmid>
ip addr show
ip route show
ping 192.168.1.253
```
