package game

import (
	"testing"

	"github.com/ema/downtime-game/internal/hub"
	"github.com/ema/downtime-game/internal/store"
)

// newTestEngine creates an engine with temp-file store and hub for testing.
func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	dbPath := t.TempDir() + "/engine-test.db"
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	h := hub.NewHub()
	go h.Run()

	return NewEngine(s, h)
}

func TestGetScenarios(t *testing.T) {
	scenarios := GetScenarios()
	if len(scenarios) == 0 {
		t.Fatal("expected at least 1 scenario, got 0")
	}

	// Verify all scenarios have required fields
	for _, s := range scenarios {
		if s.ID == "" {
			t.Error("scenario has empty ID")
		}
		if s.Name == "" {
			t.Errorf("scenario %s has empty Name", s.ID)
		}
		if s.TargetServiceID == "" {
			t.Errorf("scenario %s has empty TargetServiceID", s.ID)
		}
		if s.TimeLimit <= 0 {
			t.Errorf("scenario %s has TimeLimit=%d, want >0", s.ID, s.TimeLimit)
		}
		if s.Difficulty < 1 || s.Difficulty > 5 {
			t.Errorf("scenario %s has Difficulty=%d, want 1-5", s.ID, s.Difficulty)
		}
	}
}

func TestGetScenarioByID(t *testing.T) {
	s := GetScenarioByID("svc-crash-rojo")
	if s == nil {
		t.Fatal("expected to find scenario svc-crash-rojo")
	}
	if s.Name != "Servicio Rojo caído" {
		t.Errorf("Name = %q, want %q", s.Name, "Servicio Rojo caído")
	}

	// Non-existent
	if GetScenarioByID("nonexistent") != nil {
		t.Error("expected nil for nonexistent scenario")
	}
}

func TestGetScenarioForService(t *testing.T) {
	s := GetScenarioForService("sg-rojo")
	if s == nil {
		t.Fatal("expected scenario for sg-rojo")
	}
	if s.TargetServiceID != "sg-rojo" {
		t.Errorf("TargetServiceID = %q, want %q", s.TargetServiceID, "sg-rojo")
	}

	// Non-existent service
	if GetScenarioForService("sg-nonexistent") != nil {
		t.Error("expected nil for nonexistent service")
	}
}

func TestCreateGame(t *testing.T) {
	e := newTestEngine(t)

	game, err := e.CreateGame("TestPlayer")
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}

	if game.PlayerName != "TestPlayer" {
		t.Errorf("PlayerName = %q, want %q", game.PlayerName, "TestPlayer")
	}
	if game.Status != StatusPending {
		t.Errorf("Status = %q, want %q", game.Status, StatusPending)
	}
	if game.ID == "" {
		t.Error("ID is empty")
	}
}

func TestStartGame(t *testing.T) {
	e := newTestEngine(t)

	game, _ := e.CreateGame("TestPlayer")

	inst, err := e.StartGame(game.ID, "")
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	if inst.Game.Status != StatusActive {
		t.Errorf("game status = %q, want %q", inst.Game.Status, StatusActive)
	}
	if inst.Incident.Status != StatusActive {
		t.Errorf("incident status = %q, want %q", inst.Incident.Status, StatusActive)
	}
	if inst.Scenario == nil {
		t.Fatal("scenario is nil")
	}
	if inst.Scenario.ID == "" {
		t.Error("scenario ID is empty")
	}
	if inst.StartTime.IsZero() {
		t.Error("StartTime is zero")
	}

	// Verify service status changed
	svcStatus := e.GetServiceStatus(inst.Scenario.TargetServiceID)
	if svcStatus == nil {
		t.Fatalf("service status for %s not found", inst.Scenario.TargetServiceID)
	}
	if svcStatus.Online {
		t.Errorf("service %s should be offline during incident", inst.Scenario.TargetServiceID)
	}
	if !svcStatus.Incident {
		t.Errorf("service %s should have incident=true", inst.Scenario.TargetServiceID)
	}
}

func TestStartGame_NotFound(t *testing.T) {
	e := newTestEngine(t)

	_, err := e.StartGame("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for nonexistent game, got nil")
	}
}

func TestStartGame_NotPending(t *testing.T) {
	e := newTestEngine(t)

	game, _ := e.CreateGame("TestPlayer")
	e.StartGame(game.ID, "")

	// Try starting again
	_, err := e.StartGame(game.ID, "")
	if err == nil {
		t.Fatal("expected error for already-started game, got nil")
	}
}

func TestResolveIncident(t *testing.T) {
	e := newTestEngine(t)

	game, _ := e.CreateGame("TestPlayer")
	e.StartGame(game.ID, "")

	resolvedGame, err := e.ResolveIncident(game.ID)
	if err != nil {
		t.Fatalf("ResolveIncident failed: %v", err)
	}

	if resolvedGame.Status != StatusCompleted {
		t.Errorf("status = %q, want %q", resolvedGame.Status, StatusCompleted)
	}
	if resolvedGame.Score <= 0 {
		t.Errorf("expected positive score, got %d", resolvedGame.Score)
	}

	// Verify service restored
	svcStatus := e.GetServiceStatuses()
	var allOnline = true
	for _, s := range svcStatus {
		if !s.Online {
			allOnline = false
			t.Errorf("service %s should be online after resolution", s.ServiceID)
		}
		if s.Incident {
			t.Errorf("service %s should have incident=false after resolution", s.ServiceID)
		}
	}
	if !allOnline {
		t.Error("not all services are online after resolution")
	}
}

