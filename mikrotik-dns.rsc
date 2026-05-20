# =============================================================================
# DNS Static Records - Downtime Game
# Ejecutar EN el MikroTik (despues de aplicar configuracion)
# =============================================================================
# USO:
#   Terminal MikroTik > import file=mikrotik-dns.rsc
# =============================================================================

# Registrar los 4 servicios por nombre
/ip dns static
add address=192.168.1.200 name=rojo.downtime.game comment="SG Rojo"
add address=192.168.1.201 name=azul.downtime.game comment="SG Azul"
add address=192.168.1.202 name=verde.downtime.game comment="SG Verde"
add address=192.168.1.203 name=amarillo.downtime.game comment="SG Amarillo"

# Alias cortos (sin dominio)
add address=192.168.1.200 name=rojo comment="SG Rojo (short)"
add address=192.168.1.201 name=azul comment="SG Azul (short)"
add address=192.168.1.202 name=verde comment="SG Verde (short)"
add address=192.168.1.203 name=amarillo comment="SG Amarillo (short)"

# Servidores de infraestructura
add address=192.168.1.31 name=proxmox.downtime.game comment="Proxmox VE"
add address=192.168.1.54 name=pcrocky.downtime.game comment="PC Rocky Linux"
add address=192.168.1.253 name=router.downtime.game comment="MikroTik Router"

# Alias cortos
add address=192.168.1.31 name=proxmox comment="Proxmox (short)"
add address=192.168.1.54 name=pcrocky comment="PC Rocky (short)"
add address=192.168.1.253 name=router comment="MikroTik (short)"

:put "Registros DNS agregados exitosamente"
:put "Proba con: ping rojo.downtime.game"
