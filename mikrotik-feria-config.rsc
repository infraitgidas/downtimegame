# =============================================================================
# Configuracion MikroTik RB951 - Downtime Game
# RouterOS v6.49.x - Formato exportacion
# =============================================================================
# INSTRUCCIONES:
#   1. System > Reset Configuration (NO保留 Default)
#   2. Files > Upload > subir este archivo
#   3. Terminal > import file=mikrotik-feria-config.rsc
# O directamente pegar todo en Terminal (New Terminal)
# =============================================================================

# ---------------------------------------------------------------------------
# 1. BRIDGE LAN
# ---------------------------------------------------------------------------
/interface bridge
add name=bridge_lan

# Puertos LAN (ether4 y wlan1 quedan fuera: son WAN)
/interface bridge port
add bridge=bridge_lan interface=ether1
add bridge=bridge_lan interface=ether2
add bridge=bridge_lan interface=ether3
add bridge=bridge_lan interface=ether5

# ---------------------------------------------------------------------------
# 2. IP DEL BRIDGE (GATEWAY LAN)
# ---------------------------------------------------------------------------
/ip address
add address=192.168.1.253/24 interface=bridge_lan

# ---------------------------------------------------------------------------
# 3. DHCP
# ---------------------------------------------------------------------------
/ip pool
add name=dhcp-lan ranges=192.168.1.100-192.168.1.200

/ip dhcp-server network
add address=192.168.1.0/24 gateway=192.168.1.253 dns-server=192.168.1.253,8.8.8.8

/ip dhcp-server
add address-pool=dhcp-lan disabled=no interface=bridge_lan name=dhcp-lan

# ---------------------------------------------------------------------------
# 4. RESERVAS DHCP (LEASES ESTATICOS)
# ---------------------------------------------------------------------------
/ip dhcp-server lease
add address=192.168.1.31 mac-address=0C:9D:92:14:A1:25 server=dhcp-lan comment="Proxmox VE"
add address=192.168.1.54 mac-address=1C:1B:0D:CB:19:FD server=dhcp-lan comment="PC Rocky Linux"

# ---------------------------------------------------------------------------
# 5. WAN PRIMARIA: ether4 DHCP (RED LABORATORIO)
# ---------------------------------------------------------------------------
/ip dhcp-client
add disabled=no interface=ether4

# ---------------------------------------------------------------------------
# 6. WAN SECUNDARIA: wlan1 CLIENTE WifiDoc
# ---------------------------------------------------------------------------
/interface wireless security-profiles
add authentication-types=wpa2-psk mode=dynamic-keys name=wifidoc-station \
    wpa2-pre-shared-key="{{ WIFIDOC_PASSWORD }}"

/interface wireless
set [ find default-name=wlan1 ] disabled=yes

/interface wireless
set [ find default-name=wlan1 ] mode=station ssid="WifiDoc" \
    security-profile=wifidoc-station disabled=no

/ip dhcp-client
add disabled=no interface=wlan1

# ---------------------------------------------------------------------------
# 7. VIRTUAL AP: SSID "gidas"
# ---------------------------------------------------------------------------
/interface wireless security-profiles
add authentication-types=wpa2-psk mode=dynamic-keys name=gidas-ap \
    wpa2-pre-shared-key=gidas2026

/interface wireless
add master-interface=wlan1 mode=ap-bridge name=wlan1-ap security-profile=gidas-ap \
    ssid=gidas disabled=no

# Agregar Virtual AP al bridge LAN
/interface bridge port
add bridge=bridge_lan interface=wlan1-ap

# ---------------------------------------------------------------------------
# 8. NAT MASQUERADE
# ---------------------------------------------------------------------------
/ip firewall nat
add action=masquerade chain=srcnat out-interface=ether4
add action=masquerade chain=srcnat out-interface=wlan1

# ---------------------------------------------------------------------------
# 9. DNS
# ---------------------------------------------------------------------------
/ip dns
set allow-remote-requests=yes servers=8.8.8.8,1.1.1.1

# ---------------------------------------------------------------------------
# 10. FIREWALL
# ---------------------------------------------------------------------------
/ip firewall filter
add action=accept chain=input comment="Allow established" \
    connection-state=established,related
add action=drop chain=input comment="Drop invalid" \
    connection-state=invalid
add action=accept chain=input comment="Allow from LAN" \
    in-interface=bridge_lan
add action=accept chain=input comment="Allow from WiFi" \
    in-interface=wlan1-ap
add action=accept chain=input comment="Allow ICMP" protocol=icmp
add action=drop chain=input comment="Drop other input"

add action=accept chain=forward comment="Allow established forward" \
    connection-state=established,related
add action=drop chain=forward comment="Drop invalid forward" \
    connection-state=invalid
add action=accept chain=forward comment="Allow LAN to WAN" \
    in-interface=bridge_lan
add action=accept chain=forward comment="Allow WiFi to WAN" \
    in-interface=wlan1-ap
add action=drop chain=forward comment="Drop other forward"

# ---------------------------------------------------------------------------
# 11. RUTAS CON FAILOVER
# ---------------------------------------------------------------------------
/ip route
add check-gateway=ping comment="WAN primaria - ether4" distance=1 gateway=ether4
add check-gateway=ping comment="WAN backup - wlan1" distance=2 gateway=wlan1

# ---------------------------------------------------------------------------
# 12. SERVICIOS
# ---------------------------------------------------------------------------
/ip service
set ssh disabled=no
set telnet disabled=no
set www disabled=no
set api disabled=no
set api-ssl disabled=no

# ---------------------------------------------------------------------------
# FIN
# ---------------------------------------------------------------------------
