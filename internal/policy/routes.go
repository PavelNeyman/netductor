package policy

import "strings"

// Subject is a VLESS auth name (usually VPN user name) with its policy.
type Subject struct {
	Name   string
	Policy AccessPolicy
}

// ServiceRouteRules builds sing-box route rules that enforce internal service ACL.
// Requires inbound VLESS users to include a matching "name" field (auth_user).
//
// For each internal catalog endpoint:
//  1. allow auth_user ∈ permitted → outbound direct (optional ip + port)
//  2. reject other users hitting that port (and ip when set)
// allow_internet=false gets a broad reject for those users after service allows.
func ServiceRouteRules(subjects []Subject, cat *Catalog) []any {
	if cat == nil {
		cat = DefaultCatalog()
	}
	var rules []any
	seenPort := map[int]bool{}

	for _, svc := range cat.Services {
		if svc.Disabled || svc.Kind != KindInternal {
			continue
		}
		allowed := subjectsAllowing(subjects, svc.ID, cat)
		for _, ep := range svc.Endpoints {
			if ep.Port <= 0 {
				continue
			}
			port := ep.Port
			proto := strings.ToLower(ep.Proto)
			if proto == "" {
				proto = "tcp"
			}
			// Prefer one rule set per port (loopback + service-net share port)
			if seenPort[port] {
				continue
			}
			seenPort[port] = true

			if len(allowed) > 0 {
				rule := map[string]any{
					"auth_user": allowed,
					"port":      port,
					"outbound":  "direct",
				}
				if proto == "tcp" || proto == "udp" {
					rule["network"] = proto
				}
				rules = append(rules, rule)
			}
			rej := map[string]any{
				"port":     port,
				"action":   "reject",
				"outbound": "block",
			}
			if proto == "tcp" || proto == "udp" {
				rej["network"] = proto
			}
			rules = append(rules, rej)
		}
	}

	var noNet []string
	for _, s := range subjects {
		s.Policy.Normalize()
		if !s.Policy.AllowInternet {
			if n := strings.TrimSpace(s.Name); n != "" {
				noNet = append(noNet, n)
			}
		}
	}
	if len(noNet) > 0 {
		rules = append(rules, map[string]any{
			"auth_user": noNet,
			"action":    "reject",
			"outbound":  "block",
		})
	}
	return rules
}

func subjectsAllowing(subjects []Subject, serviceID string, cat *Catalog) []string {
	var names []string
	seen := map[string]bool{}
	for _, s := range subjects {
		if s.Name == "" {
			continue
		}
		if s.Policy.Allows(serviceID, cat) {
			names = append(names, s.Name)
			seen[s.Name] = true
		}
	}
	// relay-uplink is internet egress only. Service-net from secondary dials primary
	// as the real user (uplink-svc-<name>), so ACL stays per-user.
	_ = seen
	return names
}
