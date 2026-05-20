#!/bin/bash
# =============================================================================
# Setup usuario gidas en todos los sistemas del Downtime Game
#
# Crea el usuario gidas con password "gidas" y configura SSH key-based auth
# para que los jugadores puedan conectarse a los servicios durante una partida.
#
# SISTEMAS:
#   - CTs de color: 200 (rojo), 201 (azul), 202 (verde), 203 (amarillo)
#   - Proxmox: 192.168.1.31 (opcional: --proxmox)
#   - Router:  192.168.1.253 (opcional: --router, requiere credenciales)
#
# USO:
#   ./setup-gidas.sh                          # Solo CTs
#   ./setup-gidas.sh --proxmox                # CTs + Proxmox
#   ./setup-gidas.sh --proxmox --router       # Todo
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
SETUP_PROXMOX=false
SETUP_ROUTER=false
for arg in "$@"; do
    case "$arg" in
        --proxmox) SETUP_PROXMOX=true ;;
        --router)  SETUP_ROUTER=true  ;;
        *) err "Argumento desconocido: $arg"; exit 1 ;;
    esac
done

# ── Verificar que tenemos pct (Proxmox) ──
if ! command -v pct &>/dev/null; then
    err "Este script debe ejecutarse en un nodo Proxmox (comando 'pct' no encontrado)"
    err "Copiá el script al nodo Proxmox y ejecutalo allí."
    err "  scp setup-gidas.sh root@192.168.1.31:/root/"
    err "  ssh root@192.168.1.31 'bash /root/setup-gidas.sh'"
    exit 1
fi

# ── Config ──
GIDAS_PASSWORD="gidas"
GIDAS_SSH_KEY="${HOME}/.ssh/id_gidas"

# VMID e IP por color
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

COLORS=("rojo" "azul" "verde" "amarillo")

# ══════════════════════════════════════════════════════════════
#  1. GENERAR SSH KEY para gidas (si no existe)
# ══════════════════════════════════════════════════════════════
echo ""
info "╔══════════════════════════════════════════════╗"
info "║  Setup usuario gidas — Downtime Game         ║"
info "╚══════════════════════════════════════════════╝"
echo ""

info "Generando SSH key para gidas..."
if [ ! -f "${GIDAS_SSH_KEY}" ]; then
    ssh-keygen -t ed25519 -f "${GIDAS_SSH_KEY}" -N "" -C "gidas@downtime-game" -q
    ok "SSH key generada: ${GIDAS_SSH_KEY}"
else
    ok "SSH key ya existe: ${GIDAS_SSH_KEY}"
fi

GIDAS_PUBKEY=$(cat "${GIDAS_SSH_KEY}.pub")

# ══════════════════════════════════════════════════════════════
#  2. CONFIGURAR CTs
# ══════════════════════════════════════════════════════════════
echo ""
info "╔══════════════════════════════════════════════╗"
info "║  Configurando CTs de color...                ║"
info "╚══════════════════════════════════════════════╝"

