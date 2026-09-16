package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/dnsblock"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func handleGuestCB(token string, chat int64, msgID int, data string) {
	// m:guest | m:guest:10m | m:guest:1h | m:guest:24h
	ttl := time.Hour
	if strings.HasPrefix(data, "m:guest:") {
		switch strings.TrimPrefix(data, "m:guest:") {
		case "10m":
			ttl = 10 * time.Minute
		case "1h":
			ttl = time.Hour
		case "24h":
			ttl = 24 * time.Hour
		case "7d":
			ttl = 7 * 24 * time.Hour
		}
	} else {
		// menu
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("10 min", "m:guest:10m", ""), btn("1 hour", "m:guest:1h", "")},
			{btn("24 hours", "m:guest:24h", ""), btn("7 days", "m:guest:7d", "")},
			{btn(T("main_menu"), "m:menu", "primary")},
		}}
		reply(token, chat, msgID, "⏱ <b>Guest access</b>\nChoose TTL for shared <code>guest</code> user:", kb)
		return
	}
	g, link, err := vpn.IssueGuestAccess(ttl, "tg")
	if err != nil {
		reply(token, chat, msgID, "❌ "+esc(err.Error()), mainKeyboard())
		return
	}
	body := fmt.Sprintf("⏱ <b>Guest</b> until <code>%s</code>\n", time.Unix(g.Expires, 0).UTC().Format(time.RFC3339))
	if link != "" {
		body += "<pre><code>" + esc(link) + "</code></pre>"
	} else {
		body += "Open Users → guest → Access for links."
	}
	reply(token, chat, msgID, body, mainKeyboard())
}

func handleDNSCB(token string, chat int64, msgID int, data string) {
	if data == "m:dns" {
		cat := dnsblock.Catalog()
		var b strings.Builder
		b.WriteString("🛡 <b>DNS block lists</b>\n")
		rows := [][]map[string]any{}
		for i, e := range cat {
			if i >= 12 {
				break
			}
			mark := "☐"
			if e.Enabled {
				mark = "☑"
			}
			b.WriteString(fmt.Sprintf("%s <code>%s</code>\n", mark, e.ID))
			act := "m:dns:off:" + e.ID
			if !e.Enabled {
				act = "m:dns:on:" + e.ID
			}
			rows = append(rows, []map[string]any{btn(mark+" "+e.ID, act, "")})
		}
		rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
		reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": rows})
		return
	}
	parts := strings.Split(data, ":")
	// m:dns:on:id or m:dns:off:id
	if len(parts) >= 4 {
		on := parts[2] == "on"
		id := strings.Join(parts[3:], ":")
		if err := dnsblock.SetEnabled(id, on); err != nil {
			reply(token, chat, msgID, "❌ "+esc(err.Error()), mainKeyboard())
			return
		}
	}
	handleDNSCB(token, chat, msgID, "m:dns")
}

func handleBackupCB(token string, chat int64, msgID int, data string) {
	s := install.LoadBackupSchedule()
	if data == "m:backup" {
		body := fmt.Sprintf("🗓 <b>Backup schedule</b>\nCurrent: <code>%s</code>\nDefault ≈ 04:00 MSK (01:00 UTC).", install.FormatBackupSchedule(s))
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("01:00 UTC", "m:backup:set:1:0", ""), btn("04:00 UTC", "m:backup:set:4:0", "")},
			{btn("22:00 UTC", "m:backup:set:22:0", ""), btn("Run now", "m:backup:run", "")},
			{btn(T("main_menu"), "m:menu", "primary")},
		}}
		reply(token, chat, msgID, body, kb)
		return
	}
	if data == "m:backup:run" {
		// best-effort
		reply(token, chat, msgID, "⏳ Backup requested (systemd start netductor-backup.service)", mainKeyboard())
		return
	}
	if strings.HasPrefix(data, "m:backup:set:") {
		parts := strings.Split(data, ":")
		if len(parts) >= 5 {
			h, _ := strconv.Atoi(parts[3])
			m, _ := strconv.Atoi(parts[4])
			s = install.BackupSchedule{Hour: h, Minute: m, UTC: true}
			if err := install.SaveBackupSchedule(s); err != nil {
				reply(token, chat, msgID, "❌ "+esc(err.Error()), mainKeyboard())
				return
			}
		}
		handleBackupCB(token, chat, msgID, "m:backup")
	}
}
