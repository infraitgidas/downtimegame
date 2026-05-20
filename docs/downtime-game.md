# Downtime Game

> Contexto: Feria universitaria de laboratorios científicos y de investigación.
> Experiencia gamificada con hardware real orientada a tareas de SysAdmin/SRE.
> Documentación completa de infraestructura en [`docs/infrastructure.md`](infrastructure.md).
> Documentación de servicios HTTP por color en [`docs/services.md`](services.md).

---

## Concepto

El **Downtime Game** es un juego presencial donde un participante asume el rol de SysAdmin/SRE.
Debe diagnosticar y resolver incidentes (downtime) que ocurren aleatoriamente sobre servicios
ejecutándose en infraestructura real (Proxmox LXC, red MikroTik, Docker).

## Mecánica del Juego

1. **4 servicios por color** (🔴 Rojo, 🔵 Azul, 🟢 Verde, 🟡 Amarillo) corriendo en LXC en Proxmox
2. **Dashboard de observabilidad** en Docker (PC Rocky Linux) monitorea uptime, latencia, ping
3. **El jugador ingresa su nombre** y comienza una partida
4. **El backend elige un escenario aleatorio** que genera un downtime real en uno de los servicios
5. **El dashboard muestra alarma + cronómetro rojo** — el jugador debe diagnosticar y resolver
6. **Al recuperar el servicio**, se guarda el tiempo de resolución con el nombre del jugador

## Limitaciones y Supuestos

- ✅ Todo se ejecuta en el stack hardware local (on-premise 100%)
- ✅ Jugadores pueden tener experiencia nula o mucha
- ✅ Debe funcionar con baja o nula conectividad a internet
- ❌ Sin dependencia de cloud, SaaS ni APIs externas

## Estado del Proyecto

| Fase | Descripción | Estado |
|------|-------------|--------|
| **Fase 0** | Concepto y especificación | ✅ COMPLETADA |
| **Fase 1** | Foundation — backend Go, TUI, dashboards | 🔄 En progreso |
| **Fase 2** | Escenarios de downtime + lógica de juego | ⏳ Pendiente |
| **Fase 3** | Dashboard con métricas y alarmas reales | ⏳ Pendiente |
| **Fase 4** | Integración con hardware real (LXC, red) | ⏳ Pendiente |
| **Fase 5** | Tests, pulido, deploy en feria | ⏳ Pendiente |

## Documentación Relacionada

- [`docs/infrastructure.md`](infrastructure.md) — Topología de red, hardware, credenciales, stack tecnológico, plan de despliegue
