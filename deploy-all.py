#!/usr/bin/env python3
"""
Deploy services to Proxmox CTs via SSH + pct
Connects to Proxmox (192.168.1.31), uploads files, deploys to CTs
"""

import pexpect
import sys
import os
import time

PROXMOX_HOST = "192.168.1.31"
PROXMOX_USER = "root"
PROXMOX_PASS = os.environ.get("PROXMOX_PASS", "default-change-me")

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
SERVICE_FILE = os.path.join(SCRIPT_DIR, "services/http-service.py")
DEPLOY_FILE = os.path.join(SCRIPT_DIR, "services/deploy.sh")

COLORS = {
    "rojo": {"vmid": 200, "ip": "192.168.1.200", "port": 8080},
    "azul": {"vmid": 201, "ip": "192.168.1.201", "port": 8081},
    "verde": {"vmid": 202, "ip": "192.168.1.202", "port": 8082},
    "amarillo": {"vmid": 203, "ip": "192.168.1.203", "port": 8083},
}


PROMPT = r"root@pve-ad:~#"

def ssh_run(commands, timeout=30):
    """Run commands on Proxmox via SSH using pexpect."""
    child = pexpect.spawn(
        f"ssh -q -o StrictHostKeyChecking=no -o ConnectTimeout=10 {PROXMOX_USER}@{PROXMOX_HOST}",
        encoding="utf-8",
        timeout=timeout,
    )

    results = []
    i = child.expect(["password:", pexpect.EOF, pexpect.TIMEOUT], timeout=10)
    if i == 0:
        child.sendline(PROXMOX_PASS)
        i = child.expect([PROMPT, pexpect.EOF, pexpect.TIMEOUT], timeout=10)
        if i != 0:
            print("❌ Login failed")
            return results
    else:
        print("❌ SSH connection failed")
        return results

    # Set custom PS1 to avoid matching # in MOTD
    child.sendline("export PS1='ROOT-READY> '")
    i = child.expect(["ROOT-READY>", pexpect.EOF, pexpect.TIMEOUT], timeout=10)
    if i != 0:
        print("⚠️  Could not set custom prompt, continuing anyway")

    # Clear any buffered output
    child.expect([".+", pexpect.TIMEOUT], timeout=0.5)

    for cmd in commands:
        child.sendline(cmd)
        # Wait for command output + prompt
        i = child.expect(["ROOT-READY>", pexpect.EOF, pexpect.TIMEOUT], timeout=timeout)
        output = child.before or ""
        # Filter out the echoed command itself
        lines = []
        for line in output.strip().split("\n"):
            stripped = line.strip()
            if stripped and cmd.strip() not in stripped and "ROOT-READY" not in stripped:
                lines.append(stripped)
        results.append((cmd, "\n".join(lines)))

        # Print compact output
        summary = "\n".join(lines)
        if summary.strip():
            for l in lines[:5]:
                print(f"    {l}")
            if len(lines) > 5:
                print(f"    ... ({len(lines) - 5} more lines)")

    child.sendline("exit")
    child.close()
    return results


def scp_upload(local_path, remote_path):
    """Upload a file to Proxmox via SCP using pexpect."""
    print(f"  📄 Subiendo {os.path.basename(local_path)} → {remote_path}")
    child = pexpect.spawn(
        f"scp -o StrictHostKeyChecking=no -o ConnectTimeout=10 {local_path} {PROXMOX_USER}@{PROXMOX_HOST}:{remote_path}",
        encoding="utf-8",
        timeout=30,
    )
    i = child.expect(["password:", pexpect.EOF, pexpect.TIMEOUT], timeout=10)
    if i == 0:
        child.sendline(PROXMOX_PASS)
        child.expect(pexpect.EOF, timeout=30)
    output = child.before
    child.close()
    if "Permission denied" in str(output):
        print(f"    ❌ Permission denied")
        return False
    print(f"    ✅ OK")
    return True


def check_ct(vmid):
    """Check if a CT exists and is running."""
    results = ssh_run([f"pct status {vmid}"])
    for cmd, output in results:
        if "running" in output:
            return True, "running"
        elif "stopped" in output:
            return True, "stopped"
    return False, "not_found"


