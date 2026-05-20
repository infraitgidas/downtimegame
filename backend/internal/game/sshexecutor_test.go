package game

import (
	"os"
	"testing"
)

// ── Helper tests ──────────────────────────────────────────────────────────────

func TestColorFromServiceID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"sg-rojo", "rojo"},
		{"sg-azul", "azul"},
		{"sg-verde", "verde"},
		{"sg-amarillo", "amarillo"},
		{"rojo", "rojo"},       // no prefix
		{"sg-", "sg-"},         // no chars after prefix → returns as-is
		{"", ""},               // empty string
	}

	for _, tt := range tests {
		got := colorFromServiceID(tt.id)
		if got != tt.want {
			t.Errorf("colorFromServiceID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"~/.ssh/id_rsa", home + "/.ssh/id_rsa"},
		{"/etc/ssh/key", "/etc/ssh/key"},
		{"relative/path", "relative/path"},
		{"~", home},
	}

	for _, tt := range tests {
		got := expandPath(tt.path)
		if got != tt.want {
			t.Errorf("expandPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestEnvOrDefault(t *testing.T) {
	const key = "TEST_ENV_OR_DEFAULT_KEY"
	// Clean up
	defer os.Unsetenv(key)

	// Not set → default
	if got := envOrDefault(key, "defaultVal"); got != "defaultVal" {
		t.Errorf("unset: got %q, want %q", got, "defaultVal")
	}

	// Set → env value
	os.Setenv(key, "envValue")
	if got := envOrDefault(key, "defaultVal"); got != "envValue" {
		t.Errorf("set: got %q, want %q", got, "envValue")
	}

	// Empty → default
	os.Setenv(key, "")
	if got := envOrDefault(key, "defaultVal"); got != "defaultVal" {
		t.Errorf("empty: got %q, want %q", got, "defaultVal")
	}
}

// ── SSHConfigFromEnv tests ────────────────────────────────────────────────────

func TestSSHConfigFromEnv_Defaults(t *testing.T) {
	// Unset all LXC_SSH_* vars
	for _, k := range []string{"LXC_SSH_HOST", "LXC_SSH_USER", "LXC_SSH_KEY_PATH", "LXC_SSH_PORT"} {
		os.Unsetenv(k)
	}

	cfg := SSHConfigFromEnv()

	if cfg.Host != "192.168.1.31" {
		t.Errorf("Host = %q, want %q", cfg.Host, "192.168.1.31")
	}
	if cfg.User != "root" {
		t.Errorf("User = %q, want %q", cfg.User, "root")
	}
	if cfg.KeyPath != "~/.ssh/id_rsa" {
		t.Errorf("KeyPath = %q, want %q", cfg.KeyPath, "~/.ssh/id_rsa")
	}
	if cfg.Port != DefaultSSHPort {
		t.Errorf("Port = %d, want %d", cfg.Port, DefaultSSHPort)
	}
	if cfg.Timeout != DefaultSSHTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultSSHTimeout)
	}
}

func TestSSHConfigFromEnv_Custom(t *testing.T) {
	// Set custom values
	os.Setenv("LXC_SSH_HOST", "10.0.0.1")
	os.Setenv("LXC_SSH_USER", "admin")
	os.Setenv("LXC_SSH_KEY_PATH", "/opt/keys/proxmox")
	os.Setenv("LXC_SSH_PORT", "2222")
	defer func() {
		os.Unsetenv("LXC_SSH_HOST")
		os.Unsetenv("LXC_SSH_USER")
		os.Unsetenv("LXC_SSH_KEY_PATH")
		os.Unsetenv("LXC_SSH_PORT")
	}()

	cfg := SSHConfigFromEnv()

	if cfg.Host != "10.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Host, "10.0.0.1")
	}
	if cfg.User != "admin" {
		t.Errorf("User = %q, want %q", cfg.User, "admin")
	}
	if cfg.KeyPath != "/opt/keys/proxmox" {
		t.Errorf("KeyPath = %q, want %q", cfg.KeyPath, "/opt/keys/proxmox")
	}
	if cfg.Port != 2222 {
		t.Errorf("Port = %d, want %d", cfg.Port, 2222)
	}
}

// ── NewExecutorFromEnv tests ──────────────────────────────────────────────────

func TestNewExecutorFromEnv_Default(t *testing.T) {
	os.Unsetenv("EXECUTOR_MODE")
	e := NewExecutorFromEnv()

	if e.Name() != "simulated" {
		t.Errorf("default executor = %q, want %q", e.Name(), "simulated")
	}

	// Verify it's actually a SimulatedExecutor
	if _, ok := e.(*SimulatedExecutor); !ok {
		t.Errorf("expected *SimulatedExecutor, got %T", e)
	}
}

func TestNewExecutorFromEnv_Simulated(t *testing.T) {
	os.Setenv("EXECUTOR_MODE", "simulated")
	defer os.Unsetenv("EXECUTOR_MODE")

	e := NewExecutorFromEnv()
	if e.Name() != "simulated" {
		t.Errorf("simulated executor = %q, want %q", e.Name(), "simulated")
	}
}

func TestNewExecutorFromEnv_SSH(t *testing.T) {
	os.Setenv("EXECUTOR_MODE", "ssh")
	os.Setenv("LXC_SSH_HOST", "192.168.1.31")
	os.Setenv("LXC_SSH_USER", "root")
	os.Setenv("LXC_SSH_KEY_PATH", "~/.ssh/id_rsa")
	defer func() {
		os.Unsetenv("EXECUTOR_MODE")
		os.Unsetenv("LXC_SSH_HOST")
		os.Unsetenv("LXC_SSH_USER")
		os.Unsetenv("LXC_SSH_KEY_PATH")
	}()

	e := NewExecutorFromEnv()
	if e.Name() != "ssh" {
		t.Errorf("ssh executor = %q, want %q", e.Name(), "ssh")
	}

	// Verify it's actually an SSHExecutor
	if _, ok := e.(*SSHExecutor); !ok {
		t.Errorf("expected *SSHExecutor, got %T", e)
	}
}

func TestNewExecutorFromEnv_UnknownMode(t *testing.T) {
	os.Setenv("EXECUTOR_MODE", "foobar")
	defer os.Unsetenv("EXECUTOR_MODE")

	// Unknown mode should fall back to simulated
	e := NewExecutorFromEnv()
	if e.Name() != "simulated" {
		t.Errorf("unknown mode executor = %q, want %q", e.Name(), "simulated")
	}
}

// ── VMID mapping test ─────────────────────────────────────────────────────────

func TestServiceVMIDs(t *testing.T) {
	expected := map[string]int{
		"sg-rojo":     200,
		"sg-azul":     201,
		"sg-verde":    202,
		"sg-amarillo": 203,
	}

	if len(serviceVMIDs) != len(expected) {
		t.Errorf("serviceVMIDs has %d entries, want %d", len(serviceVMIDs), len(expected))
	}

	for id, want := range expected {
		got, ok := serviceVMIDs[id]
		if !ok {
			t.Errorf("serviceVMIDs missing entry for %q", id)
			continue
		}
		if got != want {
			t.Errorf("serviceVMIDs[%q] = %d, want %d", id, got, want)
		}
	}
}

// ── Integration test (manual, requires real LXC) ─────────────────────────────
// Para correr: LXC_SSH_HOST=192.168.1.31 go test -run TestSSHExecutor_Real -v
// Requiere:
//   - Proxmox host accesible por SSH con key auth
//   - CTs 200-203 existentes y corriendo
//   - Servicios sg-{color} instalados (via deploy.sh)

func TestSSHExecutor_Real(t *testing.T) {
	host := os.Getenv("LXC_SSH_HOST")
	if host == "" {
		t.Skip("SKIP: LXC_SSH_HOST not set — this test requires real Proxmox/LXC hardware")
	}

	cfg := SSHConfigFromEnv()
	t.Logf("Testing against Proxmox host: %s (user=%s, key=%s)",
		cfg.Host, cfg.User, cfg.KeyPath)

	e := NewSSHExecutor(cfg)

	if e.Name() != "ssh" {
		t.Fatalf("Name = %q, want %q", e.Name(), "ssh")
	}

	// Pick a scenario that targets sg-verde (the least critical service)
	scenario := &Scenario{
		ID:              "test-real-trigger",
		Name:            "Test Real Trigger",
		TargetServiceID: "sg-verde",
		FailureType:     "crash",
		TimeLimit:       120,
	}

	// Step 1: Trigger — should stop the service via pct exec
	t.Log("Step 1: Triggering failure on sg-verde...")
	if err := e.Trigger(scenario); err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}
	t.Log("✅ Trigger OK")

	// Verify service is actually down
	// (We could curl the health endpoint to confirm, but that's outside this test)

	// Step 2: Resolve — should restart the service
	t.Log("Step 2: Resolving failure on sg-verde...")
	if err := e.Resolve(scenario); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	t.Log("✅ Resolve OK")

	t.Log("⚠️  Remember to verify services are healthy after test")
}
