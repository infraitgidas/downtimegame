package checker

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ema/downtime-game/internal/game"
)

// mockService wraps an httptest.Server that simulates a game service's /health endpoint.
type mockService struct {
	mu       sync.RWMutex
	server   *httptest.Server
	online   bool
	delay    time.Duration
}

func newMockService(id string, startOnline bool) *mockService {
	m := &mockService{online: startOnline}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		online := m.online
		delay := m.delay
		m.mu.RUnlock()

		if delay > 0 {
			time.Sleep(delay)
		}

		if !online {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	return m
}

func (m *mockService) SetOnline(online bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.online = online
}

func (m *mockService) SetDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = d
}

func (m *mockService) Close() {
	m.server.Close()
}

func (m *mockService) URL() string {
	return m.server.URL
}

// serviceConfigFromAddr creates a ServiceConfig pointing to a mock server.
// addr is the listener address like "127.0.0.1:41645".
func serviceConfigFromAddr(id, name, addr string) game.ServiceConfig {
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	if host == "" {
		host = "127.0.0.1"
	}
	return game.ServiceConfig{
		ID:    id,
		Name:  name,
		Color: name,
		IP:    host,
		Port:  port,
	}
}

func TestChecker_InitialStatus(t *testing.T) {
	svc := newMockService("test-svc", true)
	defer svc.Close()

	changes := make(chan struct {
		serviceID string
		online    bool
	}, 10)

	services := []game.ServiceConfig{
		serviceConfigFromAddr("test-svc", "Test", svc.server.Listener.Addr().String()),
	}

	c := New(services, 50*time.Millisecond, 100*time.Millisecond,
		func(serviceID string, online bool) {
			changes <- struct {
				serviceID string
				online    bool
			}{serviceID, online}
		})

	c.Start()
	defer c.Stop()

	// Wait for initial check
	time.Sleep(100 * time.Millisecond)

	// Service should be online
	online, known := c.GetStatus("test-svc")
	if !known {
		t.Fatal("service should be known after first check")
	}
	if !online {
		t.Error("service should be online")
	}
}

func TestChecker_DetectsOffline(t *testing.T) {
	svc := newMockService("test-svc", true)
	defer svc.Close()

	changeDetected := make(chan bool, 1)

	services := []game.ServiceConfig{
		serviceConfigFromAddr("test-svc", "Test", svc.server.Listener.Addr().String()),
	}

	c := New(services, 50*time.Millisecond, 100*time.Millisecond,
		func(serviceID string, online bool) {
			if !online {
				changeDetected <- true
			}
		})

	c.Start()
	defer c.Stop()

	// Let it establish initial state (online)
	time.Sleep(100 * time.Millisecond)

	// Take service offline
	svc.SetOnline(false)

	// Wait for change detection
	select {
	case <-changeDetected:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for offline detection")
	}

	// Verify status
	online, _ := c.GetStatus("test-svc")
	if online {
		t.Error("service should be offline")
	}
}

func TestChecker_DetectsOnline(t *testing.T) {
	svc := newMockService("test-svc", false)
	defer svc.Close()

	changeDetected := make(chan bool, 1)

	services := []game.ServiceConfig{
		serviceConfigFromAddr("test-svc", "Test", svc.server.Listener.Addr().String()),
	}

	c := New(services, 50*time.Millisecond, 100*time.Millisecond,
		func(serviceID string, online bool) {
			if online {
				changeDetected <- true
			}
		})

	c.Start()
	defer c.Stop()

	// Let it establish initial state (offline)
	time.Sleep(100 * time.Millisecond)

	// Bring service online
	svc.SetOnline(true)

	// Wait for change detection
	select {
	case <-changeDetected:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for online detection")
	}

	// Verify status
	online, _ := c.GetStatus("test-svc")
	if !online {
		t.Error("service should be online")
	}
}

