// Package ndconfig loads /etc/netductor/netductor.conf into process env (if unset).
package ndconfig

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// ConfPath is the operator-facing config (KEY=value, # comments).
func ConfPath() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_CONF")); v != "" {
		return v
	}
	return filepath.Join(paths.EtcDir(), "netductor.conf")
}

// Load reads ConfPath and sets env vars that are not already set.
// Mapping: REDIRECT_BASE → NETDUCTOR_REDIRECT_BASE, etc.
func Load() {
	p := ConfPath()
	f, err := os.Open(p)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// optional export KEY=val
		line = strings.TrimPrefix(line, "export ")
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		envKey := mapKey(key)
		if envKey == "" {
			continue
		}
		if strings.TrimSpace(os.Getenv(envKey)) == "" {
			_ = os.Setenv(envKey, val)
		}
	}
}

func mapKey(k string) string {
	k = strings.ToUpper(strings.TrimSpace(k))
	switch k {
	case "REDIRECT_BASE", "NETDUCTOR_REDIRECT_BASE":
		return "NETDUCTOR_REDIRECT_BASE"
	case "VPN_HOST", "NETDUCTOR_VPN_HOST":
		return "NETDUCTOR_VPN_HOST"
	case "CORE_HOST", "NETDUCTOR_CORE_HOST":
		return "NETDUCTOR_CORE_HOST"
	case "REDIRECT_TLS_CERT", "NETDUCTOR_REDIRECT_TLS_CERT":
		return "NETDUCTOR_REDIRECT_TLS_CERT"
	case "REDIRECT_TLS_KEY", "NETDUCTOR_REDIRECT_TLS_KEY":
		return "NETDUCTOR_REDIRECT_TLS_KEY"
	case "PUBLIC_HOSTNAME", "NETDUCTOR_PUBLIC_HOSTNAME":
		return "NETDUCTOR_PUBLIC_HOSTNAME"
	default:
		if strings.HasPrefix(k, "NETDUCTOR_") {
			return k
		}
		return ""
	}
}
