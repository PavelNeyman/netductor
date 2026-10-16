package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/edgestorage"
)

func handleEdgeStorageCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:edgestor" && !strings.HasPrefix(data, "m:edgestor:") {
		return false
	}
	ru := getLang() != "en"
	nav := map[string]any{"inline_keyboard": [][]map[string]any{
		{btn("⬅️ "+parentRouters(), "m:cat:routers", "primary"), btn(T("main_menu"), "m:menu", "")},
	}}

	if data == "m:edgestor" {
		devs := edge.ListDevices()
		var b strings.Builder
		if ru {
			b.WriteString("💾 <b>USB / storage на edge</b>\nБэкап A+B, restore, mute алертов, offline pack.\nВыберите устройство:\n")
		} else {
			b.WriteString("💾 <b>Edge USB / storage</b>\nBackup A+B, restore, mute alerts, offline pack.\nPick a device:\n")
		}
		var ids []string
		for _, d := range devs {
			if d.DeviceID == "" {
				continue
			}
			ids = append(ids, d.DeviceID)
		}
		if len(ids) == 0 {
			if ru {
				b.WriteString("<i>Нет edge-устройств</i>")
			} else {
				b.WriteString("<i>No edge devices</i>")
			}
			reply(token, chat, msgID, b.String(), nav)
			return true
		}
		b.WriteString("<table bordered striped compact>\n<tr><th>#</th><th>id</th></tr>\n")
		for i, id := range ids {
			if i >= 12 {
				break
			}
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td></tr>\n", i+1, esc(id)))
		}
		b.WriteString("</table>\n")
		b.WriteString(`<tg-button-row align="left">`)
		for i, id := range ids {
			if i >= 12 {
				break
			}
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:edgestor:dev:%s">%d</tg-button>`, id, i+1))
		}
		b.WriteString(`</tg-button-row>`)
		reply(token, chat, msgID, b.String(), nav)
		return true
	}

	rest := strings.TrimPrefix(data, "m:edgestor:")
	if strings.HasPrefix(rest, "dev:") {
		id := strings.TrimPrefix(rest, "dev:")
		var b strings.Builder
		b.WriteString(fmt.Sprintf("💾 <b>%s</b>\n", esc(id)))
		b.WriteString(formatEdgeStorageListHTML(id, ru))
		labels := []struct{ data, en, ru string }{
			{"list:" + id, "📋 List", "📋 Список"},
			{"bak:" + id, "Backup A+B", "Бэкап A+B"},
			{"rst:" + id, "Restore", "Restore"},
			{"fmt:" + id, "Format+restore", "Format+restore"},
			{"mute:" + id, "Mute alerts", "Mute алертов"},
			{"unm:" + id, "Unmute", "Unmute"},
			{"pack:" + id, "Offline pack", "Offline pack"},
		}
		b.WriteString(`<tg-button-row align="left">`)
		for i, L := range labels {
			lab := L.en
			if ru {
				lab = L.ru
			}
			style := "link"
			if i == 1 {
				style = "primary"
			}
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="%s" data="m:edgestor:%s">%s</tg-button>`, style, L.data, lab))
			if i == 3 {
				b.WriteString(`</tg-button-row><tg-button-row align="left">`)
			}
		}
		b.WriteString(`</tg-button-row>`)
		reply(token, chat, msgID, b.String(), nav)
		return true
	}

	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		reply(token, chat, msgID, "❌ bad callback", nav)
		return true
	}
	act, id := parts[0], parts[1]
	var msg string
	switch act {
	case "list":
		msg = fmt.Sprintf("💾 <b>%s</b>\n", esc(id)) + formatEdgeStorageListHTML(id, ru)
	case "bak":
		a := edge.EnqueueCmd(id, "config_backup", "")
		bb := edge.EnqueueCmd(id, "storage_backup", "")
		if ru {
			msg = fmt.Sprintf("✅ Очередь бэкапа\nconfig=<code>%s</code>\nstorage=<code>%s</code>", esc(a), esc(bb))
		} else {
			msg = fmt.Sprintf("✅ Backup queued\nconfig=<code>%s</code>\nstorage=<code>%s</code>", esc(a), esc(bb))
		}
	case "rst":
		cid := edge.EnqueueCmd(id, "storage_restore_remote", "")
		if cid == "" {
			msg = "❌ enqueue failed"
		} else if ru {
			msg = "✅ Restore с primary в очереди: <code>" + esc(cid) + "</code>\nПосле готовности — reboot вручную."
		} else {
			msg = "✅ Remote restore queued: <code>" + esc(cid) + "</code>\nReboot manually when ready."
		}
	case "fmt":
		_ = edge.EnqueueCmd(id, "storage_init_usb", "format=yes")
		cid := edge.EnqueueCmd(id, "storage_restore_remote", "")
		if ru {
			msg = "⚠️ Format + restore в очереди: <code>" + esc(cid) + "</code>"
		} else {
			msg = "⚠️ Format + restore queued: <code>" + esc(cid) + "</code>"
		}
	case "mute":
		_ = edge.SetUSBAlertMute(id, true)
		if ru {
			msg = "✅ USB-алерты выключены"
		} else {
			msg = "✅ mute USB alerts"
		}
	case "unm":
		_ = edge.SetUSBAlertMute(id, false)
		if ru {
			msg = "✅ USB-алерты включены"
		} else {
			msg = "✅ unmute USB alerts"
		}
	case "pack":
		dir, err := edgestorage.WriteOfflinePack(id, "/var/lib/netductor/edge-offline/"+id)
		if err != nil {
			msg = "❌ " + esc(err.Error())
		} else if ru {
			msg = "✅ Offline pack: <code>" + esc(dir) + "</code>"
		} else {
			msg = "✅ Offline pack: <code>" + esc(dir) + "</code>"
		}
	default:
		msg = "❌ unknown"
	}
	reply(token, chat, msgID, msg, nav)
	return true
}

func formatEdgeStorageListHTML(id string, ru bool) string {
	var b strings.Builder
	list, _ := edgestorage.ListManifests(id)
	if ru {
		b.WriteString("<b>Манифесты на primary</b>\n")
	} else {
		b.WriteString("<b>Manifests on primary</b>\n")
	}
	if len(list) == 0 {
		b.WriteString("<i>—</i>\n")
	} else {
		b.WriteString("<table bordered striped compact>\n<tr><th>#</th><th>kind</th><th>file</th><th>when</th></tr>\n")
		for i, m := range list {
			if i >= 8 {
				b.WriteString(fmt.Sprintf("<tr><td colspan=\"4\">… +%d</td></tr>\n", len(list)-8))
				break
			}
			when := m.Created
			if len(when) > 19 {
				when = when[:19]
			}
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>\n",
				i+1, esc(m.Kind), esc(shortName(m.File)), esc(when)))
		}
		b.WriteString("</table>\n")
	}
	ups := edge.ListBackups(id)
	if ru {
		b.WriteString("<b>Загрузки с агента</b>\n")
	} else {
		b.WriteString("<b>Agent uploads</b>\n")
	}
	if len(ups) == 0 {
		b.WriteString("<i>—</i>\n")
	} else {
		b.WriteString("<table bordered striped compact>\n<tr><th>#</th><th>name</th><th>size</th></tr>\n")
		for i, u := range ups {
			if i >= 8 {
				break
			}
			name, _ := u["name"].(string)
			sz := u["size"]
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td><td>%v</td></tr>\n", i+1, esc(shortName(name)), sz))
		}
		b.WriteString("</table>\n")
	}
	return b.String()
}

func shortName(s string) string {
	if len(s) <= 28 {
		return s
	}
	return s[:12] + "…" + s[len(s)-12:]
}
