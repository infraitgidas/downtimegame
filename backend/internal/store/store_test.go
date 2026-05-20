package store

import (
	"os"
	"testing"
	"time"
)

// newTestStore creates a temporary SQLite store for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := t.TempDir() + "/test.db"
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateGame(t *testing.T) {
	s := newTestStore(t)

	game, err := s.CreateGame("test-id-1", "Alice")
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}

	if game.ID != "test-id-1" {
		t.Errorf("ID = %q, want %q", game.ID, "test-id-1")
	}
	if game.PlayerName != "Alice" {
		t.Errorf("PlayerName = %q, want %q", game.PlayerName, "Alice")
	}
	if game.Status != "pending" {
		t.Errorf("Status = %q, want %q", game.Status, "pending")
	}
	if game.Score != 0 {
		t.Errorf("Score = %d, want 0", game.Score)
	}
	if game.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, expected a timestamp")
	}
}

func TestGetGameNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetGame("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent game, got nil")
	}
}

func TestGetGame(t *testing.T) {
	s := newTestStore(t)

	created, err := s.CreateGame("test-id-2", "Bob")
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}

	got, err := s.GetGame("test-id-2")
	if err != nil {
		t.Fatalf("GetGame failed: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.PlayerName != created.PlayerName {
		t.Errorf("PlayerName = %q, want %q", got.PlayerName, created.PlayerName)
	}
}

func TestListGames(t *testing.T) {
	s := newTestStore(t)

	// Empty list
	games, err := s.ListGames()
	if err != nil {
		t.Fatalf("ListGames failed: %v", err)
	}
	if len(games) != 0 {
		t.Errorf("expected 0 games, got %d", len(games))
	}

	// Create 2 games with different timestamps
	s.CreateGame("id-1", "Alice")
	time.Sleep(10 * time.Millisecond) // ensure different timestamp
	s.CreateGame("id-2", "Bob")

	games, err = s.ListGames()
	if err != nil {
		t.Fatalf("ListGames failed: %v", err)
	}
	if len(games) != 2 {
		t.Errorf("expected 2 games, got %d", len(games))
	}

	// Most recent first
	if games[0].PlayerName != "Bob" {
		t.Errorf("expected Bob first (most recent), got %s", games[0].PlayerName)
	}
}

func TestUpdateGameStatus(t *testing.T) {
	s := newTestStore(t)
	s.CreateGame("id-3", "Charlie")

	// Pending → Active
	if err := s.UpdateGameStatus("id-3", "active"); err != nil {
		t.Fatalf("UpdateGameStatus to active failed: %v", err)
	}

	game, _ := s.GetGame("id-3")
	if game.Status != "active" {
		t.Errorf("Status = %q, want %q", game.Status, "active")
	}
	if game.StartedAt == nil {
		t.Error("StartedAt is nil, expected a timestamp")
	}

	// Active → Completed
	s.UpdateGameScore("id-3", 2500)
	if err := s.UpdateGameStatus("id-3", "completed"); err != nil {
		t.Fatalf("UpdateGameStatus to completed failed: %v", err)
	}

	game, _ = s.GetGame("id-3")
	if game.Status != "completed" {
		t.Errorf("Status = %q, want %q", game.Status, "completed")
	}
	if game.EndedAt == nil {
		t.Error("EndedAt is nil, expected a timestamp")
	}
}

func TestUpdateGameScore(t *testing.T) {
	s := newTestStore(t)
	s.CreateGame("id-4", "Diana")

	if err := s.UpdateGameScore("id-4", 3000); err != nil {
		t.Fatalf("UpdateGameScore failed: %v", err)
	}

	game, _ := s.GetGame("id-4")
	if game.Score != 3000 {
		t.Errorf("Score = %d, want 3000", game.Score)
	}
}

func TestCreateIncident(t *testing.T) {
	s := newTestStore(t)
	s.CreateGame("id-5", "Eve")

	incident, err := s.CreateIncident("inc-1", "id-5", "scenario-1", "sg-rojo")
	if err != nil {
		t.Fatalf("CreateIncident failed: %v", err)
	}

	if incident.ID != "inc-1" {
		t.Errorf("ID = %q, want %q", incident.ID, "inc-1")
	}
	if incident.GameID != "id-5" {
		t.Errorf("GameID = %q, want %q", incident.GameID, "id-5")
	}
	if incident.ScenarioID != "scenario-1" {
		t.Errorf("ScenarioID = %q, want %q", incident.ScenarioID, "scenario-1")
	}
	if incident.ServiceID != "sg-rojo" {
		t.Errorf("ServiceID = %q, want %q", incident.ServiceID, "sg-rojo")
	}
	if incident.Status != "active" {
		t.Errorf("Status = %q, want %q", incident.Status, "active")
	}
}

func TestResolveIncident(t *testing.T) {
	s := newTestStore(t)
	s.CreateGame("id-6", "Frank")
	s.CreateIncident("inc-2", "id-6", "scenario-2", "sg-azul")

	if err := s.ResolveIncident("inc-2"); err != nil {
		t.Fatalf("ResolveIncident failed: %v", err)
	}

	incident, err := s.GetActiveIncident("id-6")
	if err == nil {
		t.Fatalf("expected error for resolved incident, got incident %+v", incident)
	}
}

