package ndconfig

import (
	"fmt"
	"os"
	"strconv"
	"path/filepath"
	"strings"
)

// Product defaults (overridable via netductor.conf / env). Keys load through Load().

const (
	DefSSHPort           = 52222
	DefAgentMTLSPort     = "8789"
	DefAPIPort           = "8787"
	DefRedirectHTTPSPort = "8443"
	DefDefaultSNI        = "api.vk.me"
	DefLampacPort        = "9118"
	DefSvcSPPrimary      = "10.87.10.1"
	DefSvcSPSecondary    = "10.87.10.2"
	DefSvcPSPrimary      = "10.87.11.1"
	DefSvcPSSecondary    = "10.87.11.2"
	DefSvcSPCIDR         = "10.87.10.0/30"
	DefSvcPSCIDR         = "10.87.11.0/30"
)

// EnvOr returns env if non-empty, else def.
func EnvOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// SSHPort — NETDUCTOR_SSH_PORT or 52222.
func SSHPort() int {
	v := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT"))
	if v == "" {
		return DefSSHPort
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 65535 {
		return DefSSHPort
	}
	return n
}

func AgentMTLSPort() string { return EnvOr("NETDUCTOR_AGENT_MTLS_PORT", DefAgentMTLSPort) }
func APIPort() string       { return EnvOr("NETDUCTOR_API_PORT", DefAPIPort) }

// RedirectHTTPSPort for LE import redirect (not Reality :443).
func RedirectHTTPSPort() string {
	return EnvOr("NETDUCTOR_REDIRECT_HTTPS_PORT", DefRedirectHTTPSPort)
}

// DefaultSNI for Reality when operator does not pass --sni / set-sni yet.
func DefaultSNI() string {
	if v := strings.TrimSpace(os.Getenv("SINGBOX_REALITY_SNI")); v != "" {
		return v
	}
	return EnvOr("NETDUCTOR_DEFAULT_SNI", DefDefaultSNI)
}

func LampacPort() string { return EnvOr("NETDUCTOR_LAMPAC_PORT", DefLampacPort) }

// HY2Enabled — off by default (product dropped HY2 for clients).
func HY2Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("NETDUCTOR_HY2")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func SvcSPCIDR() string { return EnvOr("NETDUCTOR_SVC_SP_CIDR", DefSvcSPCIDR) }
func SvcPSCIDR() string { return EnvOr("NETDUCTOR_SVC_PS_CIDR", DefSvcPSCIDR) }
func SvcSPPrimary() string { return EnvOr("NETDUCTOR_SVC_SP_PRIMARY", DefSvcSPPrimary) }
func SvcSPSecondary() string {
	return EnvOr("NETDUCTOR_SVC_SP_SECONDARY", DefSvcSPSecondary)
}

// DefaultsConfSnippet is written into netductor.conf (commented) for operators.
func DefaultsConfSnippet() string {
	return fmt.Sprintf(`# netductor product defaults (uncomment to override)
# SSH_PORT=%d
# AGENT_MTLS_PORT=%s
# API_PORT=%s
# REDIRECT_HTTPS_PORT=%s
# DEFAULT_SNI=%s
# LAMPAC_PORT=%s
# HY2=0
# SVC_SP_CIDR=%s
# SVC_PS_CIDR=%s
`, DefSSHPort, DefAgentMTLSPort, DefAPIPort, DefRedirectHTTPSPort, DefDefaultSNI, DefLampacPort, DefSvcSPCIDR, DefSvcPSCIDR)
}

// EnsureDefaultsInConf appends commented defaults if conf has no SSH_PORT / DEFAULT_SNI hints.
func EnsureDefaultsInConf() {
	path := ConfPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	b, err := os.ReadFile(path)
	exist := ""
	if err == nil {
		exist = string(b)
	}
	if strings.Contains(exist, "DEFAULT_SNI") || strings.Contains(exist, "# SSH_PORT=") {
		return
	}
	snippet := "\n" + DefaultsConfSnippet()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(snippet)
}
