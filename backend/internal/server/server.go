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

	"github.com/ema/downtime-game/internal/game"
	"github.com/ema/downtime-game/internal/hub"
	"github.com/ema/downtime-game/internal/store"
)

// Server wraps the HTTP server, router, WebSocket hub, and game engine.
type Server struct {
	Hub     *hub.Hub
	Engine  *game.Engine
	Router  *chi.Mux
	Store   *store.Store
	upgrader websocket.Upgrader
}

// New creates a new Server with configured routes and middleware.
func New(gameHub *hub.Hub, gameEngine *game.Engine, dataStore *store.Store) *Server {
	s := &Server{
		Hub:    gameHub,
		Engine: gameEngine,
		Store:  dataStore,
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

	// Public endpoints
	s.Router.Get("/health", s.handleHealth)
	s.Router.Get("/ws", s.handleWebSocket)

	// API v1
	s.Router.Route("/api", func(r chi.Router) {
		r.Get("/games", s.handleListGames)
		r.Get("/games/{id}", s.handleGetGame)
		r.Post("/games", s.handleCreateGame)
		r.Post("/games/{id}/start", s.handleStartGame)
		r.Post("/games/{id}/resolve", s.handleResolveIncident)
		r.Post("/games/{id}/abandon", s.handleAbandonGame)
		r.Delete("/games/{id}", s.handleDeleteGame)
		r.Get("/games/{id}/solution", s.handleGameSolution)

		r.Get("/scenarios", s.handleListScenarios)
		r.Get("/services", s.handleListServices)
		r.Get("/services/status", s.handleServiceStatus)
		r.Get("/leaderboard", s.handleLeaderboard)

		// Admin operations
		r.Post("/admin/reset", s.handleAdminReset)
	})

	// Register WebSocket message handlers (admin commands)
	s.registerWSHandlers()

	return s
}

// registerWSHandlers sets up handlers for incoming WebSocket messages.
// These allow admin clients to send game commands via WS instead of REST.
func (s *Server) registerWSHandlers() {
	s.Hub.Router.Handle("create_game", func(client *hub.Client, msg hub.Message) {
		var payload struct {
			PlayerName string `json:"player_name"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("WS create_game: invalid payload: %v", err)
			return
		}
		game, err := s.Engine.CreateGame(payload.PlayerName)
		if err != nil {
			log.Printf("WS create_game error: %v", err)
			return
		}
		log.Printf("WS: game created %s (player: %s)", game.ID, payload.PlayerName)
	})

	s.Hub.Router.Handle("start_game", func(client *hub.Client, msg hub.Message) {
		var payload struct {
			GameID     string `json:"game_id"`
			ScenarioID string `json:"scenario_id,omitempty"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("WS start_game: invalid payload: %v", err)
			return
		}
		inst, err := s.Engine.StartGame(payload.GameID, payload.ScenarioID)
		if err != nil {
			log.Printf("WS start_game error: %v", err)
			return
		}
		log.Printf("WS: game started %s (scenario: %s)", payload.GameID, inst.Scenario.ID)
	})

	s.Hub.Router.Handle("resolve_incident", func(client *hub.Client, msg hub.Message) {
		var payload struct {
			GameID string `json:"game_id"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("WS resolve_incident: invalid payload: %v", err)
			return
		}
		game, err := s.Engine.ResolveIncident(payload.GameID)
		if err != nil {
			log.Printf("WS resolve_incident error: %v", err)
			return
		}
		log.Printf("WS: incident resolved for game %s (score: %d)", payload.GameID, game.Score)
	})

	s.Hub.Router.Handle("abandon_game", func(client *hub.Client, msg hub.Message) {
		var payload struct {
			GameID string `json:"game_id"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("WS abandon_game: invalid payload: %v", err)
			return
		}
		if err := s.Engine.AbandonGame(payload.GameID); err != nil {
			log.Printf("WS abandon_game error: %v", err)
			return
		}
		log.Printf("WS: game abandoned %s", payload.GameID)
	})

	log.Println("WS handlers registered: create_game, start_game, resolve_incident, abandon_game")
}

// ── Health ───────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── WebSocket ────────────────────────────────────────────────────────────────

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

// ── Games ────────────────────────────────────────────────────────────────────

type createGameRequest struct {
	PlayerName string `json:"player_name"`
}

func (s *Server) handleCreateGame(w http.ResponseWriter, r *http.Request) {
	var req createGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.PlayerName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "player_name is required"})
		return
	}

	game, err := s.Engine.CreateGame(req.PlayerName)
	if err != nil {
		log.Printf("Create game error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create game"})
		return
	}

	writeJSON(w, http.StatusCreated, game)
}

