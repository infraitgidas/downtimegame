package game

import (
	"testing"
)

func TestSimulatedExecutor(t *testing.T) {
	e := NewSimulatedExecutor()

	if e.Name() != "simulated" {
		t.Errorf("Name = %q, want %q", e.Name(), "simulated")
	}

	// Trigger should not error
	scenario := &Scenario{
		ID:              "test-scenario",
		TargetServiceID: "sg-rojo",
	}
	if err := e.Trigger(scenario); err != nil {
		t.Errorf("Trigger failed: %v", err)
	}

	// Resolve should not error
	if err := e.Resolve(scenario); err != nil {
		t.Errorf("Resolve failed: %v", err)
	}
}

func TestEngine_SetExecutor(t *testing.T) {
	e := newTestEngine(t)

	sim := NewSimulatedExecutor()
	e.SetExecutor(sim)

	if e.Executor().Name() != "simulated" {
		t.Errorf("executor = %q, want %q", e.Executor().Name(), "simulated")
	}
}
