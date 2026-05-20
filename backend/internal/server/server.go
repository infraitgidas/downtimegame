// Package server provides the HTTP server and routing for the Downtime Game.
package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"

	"github.com/ema/downtime-game/internal/hub"
)

// Server wraps the HTTP server, router, and WebSocket hub.
type Server struct {
	Hub     *hub.Hub
	Router  *chi.Mux
	upgrader websocket.Upgrader
}

// New creates a new Server with configured routes and middleware.
func New(gameHub *hub.Hub) *Server {
	s := &Server{
		Hub:    gameHub,
		Router: chi.NewRouter(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}

	s.Router.Use(chimw.Logger)
	s.Router.Use(chimw.Recoverer)
	s.Router.Use(chimw.RequestID)
	s.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s.Router.Get("/health", s.handleHealth)
	s.Router.Get("/ws", s.handleWebSocket)
	s.Router.Route("/api", func(r chi.Router) {
		r.Get("/games", s.handleListGames)
		r.Get("/services", s.handleListServices)
	})

	return s
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleWebSocket upgrades HTTP to WebSocket and registers the client.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := hub.NewClient(s.Hub, conn)
	s.Hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

// handleListGames returns the list of games (stub).
func (s *Server) handleListGames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

// handleListServices returns the list of game services.
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	services := []map[string]interface{}{
		{"id": "sg-rojo", "name": "Rojo", "color": "rojo", "ip": "192.168.1.200", "port": 8080},
		{"id": "sg-azul", "name": "Azul", "color": "azul", "ip": "192.168.1.204", "port": 8084},
		{"id": "sg-verde", "name": "Verde", "color": "verde", "ip": "192.168.1.202", "port": 8082},
		{"id": "sg-amarillo", "name": "Amarillo", "color": "amarillo", "ip": "192.168.1.203", "port": 8083},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}
