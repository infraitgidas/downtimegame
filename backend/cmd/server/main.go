// Command server is the entry point for the Downtime Game backend.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ema/downtime-game/internal/checker"
	"github.com/ema/downtime-game/internal/game"
	"github.com/ema/downtime-game/internal/hub"
	"github.com/ema/downtime-game/internal/server"
	"github.com/ema/downtime-game/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "downtime-game.db"
	}

	// ── Initialize store (SQLite) ─────────────────────────────────────────
	dataStore, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer dataStore.Close()

	// ── Initialize WebSocket hub ──────────────────────────────────────────
	gameHub := hub.NewHub()
	go gameHub.Run()

	// ── Initialize game engine ────────────────────────────────────────────
	gameEngine := game.NewEngine(dataStore, gameHub)

	// ── Configure executor mode ───────────────────────────────────────────
	// EXECUTOR_MODE=simulated (default) → no-op, for development/testing
	// EXECUTOR_MODE=ssh               → real LXC control via Proxmox SSH
	executor := game.NewExecutorFromEnv()
	gameEngine.SetExecutor(executor)
	log.Printf("Executor configured: %s", executor.Name())

	// ── Health checker ───────────────────────────────────────────────────
	// Polls real LXC /health endpoints to detect service state changes.
	// CHECKER_INTERVAL: how often to poll (default 5s)
	// CHECKER_TIMEOUT:  HTTP request timeout (default 3s)
	checkInterval := getEnvDuration("CHECKER_INTERVAL", 5*time.Second)
	checkTimeout := getEnvDuration("CHECKER_TIMEOUT", 3*time.Second)

	healthChecker := checker.New(
		game.DefaultServices(),
		checkInterval,
		checkTimeout,
		gameEngine.SetServiceStatusFromChecker,
	)
	healthChecker.Start()
	log.Printf("Health checker started (interval=%s, timeout=%s)", checkInterval, checkTimeout)

	// ── Initialize HTTP server ────────────────────────────────────────────
	srv := server.New(gameHub, gameEngine, dataStore)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Backend starting on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down...")

	healthChecker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// getEnvDuration reads an env var as a duration string (e.g. "5s", "3s")
// and returns def if unset or invalid.
func getEnvDuration(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err == nil {
		return time.Duration(n) * time.Second
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("Warning: invalid %s=%q, using default %s", key, val, def)
		return def
	}
	return d
}
