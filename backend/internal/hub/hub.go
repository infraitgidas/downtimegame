// Package hub implements a WebSocket hub for the Downtime Game.
// It manages client connections, broadcasting, and room-based messaging.
package hub

import (
	"encoding/json"
	"log"
	"sync"
)

// Message represents a WebSocket message exchanged through the hub.
type Message struct {
	Type     string          `json:"type"`
	ClientID string          `json:"client_id,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

// Hub maintains the set of active clients and provides broadcast/room features.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	rooms      map[string]map[*Client]bool
	Router     *Router
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		rooms:      make(map[string]map[*Client]bool),
		Router:     NewRouter(),
	}
}

// HandleIncoming processes a raw message from a client.
// It first tries to route through registered handlers; if no handler matches,
// it broadcasts the message to all clients (legacy behavior).
func (h *Hub) HandleIncoming(client *Client, data []byte) {
	if h.Router.Route(client, data) {
		return // handled by a registered handler
	}
	// No handler registered — broadcast to all (legacy behavior)
	h.broadcast <- data
}

// Run starts the hub's event loop. Must be called as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			client.SendJSON(Message{
				Type:     "welcome",
				ClientID: client.ID,
			})
			log.Printf("Client connected: %s (%d total)", client.ID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove from all rooms
				for _, room := range h.rooms {
					delete(room, client)
				}
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s (%d total)", client.ID, len(h.clients))

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub (channel-based, non-blocking send).
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub (channel-based, non-blocking send).
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

// BroadcastJSON sends a JSON-serializable message to all clients.
func (h *Hub) BroadcastJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	h.broadcast <- data
	return nil
}

// JoinRoom adds a client to a named room.
func (h *Hub) JoinRoom(client *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][client] = true
}

// BroadcastRoom sends a message to all clients in a room.
func (h *Hub) BroadcastRoom(room string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.rooms[room] {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
