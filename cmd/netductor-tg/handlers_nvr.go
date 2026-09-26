package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/format"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func nvrHubHTML() string {
	cfg := nvr.LoadConfig()
	cams := nvr.ListCameras()
	last, ok := nvr.LastRetentionReport()
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("🎥 <b>NVR</b>" + nl)
	if getLang() != "en" {
		b.WriteString("<i>📷 камеры · 📍 DHCP · ⚙️ конфиг · 🗜 ротация · 📼 сегменты · ⚡ события · 👁 движение · 📡 go2rtc</i>" + nl)
	} else {
		b.WriteString("<i>📷 cameras · 📍 DHCP leases · ⚙️ config · 🗜 rotate · 📼 segments · ⚡ events · 👁 motion · 📡 go2rtc</i>" + nl)
	}
	b.WriteString("<table bordered striped compact>" + nl)
	b.WriteString("<tr><th>field</th><th>value</th></tr>" + nl)
	b.WriteString(fmt.Sprintf("<tr><td>record</td><td>%v</td></tr>"+nl, cfg.RecordEnabled))
	b.WriteString(fmt.Sprintf("<tr><td>path</td><td><code>%s</code></td></tr>"+nl, esc(cfg.Path)))
	b.WriteString(fmt.Sprintf("<tr><td>retention</td><td>%dd · max %.0fGB</td></tr>"+nl, cfg.RetentionDays, cfg.MaxGB))
	b.WriteString(fmt.Sprintf("<tr><td>cameras</td><td>%d</td></tr>"+nl, len(cams)))
	mc := nvr.LoadMotion()
	b.WriteString(fmt.Sprintf("<tr><td>motion</td><td>%v / in_window=%v</td></tr>"+nl, mc.Enabled, nvr.InMotionWindow(mc, time.Now())))
	if ok {
		b.WriteString(fmt.Sprintf("<tr><td>last rotate</td><td>del=%d kept=%d</td></tr>"+nl, last.Deleted, last.Kept))
	}
	b.WriteString("</table>" + nl)
	if getLang() != "en" {
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(`<tg-button type="callback_data" style="primary" data="m:nvr:cams">📷 Камеры</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:sites">📍 DHCP</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:cfg">⚙️ Конфиг</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:rotate">🗜 Ротация</tg-button>`)
		b.WriteString(`</tg-button-row>` + nl)
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:segs">📼 Сегменты</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:events">⚡ События</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:motion">👁 Движение</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:go2rtc">📡 go2rtc</tg-button>`)
		b.WriteString(`</tg-button-row>` + nl)
	} else {
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(`<tg-button type="callback_data" style="primary" data="m:nvr:cams">📷 Cameras</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:sites">📍 DHCP</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:cfg">⚙️ Config</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:rotate">🗜 Rotate</tg-button>`)
		b.WriteString(`</tg-button-row>` + nl)
		b.WriteString(`<tg-button-row align="left">`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:segs">📼 Segments</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:events">⚡ Events</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:motion">👁 Motion</tg-button>`)
		b.WriteString(`<tg-button type="callback_data" style="link" data="m:nvr:go2rtc">📡 go2rtc</tg-button>`)
		b.WriteString(`</tg-button-row>` + nl)
	}
	return b.String()
}

func nvrKeyboard() map[string]any {
	return navKeyboard("m:tools", parentTools())
}

func nvrSitesKeyboard() map[string]any {
	return navKeyboard("m:nvr", "NVR")
}