func TestGetActiveIncident(t *testing.T) {
	s := newTestStore(t)
	s.CreateGame("id-7", "Grace")

	// No active incident
	_, err := s.GetActiveIncident("id-7")
	if err == nil {
		t.Fatal("expected error for game with no incidents")
	}

	// Create and find active incident
	s.CreateIncident("inc-3", "id-7", "scenario-3", "sg-verde")
	incident, err := s.GetActiveIncident("id-7")
	if err != nil {
		t.Fatalf("GetActiveIncident failed: %v", err)
	}
	if incident.ID != "inc-3" {
		t.Errorf("ID = %q, want %q", incident.ID, "inc-3")
	}

	// Resolve and verify not active
	s.ResolveIncident("inc-3")
	_, err = s.GetActiveIncident("id-7")
	if err == nil {
		t.Fatal("expected error after resolving incident")
	}
}

func TestLeaderboard(t *testing.T) {
	s := newTestStore(t)

	// Empty leaderboard
	entries, err := s.GetLeaderboard(10)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}

	// Add entries
	s.AddLeaderboardEntry("Alice", 2500, 45)
	s.AddLeaderboardEntry("Bob", 3200, 30)
	s.AddLeaderboardEntry("Charlie", 1800, 90)

	// Verify top entries
	entries, err = s.GetLeaderboard(2)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].PlayerName != "Bob" {
		t.Errorf("expected Bob first (highest score), got %s", entries[0].PlayerName)
	}
	if entries[0].Score != 3200 {
		t.Errorf("expected score 3200, got %d", entries[0].Score)
	}

	// Verify all entries
	entries, err = s.GetLeaderboard(10)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Verify timestamps
	if entries[0].CompletedAt.IsZero() {
		t.Error("CompletedAt is zero, expected a timestamp")
	}
}

func TestFullGameLifecycle(t *testing.T) {
	s := newTestStore(t)

	// 1. Create game
	game, err := s.CreateGame("lifecycle-1", "TestPlayer")
	if err != nil {
		t.Fatalf("CreateGame failed: %v", err)
	}

	// 2. Activate game
	if err := s.UpdateGameStatus(game.ID, "active"); err != nil {
		t.Fatalf("UpdateGameStatus failed: %v", err)
	}

	// 3. Create incident
	incident, err := s.CreateIncident("lifecycle-inc-1", game.ID, "scenario-crash", "sg-rojo")
	if err != nil {
		t.Fatalf("CreateIncident failed: %v", err)
	}
	if incident.Status != "active" {
		t.Errorf("expected active incident, got %s", incident.Status)
	}

	// 4. Verify active incident
	active, err := s.GetActiveIncident(game.ID)
	if err != nil {
		t.Fatalf("GetActiveIncident failed: %v", err)
	}
	if active.ID != incident.ID {
		t.Errorf("incident ID = %q, want %q", active.ID, incident.ID)
	}

	// 5. Simulate time passing for score calculation
	time.Sleep(10 * time.Millisecond)

	// 6. Resolve incident
	if err := s.ResolveIncident(incident.ID); err != nil {
		t.Fatalf("ResolveIncident failed: %v", err)
	}

	// 7. Update score and complete game
	if err := s.UpdateGameScore(game.ID, 2500); err != nil {
		t.Fatalf("UpdateGameScore failed: %v", err)
	}
	if err := s.UpdateGameStatus(game.ID, "completed"); err != nil {
		t.Fatalf("UpdateGameStatus failed: %v", err)
	}

	// 8. Add to leaderboard
	if err := s.AddLeaderboardEntry(game.PlayerName, 2500, 45); err != nil {
		t.Fatalf("AddLeaderboardEntry failed: %v", err)
	}

	// 9. Verify final state
	finalGame, err := s.GetGame(game.ID)
	if err != nil {
		t.Fatalf("GetGame failed: %v", err)
	}
	if finalGame.Status != "completed" {
		t.Errorf("final status = %q, want %q", finalGame.Status, "completed")
	}
	if finalGame.Score != 2500 {
		t.Errorf("final score = %d, want 2500", finalGame.Score)
	}

	// 10. Verify no active incident
	_, err = s.GetActiveIncident(game.ID)
	if err == nil {
		t.Fatal("expected no active incident after resolution")
	}

	// 11. Verify leaderboard
	entries, err := s.GetLeaderboard(10)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 leaderboard entry, got %d", len(entries))
	}
	if entries[0].PlayerName != "TestPlayer" {
		t.Errorf("leaderboard player = %q, want %q", entries[0].PlayerName, "TestPlayer")
	}
}

// Test that the database file is created when not using :memory:
func TestStoreFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer s.Close()

	// Verify file exists
	_, err = os.Stat(dbPath)
	if err != nil {
		t.Errorf("db file not created: %v", err)
	}

	// Verify we can create and read data
	s.CreateGame("file-test-1", "FileTest")
	game, err := s.GetGame("file-test-1")
	if err != nil {
		t.Fatalf("GetGame failed on file-backed store: %v", err)
	}
	if game.PlayerName != "FileTest" {
		t.Errorf("PlayerName = %q, want %q", game.PlayerName, "FileTest")
	}
}
