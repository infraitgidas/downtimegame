package game

import (
	"time"

	"github.com/ema/downtime-game/internal/store"
)

// Game types are used from the store package to avoid duplication.
// store.Game        → game lifecycle (id, player, status, score, timestamps)
// store.Incident    → downtime incident (scenario, service, timestamps)
// store.LeaderboardEntry → leaderboard results

// Scenario defines a downtime scenario that can occur in the game.
// This is defined here because it's game-domain logic, not persisted data.
type Scenario struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	TargetServiceID string   `json:"target_service_id"`
	FailureType     string   `json:"failure_type"`
	Severity        string   `json:"severity"`
	Difficulty      int      `json:"difficulty"`
	Hints           []string `json:"hints"`
	FixHint         string   `json:"fix_hint"`
	TimeLimit       int      `json:"time_limit"`
}

// ServiceStatus represents the current runtime status of a game service.
type ServiceStatus struct {
	ServiceID string `json:"service_id"`
	Online    bool   `json:"online"`
	Incident  bool   `json:"incident"`
}

// GameInstance holds the runtime state of an active game.
type GameInstance struct {
	Game      *store.Game
	Incident  *store.Incident
	Scenario  *Scenario
	StartTime time.Time
	TimerDone chan struct{}
}
