package channels

import (
	"regexp"
	"strings"

	"github.com/PavelNeyman/netductor/internal/vpn"
)

var uuidRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// userFromClashMeta extracts VPN user label from sing-box clash connection metadata.
//
// Important: metadata.host is the *destination* hostname (chatgpt.com, etc.), NOT the
// VPN user. When sing-box omits metadata.user (common on some builds), fall back to
// sourceIP so we aggregate per client, not per site.
func userFromClashMeta(meta map[string]any) (user, uuid string) {
	if meta == nil {
		return "unknown", ""
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := meta[k].(string); ok {
				if s := strings.TrimSpace(v); s != "" {
					return s
				}
			}
		}
		return ""
	}
	user = get("user", "User")
	uuid = get("uuid", "UUID", "uid")
	if user != "" && user != "unknown" {
		return user, uuid
	}
	if uuid != "" {
		return uuid, uuid
	}
	// Client identity when name is not exposed by clash API.
	if sip := get("sourceIP", "source_ip", "src"); sip != "" {
		return "client:" + sip, uuid
	}
	return "unknown", uuid
}

// resolveVPNUserName maps uuid or raw label to registry user name when possible.
func resolveVPNUserName(label, uuid string) string {
	list, err := vpn.List()
	if err != nil || len(list) == 0 {
		return label
	}
	byUUID := map[string]string{}
	byName := map[string]bool{}
	for _, u := range list {
		if u.UUID != "" {
			byUUID[strings.ToLower(u.UUID)] = u.Name
		}
		if u.Name != "" {
			byName[u.Name] = true
		}
	}
	if uuid != "" {
		if n := byUUID[strings.ToLower(uuid)]; n != "" {
			return n
		}
	}
	if label != "" && byName[label] {
		return label
	}
	if uuidRE.MatchString(label) {
		if n := byUUID[strings.ToLower(label)]; n != "" {
			return n
		}
	}
	return label
}
