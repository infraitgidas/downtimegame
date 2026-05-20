# nov/18/2025 14:19:06 by RouterOS 6.49.19
# software id = LLBU-WG2H
#
# model = 951Ui-2HnD
# serial number = 558304337A4F
/interface bridge
add name=bridge-lan
/interface wireless security-profiles
set [ find default=yes ] supplicant-identity=MikroTik
add authentication-types=wpa2-psk mode=dynamic-keys name=Mi-Seguridad \
    supplicant-identity=MikroTik wpa2-pre-shared-key="{{ ROUTER_WIFI_PASSWORD }}"
/interface wireless
set [ find default-name=wlan1 ] band=2ghz-onlyn channel-width=20/40mhz-Ce \
    disabled=no frequency=auto mode=ap-bridge security-profile=Mi-Seguridad \
    ssid=networking
/ip pool
add name=dhcp_pool ranges=192.168.88.10-192.168.88.254
/ip dhcp-server
add address-pool=dhcp_pool disabled=no interface=bridge-lan name=dhcp-lan
/queue type
add kind=sfq name=SFQ-Trafico
/interface bridge port
add bridge=bridge-lan interface=ether2
add bridge=bridge-lan interface=ether3
add bridge=bridge-lan interface=ether4
add bridge=bridge-lan interface=ether5
add bridge=bridge-lan interface=wlan1
/ip address
add address=192.168.88.1/24 interface=bridge-lan network=192.168.88.0
/ip dhcp-client
add disabled=no interface=ether1
/ip dhcp-server lease
add address=192.168.88.250 comment="IP Fija para 080027FD1F23" mac-address=\
    08:00:27:FD:1F:23 server=dhcp-lan
add address=192.168.88.251 comment="IP Fija para 0800277E0B90" mac-address=\
    08:00:27:7E:0B:90 server=dhcp-lan
/ip dhcp-server network
add address=192.168.88.0/24 dns-server=8.8.8.8 gateway=192.168.88.1
/ip dns
set allow-remote-requests=yes servers=8.8.8.8,8.8.4.4
/ip firewall filter
add action=fasttrack-connection chain=forward comment=FastTrack \
    connection-state=established,related
add action=accept chain=forward comment="Aceptar trafico nuevo" \
    connection-state=new
/ip firewall nat
add action=masquerade chain=srcnat out-interface=ether1
/system clock
set time-zone-name=America/Argentina/Buenos_Aires