func handleNVRCB(token string, chat int64, msgID int, data string) {
	switch {
	case data == "m:nvr" || data == "m:nvr:hub":
		reply(token, chat, msgID, nvrHubHTML(), nvrKeyboard())
	case data == "m:nvr:sites":
		nl := string([]byte{10})
		var b strings.Builder
		b.WriteString("📍 <b>NVR site (edge)</b>" + nl)
		if getLang() != "en" {
			b.WriteString("<i># — DHCP leases с edge</i>" + nl)
		} else {
			b.WriteString("<i># — DHCP leases from edge</i>" + nl)
		}
		var devices []edge.Device
		for _, d := range edge.ListDevices() {
			if d.Status != edge.StatusApproved {
				continue
			}
			devices = append(devices, d)
		}
		b.WriteString("<table bordered striped compact>" + nl + "<tr><th>#</th><th>host</th></tr>" + nl)
		for i, d := range devices {
			if i >= 12 {
				break
			}
			label := d.Hostname
			if label == "" {
				label = d.DeviceID
			}
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td></tr>"+nl, i+1, esc(label)))
		}
		b.WriteString("</table>" + nl)
		b.WriteString(`<tg-button-row align="left">`)
		for i, d := range devices {
			if i >= 12 {
				break
			}
			id := d.DeviceID
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:lease:%s">%d</tg-button>`, id, i+1))
		}
		b.WriteString(`</tg-button-row>` + nl)
		if len(devices) == 0 {
			b.WriteString("<i>(no approved edge)</i>" + nl)
		}
		reply(token, chat, msgID, b.String(), nvrSitesKeyboard())
	case strings.HasPrefix(data, "m:nvr:lease:"):
		did := strings.TrimPrefix(data, "m:nvr:lease:")
		cmdID := edge.EnqueueCmd(did, "dhcp_leases", "")
		if cmdID == "" {
			reply(token, chat, msgID, "enqueue failed (device approved? online?)", nvrSitesKeyboard())
			return
		}
		reply(token, chat, msgID, "leases from <code>"+esc(did)+"</code>...\ncmd <code>"+esc(cmdID)+"</code>", nvrSitesKeyboard())
		go func(chat int64, msgID int, did, cmdID string) {
			res, err := edge.WaitCmdResult(cmdID, 90*time.Second)
			if err != nil {
				reply(token, chat, msgID, "wait: "+esc(err.Error())+"\nTry again when agent is online.", nvrSitesKeyboard())
				return
			}
			raw, _ := res["result"].(string)
			leases := nvr.ParseLeasesResult(raw)
			payload, _ := json.Marshal(leases)
			setState(chat, "nvr_lease_cache", did+"\n"+string(payload))
			var b strings.Builder
			b.WriteString(fmt.Sprintf("<b>Leases · %s</b> (%d)\n", esc(did), len(leases)))
			b.WriteString("<table>\n<tr><th>#</th><th>IP</th><th>host</th></tr>\n")
			nShow := 0
			for i, l := range leases {
				if i >= 15 {
					break
				}
				host := l.Hostname
				if host == "" {
					host = l.MAC
				}
				b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td><td>%s</td></tr>\n", i+1, esc(l.IP), esc(host)))
				nShow++
			}
			b.WriteString("</table>\n")
			if nShow > 0 {
				b.WriteString("<tg-button-row>")
				for i := 0; i < nShow; i++ {
					b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:add:%d">%d</tg-button>`, i, i+1))
				}
				b.WriteString("</tg-button-row>\n")
			}
			if len(leases) == 0 {
				b.WriteString("empty or bad agent payload:\n<pre>" + esc(trunc(raw, 400)) + "</pre>")
			}
			reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": [][]map[string]any{
				{btn(T("back"), "m:nvr:sites", ""), btn(T("main_menu"), "m:menu", "primary")},
			}})
		}(chat, msgID, did, cmdID)
	case strings.HasPrefix(data, "m:nvr:add:"):
		idxStr := strings.TrimPrefix(data, "m:nvr:add:")
		if chatState[chat] != "nvr_lease_cache" || chatExtra[chat] == "" {
			reply(token, chat, msgID, "lease cache expired — load leases again", nvrSitesKeyboard())
			return
		}
		parts := strings.SplitN(chatExtra[chat], "\n", 2)
		if len(parts) != 2 {
			reply(token, chat, msgID, "bad cache", nvrKeyboard())
			return
		}
		did := parts[0]
		var leases []nvr.Lease
		_ = json.Unmarshal([]byte(parts[1]), &leases)
		var idx int
		fmt.Sscanf(idxStr, "%d", &idx)
		if idx < 0 || idx >= len(leases) {
			reply(token, chat, msgID, "bad index", nvrKeyboard())
			return
		}
		l := leases[idx]
		name := l.Hostname
		if name == "" {
			mac := strings.ReplaceAll(l.MAC, ":", "")
			if len(mac) > 6 {
				mac = mac[len(mac)-6:]
			}
			name = "cam-" + mac
		}
		draft, _ := json.Marshal(map[string]string{
			"site_id": did, "mac": l.MAC, "lan_ip": l.IP, "name": name,
		})
		setState(chat, "wait_nvr_pass", string(draft))
		reply(token, chat, msgID, fmt.Sprintf(
			"Camera <b>%s</b>\nIP <code>%s</code> MAC <code>%s</code>\nsite <code>%s</code>\n\nSend <b>RTSP password</b> (Tapo camera account), or <code>-</code> to skip:",
			esc(name), esc(l.IP), esc(l.MAC), esc(did)), nvrKeyboard())
	case data == "m:nvr:cams":
		cams := nvr.ListCameras()
		nl := string([]byte{10})
		var b strings.Builder
		b.WriteString("📷 <b>Cameras</b>" + nl)
		if getLang() != "en" {
			b.WriteString("<i># · P probe · R record · S stop · ◀▶▲▼ PTZ</i>" + nl)
		} else {
			b.WriteString("<i># · P probe · R record · S stop · ◀▶▲▼ PTZ</i>" + nl)
		}
		payload, _ := json.Marshal(cams)
		setState(chat, "nvr_cam_cache", string(payload))
		b.WriteString("<table bordered striped compact>" + nl)
		b.WriteString("<tr><th>#</th><th>name</th><th>ip</th><th>rec</th></tr>" + nl)
		n := len(cams)
		if n > 10 {
			n = 10
		}
		for i := 0; i < n; i++ {
			c := cams[i]
			rec := "—"
			if c.Record {
				rec = "🟢"
			}
			b.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>"+nl,
				i+1, esc(c.Name), esc(c.LANIP), rec))
		}
		b.WriteString("</table>" + nl)
		for i := 0; i < n; i++ {
			b.WriteString(`<tg-button-row align="left">`)
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:probe:%d">P%d</tg-button>`, i, i+1))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:rec:%d">R%d</tg-button>`, i, i+1))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:stop:%d">S%d</tg-button>`, i, i+1))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:ptz:%d:left">%d◀</tg-button>`, i, i+1))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:ptz:%d:right">%d▶</tg-button>`, i, i+1))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:ptz:%d:up">%d▲</tg-button>`, i, i+1))
			b.WriteString(fmt.Sprintf(`<tg-button type="callback_data" style="link" data="m:nvr:ptz:%d:down">%d▼</tg-button>`, i, i+1))
			b.WriteString(`</tg-button-row>` + nl)
		}
		if len(cams) == 0 {
			b.WriteString(T("nvr_empty") + nl)
		}
		reply(token, chat, msgID, b.String(), nvrKeyboard())
	case strings.HasPrefix(data, "m:nvr:probe:"), strings.HasPrefix(data, "m:nvr:rec:"), strings.HasPrefix(data, "m:nvr:stop:"):
		parts := strings.Split(data, ":")
		if len(parts) < 4 {
			reply(token, chat, msgID, "bad", nvrKeyboard())
			return
		}
		action := parts[2]
		var idx int
		fmt.Sscanf(parts[3], "%d", &idx)
		if chatState[chat] != "nvr_cam_cache" || chatExtra[chat] == "" {
			reply(token, chat, msgID, "open Cameras again", nvrKeyboard())
			return
		}
		var cams []nvr.Camera
		if json.Unmarshal([]byte(chatExtra[chat]), &cams) != nil || idx < 0 || idx >= len(cams) {
			reply(token, chat, msgID, "cache", nvrKeyboard())
			return
		}
		c := cams[idx]
		switch action {
		case "probe":
			arg := c.LANIP
			if arg == "" {
				reply(token, chat, msgID, "no lan_ip", nvrKeyboard())
				return
			}
			port := c.RTSPPort
			if port == 0 {
				port = 554
			}
			arg = fmt.Sprintf("%s:%d", arg, port)
			if url := nvr.RTSPURL(c); url != "" {
				arg = url
			}
			cmdID := edge.EnqueueCmd(c.SiteID, "rtsp_probe", arg)
			if cmdID == "" {
				reply(token, chat, msgID, "enqueue failed", nvrKeyboard())
				return
			}
			reply(token, chat, msgID, "probe <code>"+esc(c.Name)+"</code>...", nvrKeyboard())
			go func() {
				res, err := edge.WaitCmdResult(cmdID, 60*time.Second)
				if err != nil {
					reply(token, chat, 0, esc(err.Error()), nvrKeyboard())
					return
				}
				r, _ := res["result"].(string)
				reply(token, chat, 0, "<b>"+esc(c.Name)+"</b>\n<code>"+esc(r)+"</code>", nvrKeyboard())
			}()
		case "rec":
			url := nvr.RTSPURL(c)
			if url == "" || c.SiteID == "" {
				reply(token, chat, msgID, "need RTSP password + site", nvrKeyboard())
				return
			}
			seg := nvr.LoadConfig().SegmentSec
			if seg <= 0 {
				seg = 300
			}
			arg := fmt.Sprintf("%s|%s|%d", c.ID, url, seg)
			cmdID := edge.EnqueueCmd(c.SiteID, "nvr_record_start", arg)
			reply(token, chat, msgID, "start record...", nvrKeyboard())
			go func() {
				res, err := edge.WaitCmdResult(cmdID, 60*time.Second)
				if err != nil {
					reply(token, chat, 0, esc(err.Error()), nvrKeyboard())
					return
				}
				r, _ := res["result"].(string)
				reply(token, chat, 0, "REC "+esc(r), nvrKeyboard())
			}()
		case "stop":
			cmdID := edge.EnqueueCmd(c.SiteID, "nvr_record_stop", c.ID)
			reply(token, chat, msgID, "stop...", nvrKeyboard())
			go func() {
				res, err := edge.WaitCmdResult(cmdID, 60*time.Second)
				if err != nil {
					reply(token, chat, 0, esc(err.Error()), nvrKeyboard())
					return
				}
				r, _ := res["result"].(string)
				reply(token, chat, 0, "STOP "+esc(r), nvrKeyboard())
			}()
		}

	case strings.HasPrefix(data, "m:nvr:ptz:"):
		parts := strings.Split(data, ":")
		if len(parts) < 5 {
			reply(token, chat, msgID, "bad ptz", nvrKeyboard())
			return
		}
		var idx int
		fmt.Sscanf(parts[3], "%d", &idx)
		dir := strings.Join(parts[4:], ":")
		if chatState[chat] != "nvr_cam_cache" || chatExtra[chat] == "" {
			reply(token, chat, msgID, "open Cameras again", nvrKeyboard())
			return
		}
		var cams []nvr.Camera
		if json.Unmarshal([]byte(chatExtra[chat]), &cams) != nil || idx < 0 || idx >= len(cams) {
			reply(token, chat, msgID, "cache", nvrKeyboard())
			return
		}
		c := cams[idx]
		pass := nvr.GetSecret(c.SecretRef)
		if pass == "" {
			pass = nvr.GetSecret(c.ID)
		}
		arg := c.LANIP + "|" + c.RTSPUser + "|" + pass + "|" + dir
		cmdID := edge.EnqueueCmd(c.SiteID, "camera_ptz", arg)
		reply(token, chat, msgID, "PTZ "+esc(dir)+"… (tapo)", nvrKeyboard())
		go func() {
			res, err := edge.WaitCmdResult(cmdID, 30*time.Second)
			if err != nil {
				reply(token, chat, 0, esc(err.Error()), nvrKeyboard())
				return
			}
			r, _ := res["result"].(string)
			reply(token, chat, 0, "<code>"+esc(r)+"</code>", nvrKeyboard())
		}()
	case data == "m:nvr:events":
		var b strings.Builder
		b.WriteString("<b>NVR events</b>\n")
		evs := nvr.ListEventsTail(20)
		if len(evs) == 0 {
			b.WriteString("empty\n")
		}
		for _, e := range evs {
			b.WriteString(fmt.Sprintf("- %d %s %s %s\n", e.TS, esc(e.Type), esc(e.CameraID), esc(e.Detail)))
		}
		reply(token, chat, msgID, b.String(), nvrKeyboard())
	case data == "m:nvr:motion":
		mc := nvr.LoadMotion()
		raw, _ := json.MarshalIndent(mc, "", "  ")
		inw := nvr.InMotionWindow(mc, time.Now())
		r := format.API("nvr-config", raw, catalogLang())
		hint := "in_window="+fmt.Sprintf("%v", inw)
		reply(token, chat, msgID, "👁 <b>Motion</b>\n"+r.HTML+"\n"+hint, nvrKeyboard())
	case data == "m:nvr:cfg":
		cfg := nvr.LoadConfig()
		raw, _ := json.MarshalIndent(cfg, "", "  ")
		r := format.API("nvr-config", raw, catalogLang())
		reply(token, chat, msgID, "⚙️ <b>NVR config</b>\n"+r.HTML, nvrKeyboard())
	case data == "m:nvr:rotate":
		rep, err := nvr.RunRetention(nvr.LoadConfig())
		if err != nil {
			reply(token, chat, msgID, esc(err.Error()), nvrKeyboard())
			return
		}
		reply(token, chat, msgID, fmt.Sprintf("retention done\ndeleted=%d (%d bytes)\nkept=%d total=%d",
			rep.Deleted, rep.DeletedBytes, rep.Kept, rep.TotalBytes), nvrKeyboard())
	case data == "m:nvr:segs":
		files, err := nvr.ListSegmentFiles(nvr.LoadConfig().Path)
		if err != nil {
			reply(token, chat, msgID, esc(err.Error()), nvrKeyboard())
			return
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("<b>Segments</b> (%d)\n", len(files)))
		start := 0
		if len(files) > 12 {
			start = len(files) - 12
		}
		for _, f := range files[start:] {
			b.WriteString(fmt.Sprintf("• %s %s %dB\n", f.ModTime.Format("01-02 15:04"), esc(f.Camera), f.Size))
		}
		if len(files) == 0 {
			b.WriteString("no files yet\n")
		}
		reply(token, chat, msgID, b.String(), nvrKeyboard())
	default:
		reply(token, chat, msgID, nvrHubHTML(), nvrKeyboard())
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func handleNVRMessage(token string, chat int64, text string) bool {
	if chatState[chat] != "wait_nvr_pass" {
		return false
	}
	draftJSON := chatExtra[chat]
	setState(chat, "", "")
	var draft map[string]string
	if json.Unmarshal([]byte(draftJSON), &draft) != nil {
		reply(token, chat, 0, "bad draft", nvrKeyboard())
		return true
	}
	pass := strings.TrimSpace(text)
	c := nvr.Camera{
		SiteID:   draft["site_id"],
		Name:     draft["name"],
		MAC:      draft["mac"],
		LANIP:    draft["lan_ip"],
		RTSPUser: "admin",
		RTSPPath: "/stream1",
		RTSPPort: 554,
		Enabled:  true,
		Record:   true,
		Features: map[string]bool{"ptz": true, "night": true},
	}
	out, err := nvr.UpsertCamera(c)
	if err != nil {
		reply(token, chat, 0, esc(err.Error()), nvrKeyboard())
		return true
	}
	if pass != "" && pass != "-" {
		out.SecretRef = out.ID
		out, _ = nvr.UpsertCamera(out)
		_ = nvr.SetSecret(out.ID, pass)
	}
	if out.MAC != "" && out.LANIP != "" && out.SiteID != "" {
		arg := "mac=" + out.MAC + "|ip=" + out.LANIP + "|name=" + out.Name
		_ = edge.EnqueueCmd(out.SiteID, "dhcp_static", arg)
	}
	reply(token, chat, 0, fmt.Sprintf("camera <b>%s</b> id=<code>%s</code>\nstatic DHCP queued if agent online",
		esc(out.Name), out.ID), nvrKeyboard())
	return true
}
