// Command server is the entry point for the Downtime Game backend.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
