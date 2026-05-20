package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Game represents a single play session.
type Game struct {
	ID         string     `json:"id"`
	PlayerName string     `json:"player_name"`
	Status     string     `json:"status"`
	Score      int        `json:"score"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateGame inserts a new game and returns it.
func (s *Store) CreateGame(id, playerName string) (*Game, error) {
	now := time.Now().UTC()
	_, err := s.DB.Exec(
		`INSERT INTO games (id, player_name, status, created_at) VALUES (?, ?, 'pending', ?)`,
		id, playerName, now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, err
	}
	return &Game{
		ID:         id,
		PlayerName: playerName,
		Status:     "pending",
		Score:      0,
		CreatedAt:  now,
	}, nil
}

// GetGame retrieves a game by ID.
func (s *Store) GetGame(id string) (*Game, error) {
	row := s.DB.QueryRow(
		`SELECT id, player_name, status, score, started_at, ended_at, created_at FROM games WHERE id = ?`,
		id,
	)
	g := &Game{}
	var createdAt, startedAt, endedAt sql.NullString
	if err := row.Scan(&g.ID, &g.PlayerName, &g.Status, &g.Score, &startedAt, &endedAt, &createdAt); err != nil {
		return nil, err
	}
	if createdAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, createdAt.String)
		g.CreatedAt = t
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, startedAt.String)
		g.StartedAt = &t
	}
	if endedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, endedAt.String)
		g.EndedAt = &t
	}
	return g, nil
}

// ListGames returns all games ordered by creation time descending.
func (s *Store) ListGames() ([]*Game, error) {
	rows, err := s.DB.Query(
		`SELECT id, player_name, status, score, started_at, ended_at, created_at FROM games ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*Game
	for rows.Next() {
		g := &Game{}
		var createdAt, startedAt, endedAt sql.NullString
		if err := rows.Scan(&g.ID, &g.PlayerName, &g.Status, &g.Score, &startedAt, &endedAt, &createdAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, createdAt.String)
			g.CreatedAt = t
		}
		if startedAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, startedAt.String)
			g.StartedAt = &t
		}
		if endedAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, endedAt.String)
			g.EndedAt = &t
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

// UpdateGameStatus changes a game's status. Valid transitions:
// pending -> active, active -> completed, active -> abandoned
func (s *Store) UpdateGameStatus(id, status string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var query string
	switch status {
	case "active":
		query = `UPDATE games SET status = ?, started_at = ? WHERE id = ?`
	case "completed":
		query = `UPDATE games SET status = ?, score = ?, ended_at = ? WHERE id = ?`
	case "abandoned":
		query = `UPDATE games SET status = ?, ended_at = ? WHERE id = ?`
	default:
		query = `UPDATE games SET status = ? WHERE id = ?`
	}

	var err error
	switch status {
	case "completed":
		game, err := s.GetGame(id)
		if err != nil {
			return err
		}
		_, err = s.DB.Exec(query, status, game.Score, now, id)
	case "active", "abandoned":
		_, err = s.DB.Exec(query, status, now, id)
	default:
		_, err = s.DB.Exec(query, status, id)
	}
	return err
}

// UpdateGameScore updates the score for a game.
func (s *Store) UpdateGameScore(id string, score int) error {
	_, err := s.DB.Exec(`UPDATE games SET score = ? WHERE id = ?`, score, id)
	return err
}

// DeleteGame removes a game and its incidents from the database.
// Only allowed for non-active games.
func (s *Store) DeleteGame(id string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check game exists and is not active
	var status string
	err = tx.QueryRow(`SELECT status FROM games WHERE id = ?`, id).Scan(&status)
	if err != nil {
		return err
	}
	if status == "active" {
		return fmt.Errorf("cannot delete active game: %s", id)
	}

	// Delete incidents first (FK constraint)
	if _, err := tx.Exec(`DELETE FROM incidents WHERE game_id = ?`, id); err != nil {
		return err
	}
	// Delete game
	if _, err := tx.Exec(`DELETE FROM games WHERE id = ?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// ListActiveGames returns all games with status 'active'.
func (s *Store) ListActiveGames() ([]*Game, error) {
	rows, err := s.DB.Query(
		`SELECT id, player_name, status, score, started_at, ended_at, created_at FROM games WHERE status = 'active' ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*Game
	for rows.Next() {
		g := &Game{}
		var createdAt, startedAt, endedAt sql.NullString
		if err := rows.Scan(&g.ID, &g.PlayerName, &g.Status, &g.Score, &startedAt, &endedAt, &createdAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, createdAt.String)
			g.CreatedAt = t
		}
		if startedAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, startedAt.String)
			g.StartedAt = &t
		}
		if endedAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, endedAt.String)
			g.EndedAt = &t
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