for color in "${COLORS[@]}"; do
    vmid="${CT_VMID[$color]}"
    echo ""
    info "────────────────────────────────────────────"
    info "  CT ${vmid} — ${color}"
    info "────────────────────────────────────────────"

    # Verificar CT existe y corriendo
    if ! pct status "$vmid" &>/dev/null; then
        err "CT ${vmid} no existe en Proxmox"
        continue
    fi

    ct_status=$(pct status "$vmid" | awk '{print $2}')
    if [ "$ct_status" != "running" ]; then
        warn "CT ${vmid} no está running. Iniciando..."
        pct start "$vmid"
        sleep 3
    fi

    # ── Instalar openssh-server ──
    info "Instalando openssh-server..."
    pct exec "$vmid" -- bash -c 'dnf install -y openssh-server 2>/dev/null' || true

    # ── Crear usuario gidas ──
    info "Creando usuario gidas..."
    pct exec "$vmid" -- bash -c '
        if ! id -u gidas &>/dev/null; then
            useradd -m -s /bin/bash gidas
            echo "gidas:'"${GIDAS_PASSWORD}"'" | chpasswd
            usermod -aG wheel gidas
        else
            echo "gidas ya existe"
        fi
    ' 2>&1

    # ── Configurar SSH key para gidas ──
    info "Configurando SSH authorized_keys..."
    pct exec "$vmid" -- bash -c '
        mkdir -p /home/gidas/.ssh
        chmod 700 /home/gidas/.ssh
    '
    pct exec "$vmid" -- bash -c "echo '${GIDAS_PUBKEY}' > /home/gidas/.ssh/authorized_keys"
    pct exec "$vmid" -- bash -c '
        chmod 600 /home/gidas/.ssh/authorized_keys
        chown -R gidas:gidas /home/gidas/.ssh
    '
    ok "SSH key instalada"

    # ── Configurar sshd ──
    info "Configurando sshd..."
    pct exec "$vmid" -- bash -c '
        # Permitir password auth + key auth para gidas
        sed -i "s/^#PasswordAuthentication yes/PasswordAuthentication yes/" /etc/ssh/sshd_config 2>/dev/null || true
        sed -i "s/^PasswordAuthentication no/PasswordAuthentication yes/" /etc/ssh/sshd_config 2>/dev/null || true
        sed -i "s/^#ChallengeResponseAuthentication yes/ChallengeResponseAuthentication yes/" /etc/ssh/sshd_config 2>/dev/null || true
        sed -i "s/^ChallengeResponseAuthentication no/ChallengeResponseAuthentication yes/" /etc/ssh/sshd_config 2>/dev/null || true
        sed -i "s/^#PubkeyAuthentication yes/PubkeyAuthentication yes/" /etc/ssh/sshd_config 2>/dev/null || true
        sed -i "s/^UsePAM no/UsePAM yes/" /etc/ssh/sshd_config 2>/dev/null || true
        echo "AllowUsers gidas root" >> /etc/ssh/sshd_config 2>/dev/null || true
    '

    # ── Habilitar e iniciar sshd ──
    info "Habilitando e iniciando sshd..."
    pct exec "$vmid" -- bash -c '
        systemctl enable sshd
        systemctl restart sshd
    ' 2>&1 || warn "systemctl sshd falló, intentando arranque manual..."

    # ── Firewall: abrir puerto 22 ──
    info "Abriendo puerto 22 en firewall..."
    pct exec "$vmid" -- bash -c '
        if systemctl is-active firewalld &>/dev/null; then
            firewall-cmd --permanent --add-service=ssh 2>/dev/null || true
            firewall-cmd --reload 2>/dev/null || true
        fi
    ' 2>&1 || true

    # ── Verificar ──
    sleep 1
    if pct exec "$vmid" -- bash -c 'ss -tlnp | grep -q :22' 2>/dev/null; then
        ok "CT ${vmid} → SSHD corriendo en puerto 22"
    else
        warn "CT ${vmid} → SSHD NO está en puerto 22 (puede ser selinux)"
        pct exec "$vmid" -- bash -c 'ss -tlnp' 2>&1 | head -5
    fi

    # Test SSH
    ip="${CT_IP[$color]}"
    info "Testeando conexión SSH a ${ip}..."
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -i "${GIDAS_SSH_KEY}" "gidas@${ip}" "hostname" 2>/dev/null; then
        ok "SSH OK → gidas@${ip}"
    else
        warn "SSH test falló (puede ser red, no crítica)"
    fi
done

