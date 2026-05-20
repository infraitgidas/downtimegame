# Guía rápida — Downtime Game en la Feria

> Documento para los voluntarios que operan el stand.
> Tiempo de lectura: 5 minutos.

---

## 🎯 Objetivo del juego

El jugador asume el rol de un **SysAdmin/SRE**. Un servicio de TI se cae (Rojo, Azul, Verde o Amarillo) y el jugador tiene **2-3 minutos** para conectarse vía SSH al servidor afectado, diagnosticar el problema y arreglarlo.

---

## 🖥 Acceso a los sistemas

| Sistema | IP | Usuario | Contraseña |
|---------|----|---------|------------|
| **Dashboard** (pantalla grande) | `http://192.168.1.54/` | — | — |
| **Admin Panel** (control) | `http://192.168.1.54/admin/` | — | — |
| **CT Rojo** | `ssh gidas@192.168.1.200` | `gidas` | `gidas` |
| **CT Azul** | `ssh gidas@192.168.1.204` | `gidas` | `gidas` |
| **CT Verde** | `ssh gidas@192.168.1.202` | `gidas` | `gidas` |
| **CT Amarillo** | `ssh gidas@192.168.1.203` | `gidas` | `gidas` |

> Los jugadores se conectan a los CTs con `ssh gidas@<ip>` y password `gidas`.

---

## 🎮 Cómo funciona una partida

```
1. Admin → "Crear Partida" (nombre del jugador)
2. Admin → Seleccionar partida pendiente → "Iniciar"
   (Opcional: elegir escenario específico en el selector)
3. Dashboard → Muestra alarma con cronómetro y pistas
4. Dashboard → Pistas aparecen cada 60 segundos
5. Jugador → SSH al servicio afectado y lo repara
6. Admin → "Resolver Incidente" (si el jugador avisa que ya arregló)
   O
   Dashboard → "ABANDONAR — VER SOLUCIÓN" (si se rinden)
```

---

## 🖥 El Dashboard (pantalla principal)

Se muestra en el monitor grande. Muestra:

- **Alarma roja pulsante** cuando hay un incidente activo
- **Cronómetro DOWNTIME** gigante con cuenta regresiva
- **Barra de progreso** que se llena a medida que pasa el tiempo
- **Pistas** que se revelan cada 60 segundos
- **Botón "ABANDONAR"** para rendirse y ver la solución
- **Tarjetas de servicio** con estado online/offline
- **Cuadrícula de 4 servicios** con health checks cada 5s
- **Footer** con logo GIDAS, INFRA IT, UTN

---

## 🔧 El Admin Panel (control)

Se usa en la notebook del operador. Permite:

### Controles de juego
- **Crear Partida** — Ingresar nombre del jugador
- **Iniciar Partida** — Seleccionar partida pendiente + escenario (opcional)
- **Resolver Incidente** — Cuando el jugador arregla el servicio
- **Abandonar** — Forzar abandono de una partida activa

### Modo Demo 🎮
- **Botón "Demo"** — Inicia ciclo automático: crea partida, espera 20-30s, resuelve, repite
- **Botón "⏹ Demo R1"** — Detiene el demo (muestra el round actual)
- Ideal para cuando no hay jugadores y querés mostrar el juego en acción

### Administración
- **🛑 Reset** — Abandona todas las partidas activas y restaura servicios
- **🗑 Eliminar** — Borra partidas individuales (no activas)
- **🔄 Reiniciar** — Crea nueva partida con el mismo jugador
- **Reset Global** — Confirma en dos pasos para evitar accidentes

### Información
- **Lista de partidas** — Historial con estado y puntaje
- **Leaderboard** — Top 10 jugadores
- **Escenarios** — Catálogo completo de 8 escenarios
- **WebSocket Events** — Log de eventos en vivo para debug

---

## 🌐 Escenarios disponibles

| # | Servicio | Tipo | Severidad | Dificultad | Tiempo |
|---|----------|------|-----------|------------|--------|
| 1 | Rojo | crash | critical | 2 | 120s |
| 2 | Azul | hang | critical | 3 | 150s |
| 3 | Verde | latency | major | 4 | 180s |
| 4 | Amarillo | dns_failure | major | 3 | 150s |
| 5 | Azul | crash | critical | 2 | 120s |
| 6 | Rojo | latency | major | 4 | 180s |
| 7 | Verde | hang | critical | 3 | 150s |
| 8 | Rojo | dns_failure | major | 3 | 120s |

---

## 📡 URLs del sistema

```
Dashboard:     http://192.168.1.54/
Admin Panel:   http://192.168.1.54/admin/
Health Check:  http://192.168.1.54/health
API Games:     http://192.168.1.54/api/games
Grafana:       http://192.168.1.205:3000/   (admin / gidas)
```

---

## 🚨 Posibles problemas y soluciones

### "El dashboard no muestra nada"
1. Verificar que `http://192.168.1.54/health` responda `{"status":"ok"}`
2. Si no responde → reiniciar Docker en la PC de juego

### "No me puedo conectar SSH al CT"
1. Verificar que el CT esté corriendo: `pct status 200` (desde Proxmox)
2. Verificar conectividad: `ping 192.168.1.200`
3. Verificar SSHD: `ssh gidas@192.168.1.200`

### "El modo demo no arranca"
1. Hacer un Reset Global primero
2. Verificar que no haya una partida activa trabada
3. Intentar de nuevo

### "Los servicios no responden health check"
1. Desde Proxmox: `pct exec 200 -- systemctl status sg-rojo`
2. Si está caído: `pct exec 200 -- systemctl restart sg-rojo`

---

## ✅ Checklist pre-feria

### Red
- [ ] MikroTik encendido y configurado
- [ ] Proxmox accesible (192.168.1.31)
- [ ] PC de juego encendida (192.168.1.54)
- [ ] DNS resolviendo correctamente
- [ ] WiFi funcionando para los jugadores

### LXC Containers
- [ ] CT 200 (Rojo) — corriendo, sg-rojo activo, health check OK
- [ ] CT 201 (Azul) — corriendo, sg-azul activo, health check OK
- [ ] CT 202 (Verde) — corriendo, sg-verde activo, health check OK
- [ ] CT 203 (Amarillo) — corriendo, sg-amarillo activo, health check OK
- [ ] SSH gidas@ funciona en los 4 CTs (password: gidas)
- [ ] Los 4 servicios responden HTTP en sus puertos

### Docker (PC de juego)
- [ ] Backend running (http://192.168.1.54/health → ok)
- [ ] Dashboard cargando (http://192.168.1.54/ → Downtime Game)
- [ ] Admin cargando (http://192.168.1.54/admin/ → Downtime Game)
- [ ] nginx proxy funcionando

### Monitoreo (CT 205)
- [ ] Grafana accesible (http://192.168.1.205:3000/)
- [ ] Prometheus scrape targets UP
- [ ] Dashboards de juego cargando

### Prueba rápida
- [ ] Crear partida → Iniciar → Ver alarma en dashboard
- [ ] Pistas aparecen después de 60s
- [ ] Botón "Abandonar" muestra solución
- [ ] Resolver incidente → score > 0
- [ ] Modo Demo funciona
- [ ] Leaderboard muestra puntajes

---

## 📞 Contactos

| Rol | Persona | Teléfono |
|-----|---------|----------|
| Responsable INFRA IT | | |
| Operador del stand | | |

---

> **Versión**: 1.0 · **Última actualización**: 2026-05-20
> Proyecto: [github.com/infraitgidas/downtimegame](https://github.com/infraitgidas/downtimegame)
