package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/guest"
)

func runGuestCLI(args []string) {
	if len(args) == 0 || args[0] == "-h" || args[0] == "help" {
		fmt.Print(`netductor-agent guest — guest Wi‑Fi (OpenWrt)

  guest status
  guest enable --ssid=Shop-Guest --pin=1234 [--psk=...] [--hidden=1]
  guest disable
  guest grant <mac> [minutes]   # default 10, max 1440
  guest revoke <mac>
  guest join-qr
`)
		return
	}
	switch args[0] {
	case "status":
		gc, err := guest.LoadConfig(guestConfigPath())
		if err != nil {
			fmt.Println("config:", err)
		} else {
			fmt.Printf("enabled=%v ssid=%s hidden=%v desk=:%d captive=:%d\n",
				gc.Enabled, gc.SSID, gc.Hidden, gc.DeskPort, gc.CaptivePort)
		}
		st, err := guest.OpenStore(guestStoreDir())
		if err != nil {
			fmt.Println(err)
			return
		}
		allow, pend := st.Snapshot()
		fmt.Println("pending:", len(pend))
		for _, p := range pend {
			fmt.Printf("  %s %s\n", p.Code, p.MAC)
		}
		fmt.Println("allow:", len(allow))
		for _, a := range allow {
			fmt.Printf("  %s until %s\n", a.MAC, a.ExpiresAt.Format(time.RFC3339))
		}
	case "enable":
		gc, _ := guest.LoadConfig(guestConfigPath())
		gc.Enabled = true
		gc.Hidden = true
		if gc.DeskPort == 0 {
			gc.DeskPort = 7880
		}
		if gc.CaptivePort == 0 {
			gc.CaptivePort = 7881
		}
		for _, a := range args[1:] {
			if strings.HasPrefix(a, "--ssid=") {
				gc.SSID = strings.TrimPrefix(a, "--ssid=")
			}
			if strings.HasPrefix(a, "--pin=") {
				gc.DeskPIN = strings.TrimPrefix(a, "--pin=")
			}
			if strings.HasPrefix(a, "--psk=") {
				gc.PSK = strings.TrimPrefix(a, "--psk=")
			}
			if a == "--hidden=0" {
				gc.Hidden = false
			}
		}
		if gc.SSID == "" {
			gc.SSID = "Guest"
		}
		if gc.PSK == "" {
			b := make([]byte, 12)
			_, _ = rand.Read(b)
			gc.PSK = hex.EncodeToString(b)
		}
		if gc.DeskPIN == "" && gc.DeskPINHash == "" {
			fmt.Fprintln(os.Stderr, "need --pin=")
			os.Exit(1)
		}
		if err := guest.SaveConfig(guestConfigPath(), gc); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// reload without plaintext pin
		gc, _ = guest.LoadConfig(guestConfigPath())
		fmt.Println(applyGuestNetwork(gc))
		fmt.Println("guest enabled; restart agent to bind desk/captive")
		fmt.Println("join:", guest.JoinQR(gc.SSID, gc.PSK, gc.Hidden))
	case "disable":
		gc, err := guest.LoadConfig(guestConfigPath())
		if err != nil {
			gc = guest.Config{}
		}
		gc.Enabled = false
		_ = guest.SaveConfig(guestConfigPath(), gc)
		fmt.Println("guest disabled in config (UCI left in place; remove manually if needed)")
	case "grant":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "guest grant <mac> [minutes]")
			os.Exit(1)
		}
		mins := 10
		if len(args) >= 3 {
			fmt.Sscanf(args[2], "%d", &mins)
		}
		st, _ := guest.OpenStore(guestStoreDir())
		e, err := st.Grant(args[1], time.Duration(mins)*time.Minute, "cli")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = applyGuestFirewallAllow(st)
		fmt.Println(e.MAC, e.ExpiresAt.Format(time.RFC3339))
	case "revoke":
		if len(args) < 2 {
			os.Exit(1)
		}
		st, _ := guest.OpenStore(guestStoreDir())
		_ = st.Revoke(args[1])
		_ = applyGuestFirewallAllow(st)
		fmt.Println("revoked")
	case "join-qr":
		gc, err := guest.LoadConfig(guestConfigPath())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(guest.JoinQR(gc.SSID, gc.PSK, gc.Hidden))
	default:
		fmt.Fprintln(os.Stderr, "unknown guest command")
		os.Exit(1)
	}
}
