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

// BinDir is where node binaries live (FHS).
func BinDir() string { return env("NETDUCTOR_BIN", "/usr/local/bin") }

// ShareDir is static share data (admin UI, scripts).
func ShareDir() string { return env("NETDUCTOR_SHARE", "/usr/local/share/netductor") }

// OptDir is deprecated install prefix. Defaults to StateDir (data only).
// Override NETDUCTOR_ROOT only for migration from /opt/netductor.
func OptDir() string {
	return env("NETDUCTOR_ROOT", StateDir())
}

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
	return env("NETDUCTOR_ADMIN_ROOT", filepath.Join(ShareDir(), "admin"))
}
func VPNBin() string {
	return env("NETDUCTOR_VPN_BIN", filepath.Join(BinDir(), "netductor"))
}
func ProfilesDir() string {
	return env("NETDUCTOR_PROFILES", filepath.Join(StateDir(), "profiles"))
}
func TelegramDir() string {
	return env("NETDUCTOR_TELEGRAM", filepath.Join(StateDir(), "telegram"))
}
func LampacDir() string {
	return env("NETDUCTOR_LAMPAC", filepath.Join(StateDir(), "lampac"))
}
func ScriptsDir() string {
	return env("NETDUCTOR_SCRIPTS", filepath.Join(ShareDir(), "scripts"))
}

func EnsureLayout() error {
	for _, d := range []string{
		EtcDir(), filepath.Join(EtcDir(), "secrets"), filepath.Join(EtcDir(), "sessions"),
		filepath.Join(EtcDir(), "clients"), StateDir(), filepath.Join(StateDir(), "metrics"),
		filepath.Join(StateDir(), "edge"), ProfilesDir(), TelegramDir(), LampacDir(),
		ShareDir(), AdminRoot(), ScriptsDir(),
		filepath.Join(StateDir(), "svc-paths"),
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
func SecondaryDir() string {
	return filepath.Join(StateDir(), "secondary")
}

func SecondaryDevicesFile() string {
	return filepath.Join(SecondaryDir(), "devices.json")
}
