package edge

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func edgeVPNUser(deviceID string) string {
	id := sanitizeID(deviceID)
	if id == "" {
		id = "device"
	}
	name := "edge-" + id
	// vpn.ValidName: max 64 chars
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

// EnsureVPNClient creates VPN user for device and returns subscription links.
// Primary VLESS is the online RU relay when available (whitelist-friendly).
func EnsureVPNClient(deviceID string) (map[string]string, error) {
	name := edgeVPNUser(deviceID)
	if !vpn.ValidName(name) {
		return nil, fmt.Errorf("invalid edge vpn name %q", name)
	}
	users, err := vpn.ListNative()
	if err != nil {
		return nil, fmt.Errorf("list vpn users: %w", err)
	}
	found := false
	var uuid string
	for _, u := range users {
		if u.Name == name {
			found = true
			uuid = u.UUID
			if !u.Enabled {
				// ensure peer is usable for router client
				_ = vpn.SetEnabledNative(name, true)
			}
			break
		}
	}
	if !found {
		if _, err := vpn.AddNative(name, "edge device "+deviceID); err != nil {
			return nil, fmt.Errorf("create edge vpn user %s: %w", name, err)
		}
		users, _ = vpn.ListNative()
		for _, u := range users {
			if u.Name == name {
				uuid = u.UUID
				break
			}
		}
	}
	if uuid == "" {
		return nil, fmt.Errorf("edge vpn user %s missing after create", name)
	}
	vless := edgeRelayOrCoreLink(name, uuid)
	if vless == "" {
		return nil, fmt.Errorf("no vpn links for %s — is sing-box installed?", name)
	}
	// Link fields only — never policy (fallback/dns/mode/soft_fallback).
	// Those come from the template and must survive TemplateWithVPN merge.
	return map[string]string{
		"user":         name,
		"subscription": vless,
		"vless":        vless,
		"exit":         "secondary", // informational; not a routing policy key
	}, nil
}

func edgeRelayOrCoreLink(name, uuid string) string {
	if uuid != "" {
		for _, d := range secondary.List() {
			if !secondary.Online(d, 2*time.Minute) || d.PublicIP == "" || d.PBK == "" {
				continue
			}
			sni := d.SNI
			if sni == "" {
				sni = vpn.DefaultRealitySNI
			}
			return vpn.ClientLinkForSecondary(name, uuid, d.PublicIP, d.PBK, d.SID, sni)
		}
		return vpn.VLESSLink(name, uuid)
	}
	v, _ := vpn.ReadClient(name, "link-vless.txt", "link.txt")
	return v
}

func TemplateWithVPN(deviceID string) (Template, error) {
	t, err := TemplateForDevice(deviceID)
	if err != nil {
		return nil, err
	}
	vpnSec, _ := t["vpn"].(map[string]any)
	enabled := vpnTruthy(vpnSec, "enabled", false)
	if !enabled {
		return t, nil
	}
	links, err := EnsureVPNClient(deviceID)
	if err != nil {
		t["vpn_error"] = err.Error()
		return t, nil
	}
	if vpnSec == nil {
		vpnSec = map[string]any{}
	}
	// Merge link fields only; never overwrite operator policy keys.
	policyKeys := map[string]bool{
		"enabled": true, "mode": true, "dns": true, "dns_mode": true,
		"fallback": true, "soft_fallback": true,
	}
	for k, v := range links {
		if policyKeys[k] {
			continue
		}
		vpnSec[k] = v
	}
	// Defaults only when missing (do not clobber template-set values).
	if _, ok := vpnSec["fallback"]; !ok {
		vpnSec["fallback"] = "wan"
	}
	if _, ok := vpnSec["soft_fallback"]; !ok {
		vpnSec["soft_fallback"] = true
	}
	if _, ok := vpnSec["dns"]; !ok {
		if _, ok2 := vpnSec["dns_mode"]; !ok2 {
			vpnSec["dns"] = "vpn"
		}
	}
	if _, ok := vpnSec["mode"]; !ok {
		vpnSec["mode"] = "tun"
	}
	t["vpn"] = vpnSec
	return t, nil
}

func vpnTruthy(m map[string]any, key string, def bool) bool {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		return s == "1" || s == "true" || s == "yes" || s == "on"
	case float64:
		return x != 0
	default:
		return def
	}
}
