package game

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ema/downtime-game/internal/hub"
	"github.com/ema/downtime-game/internal/store"
)

// Engine manages the game lifecycle, scenario selection, and incident triggering.
type Engine struct {
	mu             sync.RWMutex
	store          *store.Store
	hub            *hub.Hub
	executor       Executor
	serviceStatus  map[string]*ServiceStatus // serviceID -> current status
	activeGames    map[string]*GameInstance  // gameID -> game instance
}

// NewEngine creates a new game engine with the given store and hub.
// Uses a SimulatedExecutor by default; call SetExecutor to change.
func NewEngine(s *store.Store, h *hub.Hub) *Engine {
	e := &Engine{
		store:         s,
		hub:           h,
		serviceStatus: defaultServiceStatuses(),
		activeGames:   make(map[string]*GameInstance),
	}
	e.executor = NewSimulatedExecutor()
	return e
}

// SetExecutor changes the incident executor (e.g., to SSHExecutor in production).
func (e *Engine) SetExecutor(exec Executor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executor = exec
	log.Printf("Engine: executor set to %s", exec.Name())
}

// Executor returns the current executor.
func (e *Engine) Executor() Executor {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.executor
}

// defaultServiceStatuses initializes all 4 services as online.
func defaultServiceStatuses() map[string]*ServiceStatus {
	return map[string]*ServiceStatus{
		"sg-rojo":     {ServiceID: "sg-rojo", Online: true},
		"sg-azul":     {ServiceID: "sg-azul", Online: true},
		"sg-verde":    {ServiceID: "sg-verde", Online: true},
		"sg-amarillo": {ServiceID: "sg-amarillo", Online: true},
	}
}

// GetServiceStatuses returns the current status of all services.
func (e *Engine) GetServiceStatuses() []ServiceStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	statuses := make([]ServiceStatus, 0, len(e.serviceStatus))
	for _, s := range e.serviceStatus {
		statuses = append(statuses, *s)
	}
	return statuses
}

// GetServiceStatus returns the status of a single service.
func (e *Engine) GetServiceStatus(serviceID string) *ServiceStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.serviceStatus[serviceID]
}

// setServiceStatus updates the in-memory service status WITHOUT locking.
// The caller MUST hold e.mu write lock.
func (e *Engine) setServiceStatus(serviceID string, online bool, incident bool) {
	if e.serviceStatus[serviceID] != nil {
		e.serviceStatus[serviceID].Online = online
		e.serviceStatus[serviceID].Incident = incident
	}

	e.broadcastEvent(EventServiceStatus, map[string]any{
		"service_id": serviceID,
		"online":     online,
		"incident":   incident,
	})
}

// SetServiceStatusFromChecker is called by the health checker (external goroutine)
// when a real service transitions between online/offline.
// It acquires the write lock internally.
func (e *Engine) SetServiceStatusFromChecker(serviceID string, online bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.serviceStatus[serviceID] != nil {
		e.serviceStatus[serviceID].Online = online
		// Don't clear incident flag — that's game-managed
	}

	e.broadcastEvent(EventServiceStatus, map[string]any{
		"service_id": serviceID,
		"online":     online,
		"incident":   e.serviceStatus[serviceID] != nil && e.serviceStatus[serviceID].Incident,
	})
}

// CreateGame creates a new game in pending state.
func (e *Engine) CreateGame(playerName string) (*store.Game, error) {
	id := uuid.New().String()
	game, err := e.store.CreateGame(id, playerName)
	if err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}

	// Broadcast game_created event
	e.broadcastEvent(EventGameCreated, map[string]any{
		"game": game,
	})

	log.Printf("Game created: %s (player: %s)", game.ID, playerName)
	return game, nil
}

