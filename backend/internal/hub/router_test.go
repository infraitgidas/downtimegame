package hub

import (
	"encoding/json"
	"testing"
)

func TestRouter_HandleAndRoute(t *testing.T) {
	r := NewRouter()
	handled := false

	r.Handle("test_message", func(client *Client, msg Message) {
		handled = true
		if msg.Type != "test_message" {
			t.Errorf("msg.Type = %q, want %q", msg.Type, "test_message")
		}
	})

	data, _ := json.Marshal(Message{Type: "test_message"})

	// Route should find handler
	if !r.Route(nil, data) {
		t.Error("expected Route to return true")
	}
	if !handled {
		t.Error("handler was not called")
	}
}

func TestRouter_NoHandler(t *testing.T) {
	r := NewRouter()
	data, _ := json.Marshal(Message{Type: "unknown_type"})

	if r.Route(nil, data) {
		t.Error("expected Route to return false for unregistered type")
	}
}

func TestRouter_InvalidJSON(t *testing.T) {
	r := NewRouter()
	r.Handle("test", func(client *Client, msg Message) {
		t.Error("handler should not be called for invalid JSON")
	})

	if r.Route(nil, []byte("{invalid")) {
		t.Error("expected Route to return false for invalid JSON")
	}
}

func TestRouter_MultipleHandlers(t *testing.T) {
	r := NewRouter()
	var calls []string

	r.Handle("type_a", func(client *Client, msg Message) {
		calls = append(calls, "a")
	})
	r.Handle("type_b", func(client *Client, msg Message) {
		calls = append(calls, "b")
	})

	// Route type_a
	r.Route(nil, mustJSON(t, Message{Type: "type_a"}))
	if len(calls) != 1 || calls[0] != "a" {
		t.Errorf("calls = %v, want [a]", calls)
	}

	// Route type_b
	r.Route(nil, mustJSON(t, Message{Type: "type_b"}))
	if len(calls) != 2 || calls[1] != "b" {
		t.Errorf("calls = %v, want [a, b]", calls)
	}
}

func TestRouter_Payload(t *testing.T) {
	r := NewRouter()

	var payload json.RawMessage
	r.Handle("with_payload", func(client *Client, msg Message) {
		payload = msg.Payload
	})

	expectedPayload := json.RawMessage(`{"key":"value"}`)
	msg := Message{
		Type:    "with_payload",
		Payload: expectedPayload,
	}

	r.Route(nil, mustJSON(t, msg))

	if string(payload) != string(expectedPayload) {
		t.Errorf("payload = %s, want %s", string(payload), string(expectedPayload))
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	return data
}
