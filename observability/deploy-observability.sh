#!/bin/bash
# =============================================================================
# Deploy Observability Stack — Downtime Game
# Crea CT 205 en Proxmox con Prometheus + Grafana + Exporters
#
# USO:
#   ./deploy-observability.sh                    # Deploy completo
#   ./deploy-observability.sh --skip-create      # Salta creacion del CT
#   ./deploy-observability.sh --only-dashboard   # Solo actualiza dashboard
# =============================================================================
#
# Requisitos:
#   - SSH key-based auth a root@192.168.1.31 (Proxmox)
#   - LXC template rockylinux-10-defaults-20251001.tar.xz disponible en Proxmox
#     (verificar con: pveam available | grep rockylinux-10)
#   - Red 192.168.1.0/24 accesible desde el CT
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()   { echo -e "${RED}[ERR]${NC}  $*"; }

# ── Config ──
PROXMOX_HOST="${LXC_SSH_HOST:-192.168.1.31}"
PROXMOX_USER="${LXC_SSH_USER:-root}"
CT_ID=205
CT_HOSTNAME="sg-monitoring"
CT_IP="192.168.1.205/24"
CT_GW="192.168.1.1"
# Template de Rocky Linux 10 en Proxmox (mismo que servicios de color).
# Nombre exacto verificado: pveam list local | grep rockylinux-10
CT_OS="rockylinux-10-default_20251001_amd64.tar.xz"
# Storage de templates (donde está el template descargado con pveam)
CT_TEMPLATE_STORAGE="local"
# Storage para el rootfs del CT (local-lvm es el default de Proxmox).
# Verificar con: pct storage | grep -E "local|lvm|zfs"
CT_ROOTFS_STORAGE="${CT_ROOTFS_STORAGE:-local-lvm}"
CT_CORES=1
CT_MEMORY=1024
CT_SWAP=256
CT_DISK=10

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROMETHEUS_VERSION="3.2.1"
NODE_EXPORTER_VERSION="1.8.2"
BLACKBOX_VERSION="0.25.0"
GRAFANA_VERSION="latest"

# ── Parse args ──
SKIP_CREATE=false
ONLY_DASHBOARD=false
for arg in "$@"; do
    case "$arg" in
        -h|--help)
            echo "Uso: $0 [--skip-create] [--only-dashboard] [-h]"
            echo ""
            echo "  --skip-create      Salta creacion del CT y reinstalacion de servicios"
            echo "  --only-dashboard   Solo actualiza el dashboard de Grafana (CT debe existir)"
            echo "  -h, --help         Muestra esta ayuda"
            echo ""
            echo "Requisitos:"
            echo "  - SSH key a root@${PROXMOX_HOST}"
            echo "  - Template '${CT_OS}' en Proxmox"
            echo ""
            exit 0
            ;;
        --skip-create) SKIP_CREATE=true ;;
        --only-dashboard) ONLY_DASHBOARD=true ;;
        *) err "Argumento desconocido: $arg (usa --help para ayuda)"; exit 1 ;;
    esac
done

# ── SSH helper ──
ssh_proxmox() {
    ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 \
        "${PROXMOX_USER}@${PROXMOX_HOST}" "$@"
}

scp_proxmox() {
    scp -o StrictHostKeyChecking=no -o ConnectTimeout=5 \
        "$1" "${PROXMOX_USER}@${PROXMOX_HOST}:$2"
}

pct_exec() {
    # Usa printf '%q' para escapar cada argumento correctamente.
    # Sin esto, los bash -c con comandos largos se rompen al pasar por
    # ssh_proxmox porque las comillas anidadas colapsan.
    local cmd=""
    for arg in "$@"; do
        if [ -z "$cmd" ]; then
            cmd="$(printf '%q' "$arg")"
        else
            cmd="$cmd $(printf '%q' "$arg")"
        fi
    done
    ssh_proxmox "pct exec ${CT_ID} -- ${cmd}"
}

ct_upload() {
    local src="$1" dst="$2"
    local tmp="/tmp/$(basename "$src")"
    scp_proxmox "$src" "$tmp"
    ssh_proxmox "pct push ${CT_ID} ${tmp} ${dst} --perms $3"
    ssh_proxmox "rm -f ${tmp}"
}

# ══════════════════════════════════════════════════════════════
#  MAIN
# ══════════════════════════════════════════════════════════════

