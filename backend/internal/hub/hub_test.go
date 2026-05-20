package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHubNew(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub() returned nil")
	}
	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", h.ClientCount())
	}
}

func TestHubRegisterUnregister(t *testing.T) {
	h := NewHub()
	go h.Run()

	// Create a test server with WebSocket upgrade
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		client := NewClient(h, conn)
		h.register <- client
		go client.ReadPump()
		go client.WritePump()
	}))
	defer server.Close()

	// Connect a client
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	// Read welcome message
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}

	var welcome Message
	if err := json.Unmarshal(msg, &welcome); err != nil {
		t.Fatal(err)
	}
	if welcome.Type != "welcome" {
		t.Errorf("expected type 'welcome', got '%s'", welcome.Type)
	}
	if welcome.ClientID == "" {
		t.Error("expected non-empty client_id")
	}

	// Wait a bit for registration
	time.Sleep(50 * time.Millisecond)
	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.ClientCount())
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	go h.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		client := NewClient(h, conn)
		h.register <- client
		go client.ReadPump()
		go client.WritePump()
	}))
	defer server.Close()

	// Connect 3 clients
	clients := make([]*websocket.Conn, 3)
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	for i := range clients {
		ws, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer ws.Close()
		clients[i] = ws
		// Consume welcome message
		_, _, err = ws.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(50 * time.Millisecond)
	if h.ClientCount() != 3 {
		t.Fatalf("expected 3 clients, got %d", h.ClientCount())
	}

	// Broadcast a message
	testMsg := []byte(`{"type":"test","payload":"hello"}`)
	h.Broadcast(testMsg)

	// All 3 clients should receive it
	for i, ws := range clients {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("client %d: read error: %v", i, err)
		}
		if string(msg) != string(testMsg) {
			t.Errorf("client %d: expected %s, got %s", i, testMsg, msg)
		}
	}
}

func TestHubBroadcastJSON(t *testing.T) {
	h := NewHub()
	go h.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		client := NewClient(h, conn)
		h.register <- client
		go client.ReadPump()
		go client.WritePump()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	// Consume welcome
	_, _, _ = ws.ReadMessage()

	time.Sleep(50 * time.Millisecond)

	// Broadcast JSON
	err = h.BroadcastJSON(Message{Type: "ping"})
	if err != nil {
		t.Fatal(err)
	}

	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(msg), `"type":"ping"`) {
		t.Errorf("expected ping message, got %s", msg)
	}
}

func TestHubClientCount(t *testing.T) {
	h := NewHub()
	if h.ClientCount() != 0 {
		t.Errorf("expected 0, got %d", h.ClientCount())
	}
}
