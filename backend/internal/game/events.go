package game

// Status constants for Game and Incident lifecycle.
const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusAbandoned = "abandoned"
	StatusResolved  = "resolved"
	StatusTimeout   = "timeout"
)

// Event type constants for WebSocket messages.
// These are sent from the server to clients (dashboard, admin).
const (
	// Game lifecycle
	EventGameCreated   = "game_created"
	EventGameStarted   = "game_started"
	EventGameCompleted = "game_completed"
	EventGameAbandoned = "game_abandoned"
	EventGameTimeout   = "game_timeout"

	// Incidents
	EventIncidentActive   = "incident_active"
	EventIncidentResolved = "incident_resolved"

	// Timer
	EventTimerTick = "timer_tick"

	// Services
	EventServiceStatus = "service_status"
)