echo ""
info "╔════════════════════════════════════════════════════╗"
info "║  Deploy Observability Stack — Downtime Game       ║"
info "║  CT ${CT_ID} — ${CT_HOSTNAME} — ${CT_IP}              ║"
info "╚════════════════════════════════════════════════════╝"
echo ""

# ── 0. Verify SSH connection ──
info "Verificando conexion a Proxmox..."
if ! ssh_proxmox "hostname" >/dev/null 2>&1; then
    err "No se puede conectar a ${PROXMOX_HOST}"
    exit 1
fi
ok "Proxmox conectado: $(ssh_proxmox "hostname" 2>/dev/null)"

# ── 1. Create CT (if needed) ──
if [ "$ONLY_DASHBOARD" = true ]; then
    info "Modo: solo dashboard."
    info "Verificando CT ${CT_ID}..."
    if ! ssh_proxmox "pct status ${CT_ID}" >/dev/null 2>&1; then
        err "CT ${CT_ID} no existe. Primero corre el script sin --only-dashboard."
        exit 1
    fi
    CT_STATUS=$(ssh_proxmox "pct status ${CT_ID}" | awk '{print $2}')
    if [ "$CT_STATUS" != "running" ]; then
        info "CT ${CT_ID} está detenido. Iniciando..."
        ssh_proxmox "pct start ${CT_ID}"
        sleep 5
    fi
    ok "CT ${CT_ID} listo ($(ssh_proxmox "pct status ${CT_ID}" | awk '{print $2}'))"
elif [ "$SKIP_CREATE" = true ]; then
    info "Saltando creacion del CT (--skip-create)"
    info "Verificando que CT ${CT_ID} existe..."
    if ! ssh_proxmox "pct status ${CT_ID}" >/dev/null 2>&1; then
        err "CT ${CT_ID} no existe. Usa --skip-create solo si ya existe."
        exit 1
    fi
    ok "CT ${CT_ID} existe"
else
    info "Paso 1: Creando CT ${CT_ID}..."
    if ssh_proxmox "pct status ${CT_ID}" >/dev/null 2>&1; then
        info "CT ${CT_ID} ya existe. Verificando estado..."
        CT_STATUS=$(ssh_proxmox "pct status ${CT_ID}" | awk '{print $2}')
        if [ "$CT_STATUS" != "running" ]; then
            info "Iniciando CT ${CT_ID}..."
            ssh_proxmox "pct start ${CT_ID}"
            sleep 5
        fi
        ok "CT ${CT_ID} listo (${CT_STATUS})"
    else
        info "Creando nuevo CT ${CT_ID}..."
        ssh_proxmox "pct create ${CT_ID} ${CT_TEMPLATE_STORAGE}:vztmpl/${CT_OS} \
            --hostname ${CT_HOSTNAME} \
            --net0 name=eth0,bridge=vmbr0,ip=${CT_IP},gw=${CT_GW} \
            --cores ${CT_CORES} \
            --memory ${CT_MEMORY} \
            --swap ${CT_SWAP} \
            --storage ${CT_ROOTFS_STORAGE} \
            --rootfs ${CT_ROOTFS_STORAGE}:${CT_DISK} \
            --unprivileged 1 \
            --features keyctl=1,nesting=1 \
            --start 1" || {
            err "Fallo al crear CT. Verificá que el template ${CT_OS} exista."
            err "Comando: pveam available | grep rockylinux-10-defaults"
            exit 1
        }
        ok "CT ${CT_ID} creado e iniciado"
        sleep 5
    fi

    # ── 2. Wait for CT to be ready ──
    info "Esperando que el CT esté listo..."
    for i in $(seq 1 30); do
        if ssh_proxmox "pct exec ${CT_ID} -- hostname" >/dev/null 2>&1; then
            break
        fi
        sleep 2
    done
    ok "CT listo: $(ssh_proxmox "pct exec ${CT_ID} -- hostname" 2>/dev/null)"

    # ── 3. Install dependencies inside CT ──
    info "Paso 3: Instalando dependencias (dnf)..."
    pct_exec dnf makecache -q
    pct_exec dnf install -y -q \
        tar curl wget gnupg2 \
        libcap which
    ok "Dependencias instaladas"

    # ── 4. Create system users ──
    info "Paso 4: Creando usuarios del sistema..."
    for user in prometheus node_exporter blackbox; do
        pct_exec useradd --no-create-home --shell /bin/false "${user}" 2>/dev/null || true
    done
    ok "Usuarios creados: prometheus, node_exporter, blackbox"

    # ── 5. Download and install Prometheus ──
    info "Paso 5: Instalando Prometheus ${PROMETHEUS_VERSION}..."
    pct_exec bash -c "
        cd /tmp
        curl -sLO https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz
        tar xzf prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz
        cp prometheus-${PROMETHEUS_VERSION}.linux-amd64/prometheus /usr/local/bin/
        cp prometheus-${PROMETHEUS_VERSION}.linux-amd64/promtool /usr/local/bin/
        mkdir -p /etc/prometheus /var/lib/prometheus
        rm -rf prometheus-${PROMETHEUS_VERSION}.linux-amd64*
        chown -R prometheus:prometheus /etc/prometheus /var/lib/prometheus
    "
    ok "Prometheus instalado"

    # ── 6. Download and install Node Exporter ──
    info "Paso 6: Instalando Node Exporter ${NODE_EXPORTER_VERSION}..."
    pct_exec bash -c "
        cd /tmp
        curl -sLO https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz
        tar xzf node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz
        cp node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter /usr/local/bin/
        rm -rf node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64*
    "
    ok "Node Exporter instalado"

    # ── 7. Download and install Blackbox Exporter ──
    info "Paso 7: Instalando Blackbox Exporter ${BLACKBOX_VERSION}..."
    pct_exec bash -c "
        cd /tmp
        curl -sLO https://github.com/prometheus/blackbox_exporter/releases/download/v${BLACKBOX_VERSION}/blackbox_exporter-${BLACKBOX_VERSION}.linux-amd64.tar.gz
        tar xzf blackbox_exporter-${BLACKBOX_VERSION}.linux-amd64.tar.gz
        cp blackbox_exporter-${BLACKBOX_VERSION}.linux-amd64/blackbox_exporter /usr/local/bin/
        rm -rf blackbox_exporter-${BLACKBOX_VERSION}.linux-amd64*
    "
    ok "Blackbox Exporter instalado"

    # ── 8. Upload config files ──
    info "Paso 8: Subiendo archivos de configuracion..."
    ct_upload "${SCRIPT_DIR}/prometheus.yml" "/etc/prometheus/prometheus.yml" "644"
    ct_upload "${SCRIPT_DIR}/blackbox.yml" "/etc/blackbox.yml" "644"
    ok "Configuraciones subidas"

    # ── 9. Create systemd service files ──
    info "Paso 9: Creando servicios systemd..."

    # Prometheus service
    cat > /tmp/prometheus.service << 'SERVICE'
