package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ema/downtime-game/internal/game"
	"github.com/ema/downtime-game/internal/hub"
	"github.com/ema/downtime-game/internal/store"
)

// newTestServer creates a server with temp-file store and hub for testing.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	dbPath := t.TempDir() + "/server-test.db"
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	h := hub.NewHub()
	go h.Run()

	e := game.NewEngine(s, h)

	return New(h, e, s)
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestListScenarios(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/scenarios", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var scenarios []map[string]any
	json.NewDecoder(w.Body).Decode(&scenarios)

	if len(scenarios) == 0 {
		t.Fatal("expected at least 1 scenario")
	}

	// Verify scenario structure
	first := scenarios[0]
	if first["id"] == "" {
		t.Error("scenario has empty id")
	}
	if first["name"] == "" {
		t.Error("scenario has empty name")
	}
	if _, ok := first["time_limit"]; !ok {
		t.Error("scenario missing time_limit")
	}
}

func TestListServices(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/services", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var services []map[string]any
	json.NewDecoder(w.Body).Decode(&services)

	if len(services) != 4 {
		t.Errorf("expected 4 services, got %d", len(services))
	}
}

func TestServiceStatus(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/services/status", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var statuses []map[string]any
	json.NewDecoder(w.Body).Decode(&statuses)

	if len(statuses) != 4 {
		t.Errorf("expected 4 services, got %d", len(statuses))
	}

	// All should start online
	for _, s := range statuses {
		if s["online"] != true {
			t.Errorf("service %s should be online", s["service_id"])
		}
	}
}

func TestCreateGame(t *testing.T) {
	srv := newTestServer(t)

	body := `{"player_name":"TestPlayer"}`
	req := httptest.NewRequest("POST", "/api/games", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["player_name"] != "TestPlayer" {
		t.Errorf("player_name = %q, want %q", resp["player_name"], "TestPlayer")
	}
	if resp["status"] != "pending" {
		t.Errorf("status = %q, want %q", resp["status"], "pending")
	}
	if resp["id"] == "" {
		t.Error("id is empty")
	}
}

func TestCreateGame_MissingName(t *testing.T) {
	srv := newTestServer(t)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/games", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListGames(t *testing.T) {
	srv := newTestServer(t)

	// Empty first
	req := httptest.NewRequest("GET", "/api/games", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var games []map[string]any
	json.NewDecoder(w.Body).Decode(&games)
	if len(games) != 0 {
		t.Errorf("expected 0 games, got %d", len(games))
	}

	// Create a game
	srv.Router.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/games", strings.NewReader(`{"player_name":"Alice"}`)),
	)

	// List again
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("GET", "/api/games", nil))

	json.NewDecoder(w.Body).Decode(&games)
	if len(games) != 1 {
		t.Errorf("expected 1 game, got %d", len(games))
	}
}

func TestGameLifecycleAPI(t *testing.T) {
	srv := newTestServer(t)

	// 1. Create game
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w,
		httptest.NewRequest("POST", "/api/games", strings.NewReader(`{"player_name":"E2ETest"}`)),
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("create game: status = %d", w.Code)
	}

	var gameResp map[string]any
	json.NewDecoder(w.Body).Decode(&gameResp)
	gameID := gameResp["id"].(string)

	// 2. Get game by ID
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("GET", "/api/games/"+gameID, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get game: status = %d", w.Code)
	}

	// 3. Start game
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("POST", "/api/games/"+gameID+"/start", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("start game: status = %d, body: %s", w.Code, w.Body.String())
	}

	var startResp map[string]any
	json.NewDecoder(w.Body).Decode(&startResp)
	if startResp["scenario"] == nil {
		t.Fatal("start response missing scenario")
	}

	// 4. Verify service is offline
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("GET", "/api/services/status", nil))

	var statuses []map[string]any
	json.NewDecoder(w.Body).Decode(&statuses)

	scenario := startResp["scenario"].(map[string]any)
	targetSvc := scenario["target_service_id"].(string)

	foundOffline := false
	for _, s := range statuses {
		if s["service_id"] == targetSvc {
			if s["online"] == false && s["incident"] == true {
				foundOffline = true
			}
		}
	}
	if !foundOffline {
		t.Errorf("service %s should be offline with incident after game start", targetSvc)
	}

	// 5. Resolve incident
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("POST", "/api/games/"+gameID+"/resolve", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("resolve: status = %d, body: %s", w.Code, w.Body.String())
	}

	var resolveResp map[string]any
	json.NewDecoder(w.Body).Decode(&resolveResp)
	if resolveResp["status"] != "completed" {
		t.Errorf("status = %q, want %q", resolveResp["status"], "completed")
	}
	if resolveResp["score"].(float64) <= 0 {
		t.Errorf("expected positive score, got %v", resolveResp["score"])
	}

	// 6. Verify leaderboard has entry
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("GET", "/api/leaderboard", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("leaderboard: status = %d", w.Code)
	}

	var entries []map[string]any
	json.NewDecoder(w.Body).Decode(&entries)
	if len(entries) != 1 {
		t.Errorf("expected 1 leaderboard entry, got %d", len(entries))
	}
	if entries[0]["player_name"] != "E2ETest" {
		t.Errorf("player = %q, want %q", entries[0]["player_name"], "E2ETest")
	}
	if entries[0]["score"].(float64) <= 0 {
		t.Errorf("expected positive score, got %v", entries[0]["score"])
	}
}

func TestGameNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/games/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestStartNonexistentGame(t *testing.T) {
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("POST", "/api/games/nonexistent/start", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestResolveNonexistentGame(t *testing.T) {
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("POST", "/api/games/nonexistent/resolve", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAbandonGame(t *testing.T) {
	srv := newTestServer(t)

	// Create and start game
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w,
		httptest.NewRequest("POST", "/api/games", strings.NewReader(`{"player_name":"AbandonTest"}`)),
	)
	var gameResp map[string]any
	json.NewDecoder(w.Body).Decode(&gameResp)
	gameID := gameResp["id"].(string)

	srv.Router.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/games/"+gameID+"/start", nil),
	)

	// Abandon
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("POST", "/api/games/"+gameID+"/abandon", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("abandon: status = %d", w.Code)
	}

	// Verify game is abandoned
	w = httptest.NewRecorder()
	srv.Router.ServeHTTP(w, httptest.NewRequest("GET", "/api/games/"+gameID, nil))

	var game map[string]any
	json.NewDecoder(w.Body).Decode(&game)
	if game["status"] != "abandoned" {
		t.Errorf("status = %q, want %q", game["status"], "abandoned")
	}
}

func TestCORSHeaders(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("OPTIONS", "/api/games", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("missing Access-Control-Allow-Origin header")
	}
}