func TestResolveIncident_NoActiveGame(t *testing.T) {
	e := newTestEngine(t)

	_, err := e.ResolveIncident("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent game, got nil")
	}
}

func TestResolveIncident_AlreadyResolved(t *testing.T) {
	e := newTestEngine(t)

	game, _ := e.CreateGame("TestPlayer")
	e.StartGame(game.ID, "")
	e.ResolveIncident(game.ID)

	// Try resolving again
	_, err := e.ResolveIncident(game.ID)
	if err == nil {
		t.Fatal("expected error for already resolved game, got nil")
	}
}

func TestAbandonGame(t *testing.T) {
	e := newTestEngine(t)

	game, _ := e.CreateGame("TestPlayer")
	e.StartGame(game.ID, "")

	if err := e.AbandonGame(game.ID); err != nil {
		t.Fatalf("AbandonGame failed: %v", err)
	}

	// Verify service restored
	svcStatus := e.GetServiceStatuses()
	for _, s := range svcStatus {
		if !s.Online {
			t.Errorf("service %s should be online after abandon", s.ServiceID)
		}
	}
}

func TestAbandonGame_NoActiveGame(t *testing.T) {
	e := newTestEngine(t)

	if err := e.AbandonGame("nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent game, got nil")
	}
}

func TestGetServiceStatuses(t *testing.T) {
	e := newTestEngine(t)

	statuses := e.GetServiceStatuses()
	if len(statuses) != 4 {
		t.Errorf("expected 4 services, got %d", len(statuses))
	}

	for _, s := range statuses {
		if s.ServiceID == "" {
			t.Error("service has empty ID")
		}
		if !s.Online {
			t.Errorf("service %s should start online", s.ServiceID)
		}
		if s.Incident {
			t.Errorf("service %s should start without incident", s.ServiceID)
		}
	}
}

func TestGetServiceStatus(t *testing.T) {
	e := newTestEngine(t)

	s := e.GetServiceStatus("sg-rojo")
	if s == nil {
		t.Fatal("expected service status for sg-rojo")
	}
	if !s.Online {
		t.Error("sg-rojo should start online")
	}

	// Non-existent
	if e.GetServiceStatus("sg-nonexistent") != nil {
		t.Error("expected nil for nonexistent service")
	}
}

func TestScoring(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int
		timeLimit  int
		elapsed    int
		expected   int
	}{
		{
			name:       "fast resolution hard scenario",
			difficulty: 5,
			timeLimit:  180,
			elapsed:    10,
			expected:   1000 + (180-10)*10 + 5*100, // 1000 + 1700 + 500 = 3200
		},
		{
			name:       "slow resolution easy scenario",
			difficulty: 1,
			timeLimit:  120,
			elapsed:    110,
			expected:   1000 + (120-110)*10 + 1*100, // 1000 + 100 + 100 = 1200
		},
		{
			name:       "timeout scenario (elapsed >= timeLimit)",
			difficulty: 3,
			timeLimit:  60,
			elapsed:    120,
			expected:   1000 + 3*100, // base + difficulty, no time bonus = 1300
		},
		{
			name:       "exact time limit",
			difficulty: 2,
			timeLimit:  60,
			elapsed:    60,
			expected:   1000 + 2*100, // base + difficulty, no time bonus = 1200
		},
		{
			name:       "always at least base + difficulty",
			difficulty: 1,
			timeLimit:  10,
			elapsed:    100,
			expected:   1000 + 1*100, // 1100 (difficulty always counts)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateScore(tt.difficulty, tt.timeLimit, tt.elapsed)
			if score != tt.expected {
				t.Errorf("score = %d, want %d", score, tt.expected)
			}
		})
	}
}

func TestDefaultServices(t *testing.T) {
	services := DefaultServices()
	if len(services) != 4 {
		t.Errorf("expected 4 services, got %d", len(services))
	}

	expected := map[string]string{
		"sg-rojo":     "192.168.1.200",
		"sg-azul":     "192.168.1.204",
		"sg-verde":    "192.168.1.202",
		"sg-amarillo": "192.168.1.203",
	}

	for _, s := range services {
		ip, ok := expected[s.ID]
		if !ok {
			t.Errorf("unexpected service ID: %s", s.ID)
			continue
		}
		if s.IP != ip {
			t.Errorf("service %s: IP = %s, want %s", s.ID, s.IP, ip)
		}
		if s.Port <= 0 {
			t.Errorf("service %s: Port = %d, want >0", s.ID, s.Port)
		}
		if s.HealthURL() == "" {
			t.Errorf("service %s: empty HealthURL", s.ID)
		}
	}
}

func TestGameServicesConfig(t *testing.T) {
	e := newTestEngine(t)
	_ = e // engine initializes with default service statuses

	statuses := e.GetServiceStatuses()
	svcMap := make(map[string]bool)
	for _, s := range statuses {
		if svcMap[s.ServiceID] {
			t.Errorf("duplicate service ID: %s", s.ServiceID)
		}
		svcMap[s.ServiceID] = true
	}

	// Verify all 4 expected services exist
	for _, id := range []string{"sg-rojo", "sg-azul", "sg-verde", "sg-amarillo"} {
		if !svcMap[id] {
			t.Errorf("missing service: %s", id)
		}
	}
}
