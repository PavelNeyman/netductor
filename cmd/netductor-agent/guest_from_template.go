package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/guest"
)

func applyGuestFromTemplate(tmpl map[string]any) string {
	g, _ := tmpl["guest"].(map[string]any)
	if g == nil {
		return ""
	}
	enabled := false
	switch v := g["enabled"].(type) {
	case bool:
		enabled = v
	case string:
		enabled = v == "true" || v == "1"
	}
	if !enabled {
		return "\nguest: disabled in template"
	}
	gc, _ := guest.LoadConfig(guestConfigPath())
	gc.Enabled = true
	if s, _ := g["ssid"].(string); s != "" {
		gc.SSID = s
	}
	if gc.SSID == "" {
		gc.SSID = "Guest"
	}
	if s, _ := g["psk"].(string); s != "" {
		gc.PSK = s
	}
	if gc.PSK == "" {
		b := make([]byte, 12)
		_, _ = rand.Read(b)
		gc.PSK = hex.EncodeToString(b)
	}
	if pin, _ := g["desk_pin"].(string); pin != "" {
		gc.DeskPIN = pin
		gc.DeskPINHash = ""
	}
	if gc.DeskPIN == "" && gc.DeskPINHash == "" {
		gc.DeskPIN = "0000"
	}
	hidden := true
	if v, ok := g["hidden"]; ok {
		switch x := v.(type) {
		case bool:
			hidden = x
		case string:
			hidden = x != "0" && x != "false"
		}
	}
	gc.Hidden = hidden
	if gc.DeskPort == 0 {
		gc.DeskPort = 7880
	}
	if gc.CaptivePort == 0 {
		gc.CaptivePort = 7881
	}
	if err := guest.SaveConfig(guestConfigPath(), gc); err != nil {
		return "\nguest config: " + err.Error()
	}
	gc, _ = guest.LoadConfig(guestConfigPath())
	note := applyGuestNetwork(gc)
	return "\nguest enabled: " + note
}

func guestCmdStatus() string {
	st, err := guest.OpenStore(guestStoreDir())
	if err != nil {
		return err.Error()
	}
	gc, _ := guest.LoadConfig(guestConfigPath())
	allow, pend := st.Snapshot()
	var b strings.Builder
	fmt.Fprintf(&b, "enabled=%v ssid=%s\n", gc.Enabled, gc.SSID)
	b.WriteString("pending:\n")
	for _, p := range pend {
		fmt.Fprintf(&b, "  %s %s\n", p.Code, p.MAC)
	}
	b.WriteString("allow:\n")
	for _, a := range allow {
		fmt.Fprintf(&b, "  %s until %s\n", a.MAC, a.ExpiresAt.Format(time.RFC3339))
	}
	if gc.SSID != "" && gc.PSK != "" {
		fmt.Fprintf(&b, "join=%s\n", guest.JoinQR(gc.SSID, gc.PSK, gc.Hidden))
	}
	return b.String()
}

func guestCmdGrant(arg string) string {
	parts := strings.SplitN(arg, "|", 2)
	if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
		return "need CODE|minutes or mac|minutes"
	}
	mins := 10
	if len(parts) == 2 {
		fmt.Sscanf(parts[1], "%d", &mins)
	}
	st, err := guest.OpenStore(guestStoreDir())
	if err != nil {
		return err.Error()
	}
	key := strings.TrimSpace(parts[0])
	mac := key
	if !strings.Contains(key, ":") {
		sess, ok := st.LookupCode(key)
		if !ok {
			return "code not found: " + key
		}
		mac = sess.MAC
	}
	e, err := st.Grant(mac, time.Duration(mins)*time.Minute, "remote")
	if err != nil {
		return err.Error()
	}
	_ = applyGuestFirewallAllow(st)
	return fmt.Sprintf("granted %s until %s", e.MAC, e.ExpiresAt.Format(time.RFC3339))
}

func guestCmdRevoke(arg string) string {
	st, err := guest.OpenStore(guestStoreDir())
	if err != nil {
		return err.Error()
	}
	_ = st.Revoke(arg)
	_ = applyGuestFirewallAllow(st)
	return "revoked " + arg
}
