package edge

import (
	"fmt"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func edgeVPNUser(deviceID string) string {
	id := sanitizeID(deviceID)
	if id == "" {
		id = "device"
	}
	return "edge-" + id
}

// EnsureVPNClient creates VPN user for device and returns subscription links.
// Primary VLESS is the online RU relay when available (whitelist-friendly).
func EnsureVPNClient(deviceID string) (map[string]string, error) {
	name := edgeVPNUser(deviceID)
	users, _ := vpn.ListNative()
	found := false
	var uuid string
	for _, u := range users {
		if u.Name == name {
			found = true
			uuid = u.UUID
			break
		}
	}
	if !found {
		if _, err := vpn.AddNative(name, "edge device "+deviceID); err != nil {
			_ = err
		}
		users, _ = vpn.ListNative()
		for _, u := range users {
			if u.Name == name {
				uuid = u.UUID
				break
			}
		}
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
				sni = "ya.ru"
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
	enabled := false
	if vpnSec != nil {
		switch v := vpnSec["enabled"].(type) {
		case bool:
			enabled = v
		case string:
			enabled = v == "true" || v == "1"
		}
	}
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
