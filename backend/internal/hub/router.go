// Package hub provides WebSocket hub, client management, and message routing.
package hub

import "encoding/json"

// MessageHandler processes an inbound message from a client.
type MessageHandler func(client *Client, msg Message)

// Router routes incoming WebSocket messages to registered handlers by message type.
type Router struct {
	handlers map[string]MessageHandler
}

// NewRouter creates a new message router.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]MessageHandler),
	}
}

// Handle registers a handler for a given message type.
func (r *Router) Handle(msgType string, handler MessageHandler) {
	r.handlers[msgType] = handler
}

// Route dispatches a message to its registered handler.
// Returns true if a handler was found and called.
func (r *Router) Route(client *Client, data []byte) bool {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return false
	}

	handler, ok := r.handlers[msg.Type]
	if !ok {
		return false
	}

	handler(client, msg)
	return true
}