func (s *Server) handleListGames(w http.ResponseWriter, r *http.Request) {
	games, err := s.Store.ListGames()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list games"})
		return
	}
	if games == nil {
		games = []*store.Game{}
	}
	writeJSON(w, http.StatusOK, games)
}

func (s *Server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	game, err := s.Store.GetGame(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "game not found"})
		return
	}
	writeJSON(w, http.StatusOK, game)
}

type startGameRequest struct {
	ScenarioID string `json:"scenario_id,omitempty"`
}

func (s *Server) handleStartGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Optional scenario_id in body
	scenarioID := ""
	if r.Body != nil {
		var req startGameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			scenarioID = req.ScenarioID
		}
	}

	inst, err := s.Engine.StartGame(id, scenarioID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"game":       inst.Game,
		"scenario":   inst.Scenario,
		"incident":   inst.Incident,
		"time_limit": inst.Scenario.TimeLimit,
	})
}

func (s *Server) handleResolveIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	game, err := s.Engine.ResolveIncident(id)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, game)
}

func (s *Server) handleAbandonGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.Engine.AbandonGame(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "abandoned"})
}

// ── Scenarios ────────────────────────────────────────────────────────────────

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, game.GetScenarios())
}

// ── Services ─────────────────────────────────────────────────────────────────

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	services := []map[string]any{
		{"id": "sg-rojo", "name": "Rojo", "color": "rojo", "ip": "192.168.1.200", "port": 8080},
		{"id": "sg-azul", "name": "Azul", "color": "azul", "ip": "192.168.1.204", "port": 8084},
		{"id": "sg-verde", "name": "Verde", "color": "verde", "ip": "192.168.1.202", "port": 8082},
		{"id": "sg-amarillo", "name": "Amarillo", "color": "amarillo", "ip": "192.168.1.203", "port": 8083},
	}
	writeJSON(w, http.StatusOK, services)
}

func (s *Server) handleServiceStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.Engine.GetServiceStatuses()
	writeJSON(w, http.StatusOK, statuses)
}

// ── Leaderboard ──────────────────────────────────────────────────────────────

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Store.GetLeaderboard(10)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get leaderboard"})
		return
	}
	if entries == nil {
		entries = []*store.LeaderboardEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// ── Delete Game ──────────────────────────────────────────────────────────────

func (s *Server) handleDeleteGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.Engine.DeleteGame(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── Game Solution ────────────────────────────────────────────────────────────

func (s *Server) handleGameSolution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scenario := s.Engine.GetScenarioForGame(id)
	if scenario == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no active game or scenario not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scenario_id":  scenario.ID,
		"name":         scenario.Name,
		"description":  scenario.Description,
		"fix_hint":     scenario.FixHint,
		"hints":        scenario.Hints,
		"difficulty":   scenario.Difficulty,
		"time_limit":   scenario.TimeLimit,
	})
}

// ── Admin Reset ──────────────────────────────────────────────────────────────

func (s *Server) handleAdminReset(w http.ResponseWriter, r *http.Request) {
	if err := s.Engine.AdminReset(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "reset_completed",
		"services": s.Engine.GetServiceStatuses(),
	})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
