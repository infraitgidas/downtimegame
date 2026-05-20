package game

import "log"

// Executor defines the interface for triggering and resolving downtime incidents.
// Implementations can simulate failures (no-op) or execute real failures
// via SSH to LXC containers or other mechanisms.
//
// The engine manages the logical game state (service status, events).
// The executor is responsible for the physical action (e.g., stopping a process).
type Executor interface {
	// Trigger executes a scenario's failure on the target service.
	// Called AFTER the engine has already updated the logical game state.
	Trigger(scenario *Scenario) error

	// Resolve restores the target service to normal operation.
	// Called AFTER the engine has already updated the logical game state.
	Resolve(scenario *Scenario) error

	// Name returns a human-readable name for this executor type.
	Name() string
}

// SimulatedExecutor implements Executor as a no-op.
// The engine handles all state changes for simulation mode.
type SimulatedExecutor struct{}

// NewSimulatedExecutor creates a new no-op simulated executor.
func NewSimulatedExecutor() *SimulatedExecutor {
	return &SimulatedExecutor{}
}

// Trigger logs the action but does nothing physical.
func (e *SimulatedExecutor) Trigger(scenario *Scenario) error {
	log.Printf("Executor [simulated]: trigger %s on %s (no-op)", scenario.ID, scenario.TargetServiceID)
	return nil
}

// Resolve logs the action but does nothing physical.
func (e *SimulatedExecutor) Resolve(scenario *Scenario) error {
	log.Printf("Executor [simulated]: resolve %s on %s (no-op)", scenario.ID, scenario.TargetServiceID)
	return nil
}

// Name returns "simulated".
func (e *SimulatedExecutor) Name() string { return "simulated" }