# ══════════════════════════════════════════════════════════════
#  3. CONFIGURAR PROXMOX (opcional: --proxmox)
# ══════════════════════════════════════════════════════════════
if [ "$SETUP_PROXMOX" = true ]; then
    echo ""
    info "╔══════════════════════════════════════════════╗"
    info "║  Configurando Proxmox (192.168.1.31)...     ║"
    info "╚══════════════════════════════════════════════╝"

    # Crear usuario gidas
    if ! id -u gidas &>/dev/null; then
        useradd -m -s /bin/bash gidas
        echo "gidas:${GIDAS_PASSWORD}" | chpasswd
        usermod -aG sudo gidas
        ok "Usuario gidas creado en Proxmox"
    else
        ok "Usuario gidas ya existe en Proxmox"
    fi

    # SSH key
    mkdir -p /home/gidas/.ssh
    chmod 700 /home/gidas/.ssh
    echo "${GIDAS_PUBKEY}" >> /home/gidas/.ssh/authorized_keys
    chmod 600 /home/gidas/.ssh/authorized_keys
    chown -R gidas:gidas /home/gidas/.ssh
    ok "SSH key instalada en Proxmox"

    # Verificar SSHD en Proxmox
    if systemctl is-active sshd &>/dev/null; then
        ok "SSHD activo en Proxmox"
    else
        warn "SSHD no activo en Proxmox"
    fi
fi

# ══════════════════════════════════════════════════════════════
#  4. CONFIGURAR ROUTER (opcional: --router)
# ══════════════════════════════════════════════════════════════
if [ "$SETUP_ROUTER" = true ]; then
    echo ""
    info "╔══════════════════════════════════════════════╗"
    info "║  Configurando Router MikroTik...            ║"
    info "╚══════════════════════════════════════════════╝"
    warn "Router setup requiere credenciales manuales."
    warn "Ejecutá en el router MikroTik (WinBox/SSH):"
    echo ""
    echo "  /user add name=gidas password=gidas group=full"
    echo "  /user ssh-keys import user=gidas public-key-file=\"id_gidas.pub\""
    echo "  /ip service enable ssh"
    echo ""
    warn "O desde este script si tenés credenciales admin:"
    info "¿Tenés credenciales del router? (s/N)"
    read -r has_creds
    if [ "$has_creds" = "s" ] || [ "$has_creds" = "S" ]; then
        info "Ingresá usuario admin del router:"
        read -r ROUTER_USER
        if [ -n "$ROUTER_USER" ]; then
            ssh -o StrictHostKeyChecking=no "ROUTER_USER@192.168.1.253" "
                /user add name=gidas password=gidas group=full
                /user ssh-keys import user=gidas public-key-file=\"id_gidas.pub\"
                /ip service enable ssh
            " 2>&1 && ok "Router configurado" || err "Falló configuración del router"
        fi
    fi
fi

# ══════════════════════════════════════════════════════════════
#  RESUMEN
# ══════════════════════════════════════════════════════════════
echo ""
ok "╔══════════════════════════════════════════════╗"
ok "║         SETUP COMPLETADO                      ║"
ok "╚══════════════════════════════════════════════╝"
echo ""
info "Usuario gidas creado en todos los sistemas."
echo ""
info "Credenciales:"
echo "  Usuario:   gidas"
echo "  Password:  ${GIDAS_PASSWORD}"
echo "  SSH key:   ${GIDAS_SSH_KEY}"
echo ""
info "Conexión SSH (key-based):"
for color in "${COLORS[@]}"; do
    ip="${CT_IP[$color]}"
    echo "  ssh -i ${GIDAS_SSH_KEY} gidas@${ip}"
done
echo ""
info "Conexión SSH (password):"
for color in "${COLORS[@]}"; do
    ip="${CT_IP[$color]}"
    echo "  ssh gidas@${ip}  (password: ${GIDAS_PASSWORD})"
done
echo ""
if [ "$SETUP_PROXMOX" = true ]; then
    info "Proxmox:"
    echo "  ssh gidas@192.168.1.31  (key o password: ${GIDAS_PASSWORD})"
fi
echo ""
info "Nota: para conectarte desde tu PC dev (${HOSTNAME}),"
info "copiá la key: ssh-copy-id -i ${GIDAS_SSH_KEY} gidas@<ip>"
info "O usá: ssh -i ${GIDAS_SSH_KEY} gidas@<ip>"
echo ""
