#!/bin/bash
# =============================================================================
# Deploy Downtime Game a PC de juego
# USO:
#   ./deploy-pc.sh                    # Deploy en modo simulado
#   ./deploy-pc.sh --ssh              # Deploy en modo real (con LXC)
#   ./deploy-pc.sh --ssh --rebuild    # Deploy forzando rebuild
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()   { echo -e "${RED}[ERR]${NC}  $*"; }

# ── Parse args ──
SSH_MODE=false
REBUILD=""
for arg in "$@"; do
    case "$arg" in
        --ssh) SSH_MODE=true ;;
        --rebuild) REBUILD="--build" ;;
        *) err "Argumento desconocido: $arg"; exit 1 ;;
    esac
done

# ── Prerequisitos ──
info "Verificando prerequisitos..."
command -v docker >/dev/null 2>&1 || { err "Docker no instalado"; exit 1; }
command -v git >/dev/null 2>&1 || { err "Git no instalado"; exit 1; }

# Docker Compose (nuevo plugin o legacy)
if docker compose version >/dev/null 2>&1; then
    DC="docker compose"
elif docker-compose --version >/dev/null 2>&1; then
    DC="docker-compose"
else
    err "Docker Compose no instalado"
    exit 1
fi

# ── Directorio del script ──
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# ── Configurar modo ──
if [ "$SSH_MODE" = true ]; then
    info "Modo: SSH REAL (con LXC)"

    # Detectar SSH key automáticamente
    if [ -n "${LXC_SSH_KEY_PATH:-}" ]; then
        SSH_KEY="$LXC_SSH_KEY_PATH"
    else
        # Buscar cualquier key (ed25519, rsa, ecdsa)
        for key in id_ed25519 id_rsa id_ecdsa; do
            if [ -f "$HOME/.ssh/$key" ]; then
                SSH_KEY="$HOME/.ssh/$key"
                break
            fi
        done
    fi

    if [ -z "${SSH_KEY:-}" ] || [ ! -f "$SSH_KEY" ]; then
        err "No se encuentra ninguna SSH key en ~/.ssh/"
        err ""
        err "Generá una key y copiala a Proxmox:"
        err "  ssh-keygen -t ed25519"
        err "  ssh-copy-id root@192.168.1.31"
        err ""
        err "O exportá la ruta manualmente:"
        err "  export LXC_SSH_KEY_PATH=/path/to/tu_key"
        exit 1
    fi

    # Verificar conexión SSH a Proxmox
    LXC_HOST="${LXC_SSH_HOST:-192.168.1.31}"
    LXC_USER="${LXC_SSH_USER:-root}"
    info "Verificando conexión SSH a ${LXC_USER}@${LXC_HOST}..."
    if ! ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -i "$SSH_KEY" "${LXC_USER}@${LXC_HOST}" "hostname" >/dev/null 2>&1; then
        err "No se puede conectar SSH a ${LXC_HOST}"
        err "Verificá que:"
        err "  1. La PC llegue a ${LXC_HOST} (ping)"
        err "  2. La SSH key esté copiada (ssh-copy-id)"
        err "  3. Proxmox esté encendido"
        exit 1
    fi
    ok "Conexión SSH OK: $(ssh -i "$SSH_KEY" "${LXC_USER}@${LXC_HOST}" "hostname" 2>/dev/null)"

    # Exportar vars para docker-compose
    export EXECUTOR_MODE=ssh
    export LXC_SSH_HOST="$LXC_HOST"
    export LXC_SSH_USER="$LXC_USER"
    export LXC_SSH_KEY_PATH="$SSH_KEY"

    info "Configuración SSH:"
    info "  Host:   ${LXC_HOST}"
    info "  User:   ${LXC_USER}"
    info "  Key:    ${SSH_KEY}"
else
    info "Modo: SIMULADO (sin LXC)"
    export EXECUTOR_MODE=simulated
fi

# ── Deploy ──
echo ""
info "🚀 Desplegando Downtime Game..."
echo ""

cd "$SCRIPT_DIR"

# ── Levantar servicios ──
info "Levantando servicios con Docker Compose..."
if [ "$SSH_MODE" = true ]; then
    # En modo SSH, necesitamos pasar las vars de entorno
    $DC -f docker/docker-compose.yml up -d $REBUILD
else
    $DC -f docker/docker-compose.yml up -d $REBUILD
fi

# ── Esperar a que backend esté listo ──
info "Esperando que el backend responda..."
for i in $(seq 1 15); do
    if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
        ok "Backend listo!"
        break
    fi
    if [ "$i" -eq 15 ]; then
        warn "Timeout esperando al backend. Revisá los logs: make docker-logs"
    fi
    sleep 1
done

# ── Resumen ──
echo ""
ok "╔═══════════════════════════════════════════════════╗"
ok "║         DEPLOY COMPLETADO                         ║"
ok "╚═══════════════════════════════════════════════════╝"
echo ""
info "📡 URLs del sistema:"
echo ""
echo "   Dashboard:      http://192.168.1.54/"
echo "   Admin Panel:    http://192.168.1.54/admin/"
echo "   API (health):   http://192.168.1.54/api/games"
echo "   WebSocket:      ws://192.168.1.54/ws"
echo ""

if [ "$SSH_MODE" = true ]; then
    info "🔌 Modo: SSH REAL — conectado a ${LXC_HOST}"
    echo ""
    info "Para probar:"
    echo "   1. Abrí http://192.168.1.54/ (Dashboard)"
    echo "   2. Abrí http://192.168.1.54/admin/ (Admin)"
    echo "   3. Creá una partida en Admin -> Iniciar"
    echo "   4. El dashboard muestra la alarma en vivo"
    echo "   5. El SSH executor detiene el servicio en el LXC"
    echo ""
else
    info "🔌 Modo: SIMULADO (sin LXC)"
    echo ""
    info "Para modo real:"
    echo "   ./deploy-pc.sh --ssh"
    echo ""
fi

info "Comandos útiles:"
echo "   make docker-logs     — Ver logs de todos los servicios"
echo "   make docker-down     — Detener todos los servicios"
echo "   docker compose ps    — Estado de los servicios"
echo ""