// StartGame begins a game by selecting a random scenario and triggering an incident.
func (e *Engine) StartGame(gameID string) (*GameInstance, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check game exists
	game, err := e.store.GetGame(gameID)
	if err != nil {
		return nil, fmt.Errorf("game not found: %w", err)
	}
	if game.Status != StatusPending {
		return nil, fmt.Errorf("game %s is not pending (status: %s)", gameID, game.Status)
	}

	// Pick a random scenario
	scenario := e.pickRandomScenario()
	if scenario == nil {
		return nil, fmt.Errorf("no scenarios available")
	}

	// Activate game
	if err := e.store.UpdateGameStatus(gameID, StatusActive); err != nil {
		return nil, fmt.Errorf("activate game: %w", err)
	}

	// Create incident
	incident, err := e.store.CreateIncident(
		uuid.New().String(),
		gameID,
		scenario.ID,
		scenario.TargetServiceID,
	)
	if err != nil {
		return nil, fmt.Errorf("create incident: %w", err)
	}

	// Update in-memory game
	now := time.Now()
	game.Status = StatusActive
	game.StartedAt = &now

	// Mark service as offline/incident
	if e.serviceStatus[scenario.TargetServiceID] != nil {
		e.serviceStatus[scenario.TargetServiceID].Online = false
		e.serviceStatus[scenario.TargetServiceID].Incident = true
	}

	// Execute the physical failure (simulated or real)
	if err := e.executor.Trigger(scenario); err != nil {
		// Log but continue — the logical game state is already updated
		log.Printf("Executor trigger error: %v", err)
	}

	inst := &GameInstance{
		Game:      game,
		Incident:  incident,
		Scenario:  scenario,
		StartTime: now,
		TimerDone: make(chan struct{}),
	}
	e.activeGames[gameID] = inst

	// Start the timer goroutine with the instance reference
	go e.runTimer(inst, scenario.TimeLimit)

	// Broadcast events
	e.broadcastEvent(EventGameStarted, map[string]any{
		"game":          game,
		"scenario":      scenario,
		"incident":      incident,
		"time_limit":    scenario.TimeLimit,
	})

	// Also broadcast the service status update
	e.broadcastEvent(EventServiceStatus, map[string]any{
		"service_id": scenario.TargetServiceID,
		"online":     false,
		"incident":   true,
	})

	log.Printf("Game started: %s | Scenario: %s | Service: %s | Time limit: %ds",
		gameID, scenario.ID, scenario.TargetServiceID, scenario.TimeLimit)

	return inst, nil
}

// ResolveIncident marks an active incident as resolved and completes the game.
func (e *Engine) ResolveIncident(gameID string) (*store.Game, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	inst, ok := e.activeGames[gameID]
	if !ok {
		return nil, fmt.Errorf("no active game instance: %s", gameID)
	}
	if inst.Incident.Status != StatusActive {
		return nil, fmt.Errorf("incident %s is not active", inst.Incident.ID)
	}

	// Calculate score
	elapsed := time.Since(inst.StartTime).Seconds()
	score := calculateScore(inst.Scenario.Difficulty, inst.Scenario.TimeLimit, int(elapsed))

	// Update store
	if err := e.store.ResolveIncident(inst.Incident.ID); err != nil {
		return nil, fmt.Errorf("resolve incident: %w", err)
	}
	if err := e.store.UpdateGameScore(gameID, score); err != nil {
		return nil, fmt.Errorf("update score: %w", err)
	}
	if err := e.store.UpdateGameStatus(gameID, StatusCompleted); err != nil {
		return nil, fmt.Errorf("complete game: %w", err)
	}
	if err := e.store.AddLeaderboardEntry(inst.Game.PlayerName, score, int(elapsed)); err != nil {
		return nil, fmt.Errorf("add leaderboard entry: %w", err)
	}

	// Update in-memory state
	now := time.Now()
	inst.Incident.Status = StatusResolved
	inst.Incident.ResolvedAt = &now
	inst.Game.Status = StatusCompleted
	inst.Game.Score = score
	inst.Game.EndedAt = &now

	// Restore service online
	if e.serviceStatus[inst.Scenario.TargetServiceID] != nil {
		e.serviceStatus[inst.Scenario.TargetServiceID].Online = true
		e.serviceStatus[inst.Scenario.TargetServiceID].Incident = false
	}

	// Execute the physical restoration (simulated or real)
	if err := e.executor.Resolve(inst.Scenario); err != nil {
		log.Printf("Executor resolve error: %v", err)
	}

	// Stop timer
	close(inst.TimerDone)

	// Clean up active game
	delete(e.activeGames, gameID)

	// Broadcast events
	e.broadcastEvent(EventIncidentResolved, map[string]any{
		"game":          inst.Game,
		"incident":      inst.Incident,
		"scenario":      inst.Scenario,
		"score":         score,
		"elapsed_seconds": elapsed,
	})

	e.broadcastEvent(EventGameCompleted, map[string]any{
		"game":   inst.Game,
		"score":  score,
		"player": inst.Game.PlayerName,
	})

	e.broadcastEvent(EventServiceStatus, map[string]any{
		"service_id": inst.Scenario.TargetServiceID,
		"online":     true,
		"incident":   false,
	})

	log.Printf("Incident resolved: game=%s player=%s score=%d elapsed=%.0fs",
		gameID, inst.Game.PlayerName, score, elapsed)

	return inst.Game, nil
}

