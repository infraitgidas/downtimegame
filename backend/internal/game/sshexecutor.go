package game

import (
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// ── Constants ─────────────────────────────────────────────────────────────────

// DefaultSSHPort is the default SSH port for LXC/Proxmox connections.
const DefaultSSHPort = 22

// DefaultSSHTimeout is the default SSH connection and command timeout.
const DefaultSSHTimeout = 10 * time.Second

// LXC VMID mapping: service ID → Proxmox CT VMID
var serviceVMIDs = map[string]int{
	"sg-rojo":     200,
	"sg-azul":     201,
	"sg-verde":    202,
	"sg-amarillo": 203,
}

// ── SSHConfig ─────────────────────────────────────────────────────────────────

// SSHConfig holds the parameters to connect to the Proxmox host
// for executing commands inside LXC containers via pct exec.
type SSHConfig struct {
	Host    string        // Proxmox IP (e.g., "192.168.1.31")
	User    string        // SSH user (e.g., "root")
	KeyPath string        // Path to SSH private key
	Port    int           // SSH port (default 22)
	Timeout time.Duration // Connection and command timeout
}

// SSHConfigFromEnv reads SSH configuration from environment variables.
// Returns a config with defaults if env vars are not set.
//
// Env vars:
//   LXC_SSH_HOST     — Proxmox host IP (default: "192.168.1.31")
//   LXC_SSH_USER     — SSH user (default: "root")
//   LXC_SSH_KEY_PATH  — Path to SSH private key (default: "~/.ssh/id_rsa")
//   LXC_SSH_PORT     — SSH port (default: 22)
func SSHConfigFromEnv() SSHConfig {
	cfg := SSHConfig{
		Host:    envOrDefault("LXC_SSH_HOST", "192.168.1.31"),
		User:    envOrDefault("LXC_SSH_USER", "root"),
		KeyPath: envOrDefault("LXC_SSH_KEY_PATH", "~/.ssh/id_rsa"),
		Port:    DefaultSSHPort,
		Timeout: DefaultSSHTimeout,
	}
	if p := os.Getenv("LXC_SSH_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &cfg.Port)
	}
	return cfg
}

// ── SSHExecutor ───────────────────────────────────────────────────────────────

// SSHExecutor implements Executor by running systemctl commands
// inside LXC containers via SSH to the Proxmox host.
//
// Architecture:
//
//	Backend (Go) ──SSH──→ Proxmox (192.168.1.31)
//	                           │
//	                   pct exec {vmid}
//	                           │
//	                    ┌──────┴──────┐
//	                    ▼             ▼
//	               LXC-200       LXC-201
//	               (sg-rojo)     (sg-azul)
//	                    ▼             ▼
//	               systemctl     systemctl
//	               stop/start    stop/start
type SSHExecutor struct {
	config SSHConfig
}

// NewSSHExecutor creates a new SSH executor with the given config.
func NewSSHExecutor(config SSHConfig) *SSHExecutor {
	return &SSHExecutor{config: config}
}

// Name returns "ssh".
func (e *SSHExecutor) Name() string { return "ssh" }

// Trigger connects to Proxmox and stops the target service inside its LXC.
// Called AFTER the engine has already updated the logical game state.
func (e *SSHExecutor) Trigger(scenario *Scenario) error {
	vmid, ok := serviceVMIDs[scenario.TargetServiceID]
	if !ok {
		return fmt.Errorf("no VMID mapping for service: %s", scenario.TargetServiceID)
	}

	serviceName := fmt.Sprintf("sg-%s", colorFromServiceID(scenario.TargetServiceID))
	cmd := fmt.Sprintf("pct exec %d -- systemctl stop %s", vmid, serviceName)

	log.Printf("SSH Executor: triggering failure on %s (CT %d): systemctl stop %s",
		scenario.TargetServiceID, vmid, serviceName)

	output, err := e.runCommand(cmd)
	if err != nil {
		return fmt.Errorf("trigger %s: %w — output: %s", scenario.TargetServiceID, err, output)
	}

	log.Printf("SSH Executor: trigger successful — %s", output)
	return nil
}

// Resolve connects to Proxmox and starts the target service inside its LXC.
// Called AFTER the engine has already updated the logical game state.
func (e *SSHExecutor) Resolve(scenario *Scenario) error {
	vmid, ok := serviceVMIDs[scenario.TargetServiceID]
	if !ok {
		return fmt.Errorf("no VMID mapping for service: %s", scenario.TargetServiceID)
	}

	serviceName := fmt.Sprintf("sg-%s", colorFromServiceID(scenario.TargetServiceID))
	cmd := fmt.Sprintf("pct exec %d -- systemctl start %s", vmid, serviceName)

	log.Printf("SSH Executor: resolving failure on %s (CT %d): systemctl start %s",
		scenario.TargetServiceID, vmid, serviceName)

	output, err := e.runCommand(cmd)
	if err != nil {
		return fmt.Errorf("resolve %s: %w — output: %s", scenario.TargetServiceID, err, output)
	}

	log.Printf("SSH Executor: resolve successful — %s", output)
	return nil
}

// ── internal ──────────────────────────────────────────────────────────────────

// runCommand executes a command on the Proxmox host via SSH and returns output.
func (e *SSHExecutor) runCommand(cmd string) (string, error) {
	// Read SSH key
	key, err := os.ReadFile(expandPath(e.config.KeyPath))
	if err != nil {
		return "", fmt.Errorf("read SSH key %s: %w", e.config.KeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("parse SSH key: %w", err)
	}

	// Connect to Proxmox host
	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            e.config.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: use known_hosts in production
		Timeout:         e.config.Timeout,
	})
	if err != nil {
		return "", fmt.Errorf("SSH dial %s: %w", addr, err)
	}
	defer client.Close()

	// Open session and run command
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// colorFromServiceID extracts the color name from a service ID like "sg-rojo" → "rojo".
func colorFromServiceID(id string) string {
	if len(id) > 3 && id[:3] == "sg-" {
		return id[3:]
	}
	return id
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}

// envOrDefault returns the env var value or a default if not set.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// ── NewExecutorFromEnv ────────────────────────────────────────────────────────

// NewExecutorFromEnv creates an Executor based on the EXECUTOR_MODE env var.
//
//	EXECUTOR_MODE=simulated (default) → SimulatedExecutor (no-op)
//	EXECUTOR_MODE=ssh               → SSHExecutor (real LXC control)
//
// SSH configuration is read from env vars LXC_SSH_* (see SSHConfigFromEnv).
func NewExecutorFromEnv() Executor {
	mode := os.Getenv("EXECUTOR_MODE")
	if mode == "" {
		mode = "simulated"
	}

	switch mode {
	case "ssh":
		cfg := SSHConfigFromEnv()
		log.Printf("Executor mode: SSH (host=%s, user=%s, key=%s)",
			cfg.Host, cfg.User, cfg.KeyPath)
		return NewSSHExecutor(cfg)

	case "simulated":
		fallthrough
	default:
		log.Printf("Executor mode: simulated (no-op)")
		return NewSimulatedExecutor()
	}
}