[Unit]
Description=Prometheus Time Series Collection and Processing Server
Wants=network-online.target
After=network-online.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/bin/prometheus \
    --config.file=/etc/prometheus/prometheus.yml \
    --storage.tsdb.path=/var/lib/prometheus \
    --web.console.templates=/etc/prometheus/consoles \
    --web.console.libraries=/etc/prometheus/console_libraries \
    --web.listen-address=:9090 \
    --web.external-url=
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
SERVICE
    scp_proxmox /tmp/prometheus.service "/tmp/prometheus.service"
    ssh_proxmox "pct push ${CT_ID} /tmp/prometheus.service /etc/systemd/system/prometheus.service --perms 644"
    rm -f /tmp/prometheus.service

    # Node Exporter service
    cat > /tmp/node_exporter.service << 'SERVICE'
[Unit]
Description=Prometheus Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter \
    --web.listen-address=:9100
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SERVICE
    scp_proxmox /tmp/node_exporter.service "/tmp/node_exporter.service"
    ssh_proxmox "pct push ${CT_ID} /tmp/node_exporter.service /etc/systemd/system/node_exporter.service --perms 644"
    rm -f /tmp/node_exporter.service

    # Blackbox Exporter service
    cat > /tmp/blackbox_exporter.service << 'SERVICE'
[Unit]
Description=Prometheus Blackbox Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=blackbox
Group=blackbox
Type=simple
ExecStart=/usr/local/bin/blackbox_exporter \
    --config.file=/etc/blackbox.yml \
    --web.listen-address=:9115
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SERVICE
    scp_proxmox /tmp/blackbox_exporter.service "/tmp/blackbox_exporter.service"
    ssh_proxmox "pct push ${CT_ID} /tmp/blackbox_exporter.service /etc/systemd/system/blackbox_exporter.service --perms 644"
    rm -f /tmp/blackbox_exporter.service

    ok "Servicios systemd creados"

    # ── 10. Enable and start services ──
    info "Paso 10: Iniciando servicios..."
    for svc in node_exporter blackbox_exporter prometheus; do
        pct_exec systemctl daemon-reload
        pct_exec systemctl enable "${svc}"
        pct_exec systemctl start "${svc}" || warn "Warning: ${svc} may not have started cleanly"
    done
    ok "Servicios iniciados"

    # ── 11. Install Grafana ──
    info "Paso 11: Instalando Grafana (repo RPM)..."
    pct_exec bash -c '
        cat > /etc/yum.repos.d/grafana.repo << "REPO"
