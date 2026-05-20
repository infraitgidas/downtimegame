// Package checker monitors the health of game services (LXC containers)
// by polling their /health endpoints at a configurable interval.
// Emits status change events through a callback.
package checker

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ema/downtime-game/internal/game"
)

// StatusChangeHandler is called when a service transitions between online/offline.
type StatusChangeHandler func(serviceID string, online bool)

// Checker periodically polls service health endpoints and detects state changes.
type Checker struct {
	mu        sync.RWMutex
	services  []game.ServiceConfig
	interval  time.Duration
	timeout   time.Duration
	statuses  map[string]bool // serviceID -> online
	onChange  StatusChangeHandler
	stopCh    chan struct{}
	running   bool
}

// New creates a new health checker for the given services.
// interval: how often to poll (default 5s if zero)
// timeout: HTTP request timeout (default 3s if zero)
func New(services []game.ServiceConfig, interval, timeout time.Duration, onChange StatusChangeHandler) *Checker {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	statuses := make(map[string]bool, len(services))
	for _, svc := range services {
		statuses[svc.ID] = true // assume online initially
	}

	return &Checker{
		services: services,
		interval: interval,
		timeout:  timeout,
		statuses: statuses,
		onChange: onChange,
		stopCh:   make(chan struct{}),
	}
}

// Start begins polling services in a background goroutine.
func (c *Checker) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	go c.loop()
	log.Printf("Checker: started (interval=%s, timeout=%s)", c.interval, c.timeout)
}

// Stop terminates the polling goroutine.
func (c *Checker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return
	}
	c.running = false
	close(c.stopCh)
	log.Println("Checker: stopped")
}

// GetStatus returns the current known status of a service.
func (c *Checker) GetStatus(serviceID string) (online bool, known bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	online, known = c.statuses[serviceID]
	return
}

// AllStatuses returns a snapshot of all service statuses.
func (c *Checker) AllStatuses() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot := make(map[string]bool, len(c.statuses))
	for k, v := range c.statuses {
		snapshot[k] = v
	}
	return snapshot
}

// loop is the main polling goroutine.
func (c *Checker) loop() {
	// Initial check immediately
	c.checkAll()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkAll()
		case <-c.stopCh:
			return
		}
	}
}

// checkAll polls every service's health endpoint.
func (c *Checker) checkAll() {
	var wg sync.WaitGroup
	for _, svc := range c.services {
		wg.Add(1)
		go func(svc game.ServiceConfig) {
			defer wg.Done()
			c.checkService(svc)
		}(svc)
	}
	wg.Wait()
}

// checkService polls a single service and emits a status change event if needed.
func (c *Checker) checkService(svc game.ServiceConfig) {
	online := c.pingService(svc)

	c.mu.RLock()
	prevOnline, known := c.statuses[svc.ID]
	c.mu.RUnlock()

	// First check — just store the status, don't emit
	if !known {
		c.mu.Lock()
		c.statuses[svc.ID] = online
		c.mu.Unlock()
		return
	}

	// Status changed
	if prevOnline != online {
		c.mu.Lock()
		c.statuses[svc.ID] = online
		c.mu.Unlock()

		statusStr := map[bool]string{true: "ONLINE", false: "OFFLINE"}[online]
		log.Printf("Checker: %s → %s (%s)", svc.ID, statusStr, svc.HealthURL())

		if c.onChange != nil {
			c.onChange(svc.ID, online)
		}
	}
}

// pingService makes an HTTP GET to the service's /health endpoint.
func (c *Checker) pingService(svc game.ServiceConfig) bool {
	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(svc.HealthURL())
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