// AbandonGame cancels an active game (e.g., admin forces stop).
func (e *Engine) AbandonGame(gameID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	inst, ok := e.activeGames[gameID]
	if !ok {
		return fmt.Errorf("no active game: %s", gameID)
	}

	if err := e.store.UpdateGameStatus(gameID, StatusAbandoned); err != nil {
		return fmt.Errorf("abandon game: %w", err)
	}

	// Restore service
	if e.serviceStatus[inst.Scenario.TargetServiceID] != nil {
		e.serviceStatus[inst.Scenario.TargetServiceID].Online = true
		e.serviceStatus[inst.Scenario.TargetServiceID].Incident = false
	}

	close(inst.TimerDone)
	delete(e.activeGames, gameID)

	e.broadcastEvent(EventGameAbandoned, map[string]any{
		"game": inst.Game,
	})

	log.Printf("Game abandoned: %s", gameID)
	return nil
}

// pickRandomScenario selects a random scenario from the catalog.
func (e *Engine) pickRandomScenario() *Scenario {
	all := GetScenarios()
	if len(all) == 0 {
		return nil
	}
	return &all[rand.Intn(len(all))]
}

// runTimer counts down the time limit and triggers timeout if expired.
func (e *Engine) runTimer(inst *GameInstance, timeLimit int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	elapsed := 0
	for {
		select {
		case <-ticker.C:
			elapsed++
			remaining := timeLimit - elapsed

			// Broadcast timer tick every 5 seconds
			if remaining%5 == 0 || remaining <= 10 {
				e.broadcastEvent(EventTimerTick, map[string]any{
					"game_id":   inst.Game.ID,
					"elapsed":   elapsed,
					"remaining": remaining,
				})
			}

			// Timeout
			if elapsed >= timeLimit {
				e.handleTimeout(inst.Game.ID)
				return
			}

		case <-inst.TimerDone:
			return
		}
	}
}

// handleTimeout marks the game as abandoned due to timeout.
func (e *Engine) handleTimeout(gameID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	inst, ok := e.activeGames[gameID]
	if !ok {
		return
	}

	if err := e.store.UpdateGameStatus(gameID, StatusAbandoned); err != nil {
		log.Printf("Timeout: failed to abandon game %s: %v", gameID, err)
		return
	}

	// Restore service
	if e.serviceStatus[inst.Scenario.TargetServiceID] != nil {
		e.serviceStatus[inst.Scenario.TargetServiceID].Online = true
		e.serviceStatus[inst.Scenario.TargetServiceID].Incident = false
	}

	inst.Incident.Status = StatusTimeout
	delete(e.activeGames, gameID)

	e.broadcastEvent(EventGameTimeout, map[string]any{
		"game": inst.Game,
	})

	log.Printf("Game timed out: %s (limit: %ds)", gameID, inst.Scenario.TimeLimit)
}

// broadcastEvent sends a typed event through the WebSocket hub.
func (e *Engine) broadcastEvent(eventType string, payload map[string]any) {
	msg := hub.Message{
		Type:    eventType,
		Payload: toRawMessage(payload),
	}
	if err := e.hub.BroadcastJSON(msg); err != nil {
		log.Printf("Broadcast event %s: %v", eventType, err)
	}
}

// calculateScore computes the score for a resolved incident.
// Formula: base (1000) + time_bonus + difficulty_bonus
func calculateScore(difficulty, timeLimit, elapsed int) int {
	base := 1000

	// Time bonus: faster resolution = more points
	timeBonus := 0
	if elapsed < timeLimit {
		timeBonus = (timeLimit - elapsed) * 10
	}

	// Difficulty bonus: harder scenarios = more points
	difficultyBonus := difficulty * 100

	score := base + timeBonus + difficultyBonus
	if score < 100 {
		score = 100
	}

	return score
}

// toRawMessage converts a value to json.RawMessage.
func toRawMessage(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}