[grafana]
name=grafana
baseurl=https://rpm.grafana.com
repo_gpgcheck=1
enabled=1
gpgcheck=1
gpgkey=https://rpm.grafana.com/gpg.key
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
REPO
        dnf install -y -q grafana
    '
    ok "Grafana instalado"

    # ── 12. Configure Grafana provisioning ──
    info "Paso 12: Configurando Grafana provisioning..."
    pct_exec mkdir -p /etc/grafana/provisioning/datasources
    pct_exec mkdir -p /etc/grafana/provisioning/dashboards
    pct_exec mkdir -p /var/lib/grafana/dashboards
fi

# ── 13. Upload datasource + dashboard (always, even in --only-dashboard mode) ──
info "Paso 13: Subiendo datasource y dashboard de Grafana..."

# Data source YAML
ct_upload "${SCRIPT_DIR}/grafana/datasources.yaml" "/etc/grafana/provisioning/datasources/prometheus.yaml" "644"

# Dashboard provisioning config
cat > /tmp/dashboard-provider.yaml << 'YAML'
apiVersion: 1

providers:
  - name: "Downtime Game"
    orgId: 1
    folder: ""
    type: file
    disableDeletion: true
    editable: true
    options:
      path: /var/lib/grafana/dashboards
YAML
scp_proxmox /tmp/dashboard-provider.yaml "/tmp/dashboard-provider.yaml"
ssh_proxmox "pct push ${CT_ID} /tmp/dashboard-provider.yaml /etc/grafana/provisioning/dashboards/downtime-game.yaml --perms 644"
rm -f /tmp/dashboard-provider.yaml

# Dashboard JSON
ct_upload "${SCRIPT_DIR}/grafana/dashboard.json" "/var/lib/grafana/dashboards/downtime-game-observability.json" "644"

ok "Datasource y dashboard subidos"

# ── 14. Restart Grafana (if not only-dashboard, or if already running) ──
if [ "$ONLY_DASHBOARD" = true ]; then
    info "Reiniciando Grafana para aplicar nuevo dashboard..."
    pct_exec systemctl restart grafana-server || true
else
    info "Paso 14: Iniciando Grafana..."
    pct_exec systemctl enable grafana-server
    pct_exec systemctl start grafana-server
fi

# ── 15. Wait and verify ──
info "Paso 15: Verificando servicios..."
sleep 5

declare -A SERVICES=(
    ["Prometheus"]="localhost:9090/-/ready"
    ["Grafana"]="localhost:3000/api/health"
    ["Node Exporter"]="localhost:9100/metrics"
    ["Blackbox Exporter"]="localhost:9115/health"
)

echo ""
for name in "${!SERVICES[@]}"; do
    url="${SERVICES[$name]}"
    if pct_exec curl -sf "http://${url}" >/dev/null 2>&1; then
        ok "${name}: ✅ http://${CT_IP%%/*}:$(echo $url | cut -d: -f2 | cut -d/ -f1)"
    else
        warn "${name}: ⚠️  No responde en http://${url}"
    fi
done

# ── 16. Summary ──
echo ""
ok "╔═══════════════════════════════════════════════════╗"
ok "║     OBSERVABILITY DEPLOY COMPLETE                 ║"
ok "╚═══════════════════════════════════════════════════╝"
echo ""
info "📡 Accesos:"
echo ""
echo "   Grafana:        http://${CT_IP%%/*}:3000"
echo "   Prometheus:     http://${CT_IP%%/*}:9090"
echo "   Node Exporter:  http://${CT_IP%%/*}:9100/metrics"
echo "   Blackbox:       http://${CT_IP%%/*}:9115/probe"
echo ""
info "🔑 Credenciales Grafana: admin / admin"
echo ""
info "📊 Dashboard importado automaticamente:"
echo "   Downtime Game — Observability"
echo ""
info "Para actualizar solo el dashboard:"
echo "   ./deploy-observability.sh --only-dashboard"
echo ""
