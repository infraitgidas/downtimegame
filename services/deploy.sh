#!/bin/bash
# =============================================================================
# Deploy HTTP Service to CTs - Downtime Game
# Ejecutar DESDE Proxmox (192.168.1.31) via SSH o consola
# =============================================================================
# USO:
#   ./deploy.sh                  # Deploy a todos los CTs
#   ./deploy.sh rojo             # Deploy solo a un CT
#   ./deploy.sh rojo azul verde  # Deploy a varios
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICE_SCRIPT="$SCRIPT_DIR/http-service.py"

# ── Config por CT ──
declare -A CT_VMID=(
    [rojo]=200
    [azul]=201
    [verde]=202
    [amarillo]=203
)

declare -A CT_IP=(
    [rojo]="192.168.1.200"
    [azul]="192.168.1.204"
    [verde]="192.168.1.202"
    [amarillo]="192.168.1.203"
)

declare -A CT_PORT=(
    [rojo]=8080
    [azul]=8084
    [verde]=8082
    [amarillo]=8083
)

# ── Colores para output ──
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

color_echo() {
    local color=$1; shift
    echo -e "${color}$*${NC}"
}

# ── Validar argumentos ──
if [ $# -eq 0 ]; then
    COLORS_TO_DEPLOY=("rojo" "azul" "verde" "amarillo")
else
    COLORS_TO_DEPLOY=("$@")
fi

# ── Verificar que el script existe ──
if [ ! -f "$SERVICE_SCRIPT" ]; then
    color_echo "$RED" "ERROR: No se encuentra $SERVICE_SCRIPT"
    exit 1
fi

# ── Verificar que estamos en Proxmox ──
if [ ! -f /etc/pve/version ]; then
    color_echo "$YELLOW" "⚠️  No parece que estemos en Proxmox (/etc/pve/version no existe)"
    color_echo "$YELLOW" "   Los comandos 'pct' pueden no estar disponibles."
    color_echo "$YELLOW" "   ¿Querés continuar de todas formas? (s/N)"
    read -r reply
    if [ "$reply" != "s" ] && [ "$reply" != "S" ]; then
        exit 1
    fi
fi

# ══════════════════════════════════════════════════════════════
#  DEPLOY
# ══════════════════════════════════════════════════════════════

for color in "${COLORS_TO_DEPLOY[@]}"; do
    vmid="${CT_VMID[$color]}"
    ip="${CT_IP[$color]}"
    port="${CT_PORT[$color]}"

    if [ -z "$vmid" ]; then
        color_echo "$RED" "❌ Color desconocido: $color"
        continue
    fi

    echo ""
    color_echo "$BLUE" "╔══════════════════════════════════════════════╗"
    color_echo "$BLUE" "║  Desplegando en $color (CT $vmid - $ip:$port) ║"
    color_echo "$BLUE" "╚══════════════════════════════════════════════╝"

    # ── Verificar que el CT existe y está corriendo ──
    if pct status "$vmid" &>/dev/null; then
        ct_status=$(pct status "$vmid" | awk '{print $2}')
        if [ "$ct_status" != "running" ]; then
            color_echo "$YELLOW" "   ▶ Iniciando CT $vmid..."
            pct start "$vmid"
            sleep 3
        fi
    else
        color_echo "$RED" "   ❌ CT $vmid no existe en Proxmox"
        continue
    fi

    # ── Copiar el script Python al CT ──
    color_echo "$YELLOW" "   📄 Copiando http-service.py → CT $vmid..."
    pct push "$vmid" "$SERVICE_SCRIPT" "/opt/http-service.py" --perms 755

    # ── Crear systemd service ──
    color_echo "$YELLOW" "   ⚙️  Instalando systemd service..."
    cat > /tmp/sg-${color}.service << SERVICE
[Unit]
Description=Downtime Game - Servicio ${color^}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=nobody
Group=nobody
Environment=SERVICE_COLOR=$color
Environment=SERVICE_PORT=$port
Environment=SERVICE_ID=sg-${color}
ExecStart=/usr/bin/python3 /opt/http-service.py
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
SERVICE

    pct push "$vmid" "/tmp/sg-${color}.service" "/etc/systemd/system/sg-${color}.service" --perms 644

    # ── Detener servicio viejo (si existe) ──
    color_echo "$YELLOW" "   🔄 Deteniendo servicio anterior..."
    pct exec "$vmid" -- systemctl stop "sg-${color}" 2>/dev/null || true
    pct exec "$vmid" -- systemctl disable "sg-${color}" 2>/dev/null || true

    # ── Recargar systemd, habilitar e iniciar ──
    color_echo "$YELLOW" "   🚀 Iniciando nuevo servicio..."
    pct exec "$vmid" -- systemctl daemon-reload
    pct exec "$vmid" -- systemctl enable "sg-${color}"
    pct exec "$vmid" -- systemctl restart "sg-${color}"

    # ── Esperar y verificar ──
    sleep 2
    if pct exec "$vmid" -- systemctl is-active "sg-${color}" &>/dev/null; then
        color_echo "$GREEN" "   ✅ Servicio ${color} activo"
    else
        color_echo "$RED" "   ❌ Servicio ${color} NO arrancó"
        pct exec "$vmid" -- systemctl status "sg-${color}" --no-pager 2>&1 || true
        continue
    fi

    # ── Test HTTP ──
    color_echo "$YELLOW" "   🔍 Verificando endpoint /health..."
    sleep 1
    if pct exec "$vmid" -- curl -sf "http://127.0.0.1:${port}/health" &>/dev/null; then
        health=$(pct exec "$vmid" -- curl -s "http://127.0.0.1:${port}/health")
        color_echo "$GREEN" "   ✅ Health check OK: $health"
    else
        color_echo "$RED" "   ❌ Health check FAILED"
        pct exec "$vmid" -- curl -v "http://127.0.0.1:${port}/health" 2>&1 || true
    fi

    # ── Limpiar temporales ──
    rm -f "/tmp/sg-${color}.service"
done

echo ""
color_echo "$GREEN" "╔══════════════════════════════════════════════╗"
color_echo "$GREEN" "║         DEPLOY COMPLETADO                     ║"
color_echo "$GREEN" "╚══════════════════════════════════════════════╝"
echo ""
color_echo "$YELLOW" "Próximos pasos:"
echo "  1. Verificar desde cualquier browser:"
for color in "${COLORS_TO_DEPLOY[@]}"; do
    echo "     http://${CT_IP[$color]}:${CT_PORT[$color]}/"
done
echo ""
echo "  2. Agregar DNS en MikroTik (cuando esté configurado)"
echo ""