// Incident represents a downtime incident within a game.
type Incident struct {
	ID          string     `json:"id"`
	GameID      string     `json:"game_id"`
	ScenarioID  string     `json:"scenario_id"`
	ServiceID   string     `json:"service_id"`
	Status      string     `json:"status"`
	TriggeredAt time.Time  `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// CreateIncident inserts a new incident.
func (s *Store) CreateIncident(id, gameID, scenarioID, serviceID string) (*Incident, error) {
	now := time.Now().UTC()
	_, err := s.DB.Exec(
		`INSERT INTO incidents (id, game_id, scenario_id, service_id, status, triggered_at) VALUES (?, ?, ?, ?, 'active', ?)`,
		id, gameID, scenarioID, serviceID, now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, err
	}
	return &Incident{
		ID:          id,
		GameID:      gameID,
		ScenarioID:  scenarioID,
		ServiceID:   serviceID,
		Status:      "active",
		TriggeredAt: now,
	}, nil
}

// ResolveIncident marks an incident as resolved.
func (s *Store) ResolveIncident(id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.DB.Exec(
		`UPDATE incidents SET status = 'resolved', resolved_at = ? WHERE id = ?`,
		now, id,
	)
	return err
}

// GetActiveIncident returns the active (unresolved) incident for a game, if any.
func (s *Store) GetActiveIncident(gameID string) (*Incident, error) {
	row := s.DB.QueryRow(
		`SELECT id, game_id, scenario_id, service_id, status, triggered_at, resolved_at
		 FROM incidents WHERE game_id = ? AND status = 'active' ORDER BY triggered_at DESC LIMIT 1`,
		gameID,
	)
	inc := &Incident{}
	var triggeredAt, resolvedAt sql.NullString
	if err := row.Scan(&inc.ID, &inc.GameID, &inc.ScenarioID, &inc.ServiceID, &inc.Status, &triggeredAt, &resolvedAt); err != nil {
		return nil, err
	}
	if triggeredAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, triggeredAt.String)
		inc.TriggeredAt = t
	}
	if resolvedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, resolvedAt.String)
		inc.ResolvedAt = &t
	}
	return inc, nil
}

// GetLatestIncidentForGame returns the most recent incident for a game,
// regardless of status (active, resolved, timeout). Returns nil if not found.
func (s *Store) GetLatestIncidentForGame(gameID string) (*Incident, error) {
	row := s.DB.QueryRow(
		`SELECT id, game_id, scenario_id, service_id, status, triggered_at, resolved_at
		 FROM incidents WHERE game_id = ? ORDER BY triggered_at DESC LIMIT 1`,
		gameID,
	)
	inc := &Incident{}
	var triggeredAt, resolvedAt sql.NullString
	if err := row.Scan(&inc.ID, &inc.GameID, &inc.ScenarioID, &inc.ServiceID, &inc.Status, &triggeredAt, &resolvedAt); err != nil {
		return nil, err
	}
	if triggeredAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, triggeredAt.String)
		inc.TriggeredAt = t
	}
	if resolvedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, resolvedAt.String)
		inc.ResolvedAt = &t
	}
	return inc, nil
}

// LeaderboardEntry represents a completed game in the leaderboard.
type LeaderboardEntry struct {
	ID                 int       `json:"id"`
	PlayerName         string    `json:"player_name"`
	Score              int       `json:"score"`
	ResolvedInSeconds  int       `json:"resolved_in_seconds"`
	CompletedAt        time.Time `json:"completed_at"`
}

// AddLeaderboardEntry inserts a completed game into the leaderboard.
func (s *Store) AddLeaderboardEntry(playerName string, score, seconds int) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.DB.Exec(
		`INSERT INTO leaderboard (player_name, score, resolved_in_seconds, completed_at) VALUES (?, ?, ?, ?)`,
		playerName, score, seconds, now,
	)
	return err
}

// GetLeaderboard returns the top entries ordered by score descending.
func (s *Store) GetLeaderboard(limit int) ([]*LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.DB.Query(
		`SELECT id, player_name, score, resolved_in_seconds, completed_at
		 FROM leaderboard ORDER BY score DESC, resolved_in_seconds ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*LeaderboardEntry
	for rows.Next() {
		e := &LeaderboardEntry{}
		var completedAt sql.NullString
		if err := rows.Scan(&e.ID, &e.PlayerName, &e.Score, &e.ResolvedInSeconds, &completedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			t, _ := time.Parse(time.RFC3339Nano, completedAt.String)
			e.CompletedAt = t
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
