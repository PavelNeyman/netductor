package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/notify"
)

func runTGAlerts(args []string) {
	if len(args) < 1 || args[0] == "help" || args[0] == "-h" {
		fmt.Print(`netductor tg-alerts status [--json]
netductor tg-alerts set <chat_id>
netductor tg-alerts clear
netductor tg-alerts test

Channel id is usually -100… (bot must be channel admin).
`)
		os.Exit(2)
	}
	switch args[0] {
	case "status", "show":
		st := notify.GetAlertsRoutingStatus()
		if len(args) > 1 && args[1] == "--json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(st)
			return
		}
		fmt.Printf("mode=%s channel_set=%v alerts_chat_id=%s admin=%s\n",
			st.Mode, st.ChannelSet, st.AlertsChatID, st.AdminChatID)
		if st.Hint != "" {
			fmt.Println(st.Hint)
		}
	case "set":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor tg-alerts set <chat_id>")
			os.Exit(2)
		}
		if err := notify.SetAlertsChatID(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok", args[1])
		fmt.Println("restart bot: systemctl restart netductor-telegram-bot")
	case "clear":
		if err := notify.ClearAlertsChatID(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("cleared")
	case "test":
		notify.AlertOnce("test:alerts-channel", "🔔 test alert routing (netductor tg-alerts test)")
		_ = notify.FlushAlerts(true)
		fmt.Println("queued test alert →", notify.AlertsChatID())
	default:
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		os.Exit(2)
	}
}
