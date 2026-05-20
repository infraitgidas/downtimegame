package store

// migrations contains all SQL schema migrations, run in order on startup.
// Append new migrations at the end — never modify existing ones.
var migrations = []string{
	// 001: Core tables — games, incidents, leaderboard
	`CREATE TABLE IF NOT EXISTS games (
		id TEXT PRIMARY KEY,
		player_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending'
			CHECK(status IN ('pending', 'active', 'completed', 'abandoned')),
		score INTEGER NOT NULL DEFAULT 0,
		started_at TEXT,
		ended_at TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`,

	`CREATE TABLE IF NOT EXISTS incidents (
		id TEXT PRIMARY KEY,
		game_id TEXT NOT NULL REFERENCES games(id),
		scenario_id TEXT NOT NULL,
		service_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active'
			CHECK(status IN ('active', 'resolved', 'timeout')),
		triggered_at TEXT NOT NULL DEFAULT (datetime('now')),
		resolved_at TEXT
	);`,

	`CREATE TABLE IF NOT EXISTS leaderboard (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		player_name TEXT NOT NULL,
		score INTEGER NOT NULL,
		resolved_in_seconds INTEGER NOT NULL,
		completed_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`,
}
