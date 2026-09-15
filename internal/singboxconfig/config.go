// Package singboxconfig holds canonical paths and versioned write helpers for sing-box.
package singboxconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

const (
	// SchemaVersion bumps when outbound/inbound layout changes incompatibly.
	SchemaVersion = 1
	BinaryPath    = "/usr/local/bin/sing-box"
	ConfDir       = "/usr/local/etc/sing-box"
	ConfPath      = ConfDir + "/config.json"
)

// WriteJSON writes config with mode 0600 and optional schema marker in top-level _netductor.
func WriteJSON(cfg map[string]any) error {
	if cfg == nil {
		cfg = map[string]any{}
	}
	cfg["_netductor"] = map[string]any{"schema": SchemaVersion}
	if err := os.MkdirAll(ConfDir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfPath, append(raw, '\n'), 0o600)
}

// Check runs `sing-box check`.
func Check() error {
	if _, err := os.Stat(BinaryPath); err != nil {
		return fmt.Errorf("sing-box binary missing")
	}
	out, err := exec.Command(BinaryPath, "check", "-c", ConfPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sing-box check: %s: %w", string(out), err)
	}
	return nil
}

// EnsurePerms forces 0600 on config.json.
func EnsurePerms() error {
	return os.Chmod(ConfPath, 0o600)
}

// ConfDirPath returns ConfDir for callers that need the directory.
func ConfDirPath() string { return ConfDir }