func TestChecker_MultipleServices(t *testing.T) {
	svc1 := newMockService("svc-1", true)
	defer svc1.Close()
	svc2 := newMockService("svc-2", true)
	defer svc2.Close()

	var mu sync.Mutex
	changeCount := 0

	services := []game.ServiceConfig{
		serviceConfigFromAddr("svc-1", "Service 1", svc1.server.Listener.Addr().String()),
		serviceConfigFromAddr("svc-2", "Service 2", svc2.server.Listener.Addr().String()),
	}

	c := New(services, 50*time.Millisecond, 100*time.Millisecond,
		func(serviceID string, online bool) {
			mu.Lock()
			changeCount++
			mu.Unlock()
		})

	c.Start()
	defer c.Stop()

	time.Sleep(100 * time.Millisecond)

	// Verify both services are known
	for _, id := range []string{"svc-1", "svc-2"} {
		_, known := c.GetStatus(id)
		if !known {
			t.Errorf("service %s should be known", id)
		}
	}
}

func TestChecker_TimeoutDetection(t *testing.T) {
	// Service that takes too long to respond
	svc := newMockService("slow-svc", true)
	svc.SetDelay(500 * time.Millisecond) // longer than checker timeout
	defer svc.Close()

	services := []game.ServiceConfig{
		serviceConfigFromAddr("slow-svc", "Slow", svc.server.Listener.Addr().String()),
	}

	c := New(services, 200*time.Millisecond, 100*time.Millisecond,
		func(serviceID string, online bool) {
			t.Logf("Status change: %s -> online=%v", serviceID, online)
		})

	c.Start()
	defer c.Stop()

	// Wait for the checker to try and timeout
	time.Sleep(500 * time.Millisecond)

	// Service should be considered offline due to timeout
	online, known := c.GetStatus("slow-svc")
	if !known {
		t.Fatal("service should be known after check")
	}
	if online {
		t.Error("slow service should be considered offline (timeout)")
	}
}

func TestChecker_AllStatuses(t *testing.T) {
	svc1 := newMockService("svc-1", true)
	defer svc1.Close()
	svc2 := newMockService("svc-2", false)
	defer svc2.Close()

	services := []game.ServiceConfig{
		serviceConfigFromAddr("svc-1", "Online", svc1.server.Listener.Addr().String()),
		serviceConfigFromAddr("svc-2", "Offline", svc2.server.Listener.Addr().String()),
	}

	c := New(services, 50*time.Millisecond, 100*time.Millisecond, nil)
	c.Start()
	defer c.Stop()

	time.Sleep(100 * time.Millisecond)

	_ = c.AllStatuses() // Just verify no panic
}

func TestChecker_StartStop(t *testing.T) {
	svc := newMockService("test-svc", true)
	defer svc.Close()

	services := []game.ServiceConfig{{
		ID:   "test-svc",
		Name: "Test",
		IP:   svc.server.Listener.Addr().String(),
		Port: 80,
	}}

	// Start and stop multiple times — should not panic
	c := New(services, 50*time.Millisecond, 100*time.Millisecond, nil)
	c.Start()
	c.Stop()
}

func TestChecker_InvalidInterval(t *testing.T) {
	// Zero interval should use default (5s)
	c := New(nil, 0, 0, nil)
	if c.interval != 5*time.Second {
		t.Errorf("interval = %v, want 5s", c.interval)
	}
	if c.timeout != 3*time.Second {
		t.Errorf("timeout = %v, want 3s", c.timeout)
	}
}

func TestChecker_NoCallback(t *testing.T) {
	// Should not panic when onChange is nil
	svc := newMockService("test-svc", true)
	defer svc.Close()

	services := []game.ServiceConfig{{
		ID:   "test-svc",
		Name: "Test",
		IP:   svc.server.Listener.Addr().String(),
		Port: 80,
	}}

	c := New(services, 50*time.Millisecond, 100*time.Millisecond, nil)
	c.Start()
	defer c.Stop()

	time.Sleep(100 * time.Millisecond)

	// Take offline — should not panic even without callback
	svc.SetOnline(false)
	time.Sleep(100 * time.Millisecond)

	online, known := c.GetStatus("test-svc")
	if !known {
		t.Fatal("service should be known")
	}
	if online {
		t.Error("service should be offline")
	}
}
