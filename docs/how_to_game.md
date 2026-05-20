Plan de deploy y pruebas
1. En la PC (192.168.1.54)
# Clonar el repo
git clone https://github.com/infraitgidas/downtimegame.git
cd downtimegame
# Opcion A: Modo simulado (sin LXC, para probar que anda)
make deploy-pc
# Opcion B: Modo real (con LXC)
make deploy-pc-ssh
El script deploy-pc.sh se encarga de:
✅ Verificar Docker + Compose
✅ En modo SSH: checkea que exista la SSH key y que llegue a Proxmox
✅ Build y deploy de todos los servicios
✅ Esperar a que el backend responda
✅ Mostrar URLs de acceso
2. URLs después del deploy
Servicio	URL
Dashboard	http://192.168.1.54/ (http://192.168.1.54/)
Admin Panel	http://192.168.1.54/admin/ (http://192.168.1.54/admin/)
API	http://192.168.1.54/api/games (http://192.168.1.54/api/games)
WebSocket	ws://192.168.1.54/ws
3. Pruebas manuales paso a paso
# 1. Verificar que el backend responde
curl http://192.168.1.54/health
# 2. Crear una partida
curl -X POST http://192.168.1.54/api/games \
  -d '{"player_name":"Test"}'
# 3. Listar partidas
curl http://192.168.1.54/api/games
# 4. Iniciar partida (reemplazar {id} con el ID real)
curl -X POST http://192.168.1.54/api/games/{id}/start
# 5. Ver leaderboard
curl http://192.168.1.54/api/leaderboard
# 6. En el browser:
#    - Abrí http://192.168.1.54/ → Dashboard
#    - Abrí http://192.168.1.54/admin/ → Admin
#    - Creá e iniciá una partida desde Admin
#    - El Dashboard muestra la alarma roja en vivo
4. Si probás modo SSH real
Asegurate antes:
# 1. La PC 192.168.1.54 debe llegar a Proxmox
ping 192.168.1.31
# 2. SSH key copiada a Proxmox
ssh-copy-id root@192.168.1.31
# 3. Los LXC deben estar desplegados (desde Proxmox)
#    O ejecutá services/deploy.sh desde Proxmox
# 4. Deploy con SSH
make deploy-pc-ssh
5. Para monitorear
make docker-logs     # Logs de todos los servicios
make docker-down     # Parar todo
docker compose ps    # Estado de servicios