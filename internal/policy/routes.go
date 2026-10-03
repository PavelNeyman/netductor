package policy

import "strings"

// Subject is a VLESS auth name (usually VPN user name) with its policy.
type Subject struct {
	Name   string
	Policy AccessPolicy
}

// ServiceRouteRules builds sing-box route rules that enforce internal service ACL.
// Requires inbound VLESS users to include a matching "name" field (auth_user).
// Rules are ordered to run after sniff/dns; caller prepends those.
//
// For each internal catalog endpoint (port):
//  1. allow auth_user ∈ permitted subjects → continue (no outbound = fall through / direct)
//  2. reject other users hitting that port
// Subjects with services_mode=all are permitted to every internal service.
// Egress (internet) is not fully enforced here when final outbound is shared;
// allow_internet=false gets a broad reject-after-private rule when possible.
func ServiceRouteRules(subjects []Subject, cat *Catalog) []any {
	if cat == nil {
		cat = DefaultCatalog()
	}
	var rules []any

	// private / loopback always ok for system — do not reject RFC1918 wholesale
	// (would break LAN). Only pin service ports from catalog.

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
			if len(allowed) > 0 {
				// permitted users: explicit pass to direct
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
			// everyone else: reject this port
			rej := map[string]any{
				"port":     port,
				"action":   "reject",
				"outbound": "block", // dual style for sing-box variants
			}
			if proto == "tcp" || proto == "udp" {
				rej["network"] = proto
			}
			rules = append(rules, rej)
		}
	}

	// allow_internet=false: reject remaining traffic for those users (after service allows)
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
	for _, s := range subjects {
		if s.Name == "" {
			continue
		}
		if s.Policy.Allows(serviceID, cat) {
			names = append(names, s.Name)
		}
	}
	return names
}