def main():
    print("=" * 60)
    print("  🚀 Deploy Servicios HTTP a Proxmox CTs")
    print("=" * 60)
    print()

    # ── Step 1: Check Proxmox connectivity ──
    print("📡 Paso 1: Conectando a Proxmox...")
    results = ssh_run([
        "hostname",
        "cat /etc/pve/version 2>/dev/null || echo 'not-proxmox'",
    ], timeout=15)
    print()

    if not results:
        print("❌ No se pudo conectar a Proxmox")
        sys.exit(1)

    # ── Step 2: Check current CT status ──
    print("🔍 Paso 2: Verificando CTs...")
    results = ssh_run([
        "pct list",
    ], timeout=15)
    print()

    # ── Step 3: Upload service files ──
    print("📤 Paso 3: Subiendo archivos...")
    scp_upload(SERVICE_FILE, "/root/http-service.py")
    scp_upload(DEPLOY_FILE, "/root/deploy.sh")
    print()

    # ── Step 4: Create deploy script on Proxmox ──
    # Since deploy.sh references files relative to itself, we need to adjust paths
    print("⚙️  Paso 4: Preparando deploy script en Proxmox...")

    procmox_deploy = f"""#!/bin/bash
set -euo pipefail

SERVICE_SCRIPT=/root/http-service.py

declare -A CT_VMID=(
    [rojo]=200 [azul]=201 [verde]=202 [amarillo]=203
)
declare -A CT_IP=(
    [rojo]="192.168.1.200" [azul]="192.168.1.201" [verde]="192.168.1.202" [amarillo]="192.168.1.203"
)
declare -A CT_PORT=(
    [rojo]=8080 [azul]=8081 [verde]=8082 [amarillo]=8083
)

for color in rojo azul verde amarillo; do
    vmid=${{CT_VMID[$color]}}
    ip=${{CT_IP[$color]}}
    port=${{CT_PORT[$color]}}

    echo ""
    echo "===== Desplegando $color (CT $vmid - $ip:$port) ====="

    # Start CT if stopped
    status=$(pct status $vmid 2>/dev/null | awk '{{print $2}}')
    if [ "$status" != "running" ]; then
        echo "  ▶ Iniciando CT $vmid..."
        pct start $vmid
        sleep 3
    fi

    # Copy service script
    echo "  📄 Copiando http-service.py..."
    pct push $vmid "$SERVICE_SCRIPT" /opt/http-service.py --perms 755

    # Create systemd service
    UNIT_NAME=sg-$color
    cat > /tmp/$UNIT_NAME.service << SYSTEMDEOF
[Unit]
Description=Downtime Game - Servicio $color
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=nobody
Group=nobody
Environment=SERVICE_COLOR=$color
Environment=SERVICE_PORT=$port
Environment=SERVICE_ID=sg-$color
ExecStart=/usr/bin/python3 /opt/http-service.py
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
SYSTEMDEOF

    pct push $vmid /tmp/$UNIT_NAME.service /etc/systemd/system/$UNIT_NAME.service --perms 644

    # Stop old service
    echo "  🔄 Deteniendo servicio anterior..."
    pct exec $vmid -- systemctl stop $UNIT_NAME 2>/dev/null || true
    pct exec $vmid -- systemctl disable $UNIT_NAME 2>/dev/null || true

    # Start new service
    echo "  🚀 Iniciando nuevo servicio..."
    pct exec $vmid -- systemctl daemon-reload
    pct exec $vmid -- systemctl enable $UNIT_NAME
    pct exec $vmid -- systemctl restart $UNIT_NAME

    sleep 2

    # Verify
    if pct exec $vmid -- systemctl is-active $UNIT_NAME 2>/dev/null; then
        echo "  ✅ Servicio $color activo"
    else
        echo "  ❌ Servicio $color NO arrancó"
        pct exec $vmid -- systemctl status $UNIT_NAME --no-pager 2>&1 || true
        continue
    fi

    # Test endpoint
    echo "  🔍 Verificando /health..."
    sleep 1
    health=$(pct exec $vmid -- curl -sf http://127.0.0.1:$port/health 2>/dev/null) && {{
        echo "  ✅ Health: $health"
    }} || {{
        echo "  ❌ Health check FAILED"
    }}

    rm -f /tmp/$UNIT_NAME.service
done

echo ""
echo "===== DEPLOY COMPLETADO ====="
"""

    # Upload the Proxmox-specific deploy script
    with open("/tmp/deploy-proxmox.sh", "w") as f:
        f.write(procmox_deploy)

    scp_upload("/tmp/deploy-proxmox.sh", "/root/deploy-proxmox.sh")
    os.unlink("/tmp/deploy-proxmox.sh")
    print()

    # ── Step 5: Run deploy ──
    print("🚀 Paso 5: Ejecutando deploy en Proxmox...")
    print()
    results = ssh_run([
        "chmod +x /root/deploy-proxmox.sh",
    ], timeout=10)

    # Run the deploy script (this takes a while)
    print("   (Esto puede tomar ~30 segundos...)")
    print()
    results = ssh_run([
        "bash /root/deploy-proxmox.sh",
    ], timeout=120)

    # ── Step 6: Verify ──
    print("✅ Paso 6: Verificando servicios...")
    print()
    for color, info in COLORS.items():
        results = ssh_run([
            f"pct exec {info['vmid']} -- curl -sf http://127.0.0.1:{info['port']}/health 2>/dev/null || echo 'FAIL'",
        ], timeout=10)
        for cmd, output in results:
            if "FAIL" in output:
                print(f"  ❌ {color}: FAILED")
            elif '"status": "ok"' in output:
                print(f"  ✅ {color}: OK")
            else:
                print(f"  ⚠️  {color}: {output.strip()[:80]}")

    print()
    print("=" * 60)
    print("  ✅ Deploy completado!")
    print("=" * 60)

    # Cleanup remote files
    print()
    print("🧹 Limpiando archivos temporales en Proxmox...")
    ssh_run([
        "rm -f /root/http-service.py /root/deploy.sh /root/deploy-proxmox.sh",
    ], timeout=10)


if __name__ == "__main__":
    main()
