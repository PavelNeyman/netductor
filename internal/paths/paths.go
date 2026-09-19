package paths

import (
	"os"
	"path/filepath"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func EtcDir() string   { return env("NETDUCTOR_ETC", "/etc/netductor") }
func StateDir() string { return env("NETDUCTOR_STATE", "/var/lib/netductor") }
func OptDir() string   { return env("NETDUCTOR_ROOT", "/opt/netductor") }

func SessionsDir() string {
	return env("NETDUCTOR_SESSIONS", filepath.Join(EtcDir(), "sessions"))
}
func ClientsDir() string {
	return env("NETDUCTOR_CLIENTS", filepath.Join(EtcDir(), "clients"))
}
func EdgeTokenFile() string {
	return env("NETDUCTOR_EDGE_TOKEN_FILE", filepath.Join(EtcDir(), "secrets", "edge_token"))
}
func EdgeDir() string {
	return env("NETDUCTOR_EDGE_DIR", filepath.Join(StateDir(), "edge"))
}
func MetricsDir() string {
	return env("NETDUCTOR_METRICS_DIR", filepath.Join(StateDir(), "metrics"))
}
func ProbesCfg() string {
	return env("NETDUCTOR_PROBES_CFG", filepath.Join(EtcDir(), "probes.json"))
}
func AdminRoot() string {
	return env("NETDUCTOR_ADMIN_ROOT", filepath.Join(OptDir(), "runtime", "api", "admin"))
}
func VPNBin() string {
	return env("NETDUCTOR_VPN_BIN", "/usr/local/bin/netductor")
}

func EnsureLayout() error {
	for _, d := range []string{
		EtcDir(), filepath.Join(EtcDir(), "secrets"), filepath.Join(EtcDir(), "sessions"),
		filepath.Join(EtcDir(), "clients"), StateDir(), filepath.Join(StateDir(), "metrics"),
		filepath.Join(StateDir(), "edge"), OptDir(), filepath.Join(OptDir(), "bin"),
		filepath.Join(OptDir(), "runtime", "api", "admin"),
		filepath.Join(OptDir(), "runtime", "telegram"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	_ = os.Chmod(filepath.Join(EtcDir(), "secrets"), 0o700)
	_ = os.Chmod(filepath.Join(EtcDir(), "sessions"), 0o700)
	_ = os.Chmod(filepath.Join(EtcDir(), "clients"), 0o700)
	return nil
}

// SecondaryDir is state for the RU VPN-entry node.
// Prefers secondary/; falls back to legacy relay/ for reads; mkdir uses secondary.
func SecondaryDir() string {
	sec := filepath.Join(StateDir(), "secondary")
	if st, err := os.Stat(sec); err == nil && st.IsDir() {
		return sec
	}
	legacy := filepath.Join(StateDir(), "relay")
	if st, err := os.Stat(legacy); err == nil && st.IsDir() {
		return legacy
	}
	return sec
}

// SecondaryDevicesFile is devices.json under SecondaryDir.
func SecondaryDevicesFile() string {
	return filepath.Join(SecondaryDir(), "devices.json")
}
