package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func nvrHubHTML() string {
	cfg := nvr.LoadConfig()
	cams := nvr.ListCameras()
	last, ok := nvr.LastRetentionReport()
	var b strings.Builder
	b.WriteString("<b>NVR</b>\n")
	b.WriteString(fmt.Sprintf("record=%v path=<code>%s</code>\n", cfg.RecordEnabled, esc(cfg.Path)))
	b.WriteString(fmt.Sprintf("retention: %dd · max %.0fGB · free>=%.0fGB · segment %ds\n",
		cfg.RetentionDays, cfg.MaxGB, cfg.MinFreeGB, cfg.SegmentSec))
	b.WriteString(fmt.Sprintf("cameras: <b>%d</b>\n", len(cams)))
	mc := nvr.LoadMotion()
	b.WriteString(fmt.Sprintf("motion: enabled=%v in_window=%v\n", mc.Enabled, nvr.InMotionWindow(mc, time.Now())))
	if ok {
		b.WriteString(fmt.Sprintf("last rotate: deleted=%d bytes=%d kept=%d\n", last.Deleted, last.DeletedBytes, last.Kept))
	}
	for i, c := range cams {
		if i >= 8 {
			b.WriteString("...\n")
			break
		}
		b.WriteString(fmt.Sprintf("• <code>%s</code> %s (%s) %s\n", c.ID, esc(c.Name), esc(c.SiteID), c.LANIP))
	}
	return b.String()
}

func nvrKeyboard() map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn("Cameras", "m:nvr:cams", ""), btn("From leases", "m:nvr:sites", "")},
			{btn("Config", "m:nvr:cfg", ""), btn("Rotate", "m:nvr:rotate", "")},
			{btn("Segments", "m:nvr:segs", ""), btn("Events", "m:nvr:events", "")},
			{btn("Motion", "m:nvr:motion", ""), btn("go2rtc", "m:nvr:go2rtc", "")},
			{btn(T("back"), "m:tools", ""), btn(T("main_menu"), "m:menu", "primary")},
		},
	}
}

func nvrSitesKeyboard() map[string]any {
	rows := [][]map[string]any{}
	for _, d := range edge.ListDevices() {
		if d.Status != edge.StatusApproved {
			continue
		}
		label := d.Hostname
		if label == "" {
			label = d.DeviceID
		}
		if len(label) > 28 {
			label = label[:28]
		}
		id := d.DeviceID
		if len(id) > 40 {
			id = id[:40]
		}
		rows = append(rows, []map[string]any{btn(label, "m:nvr:lease:"+id, "")})
		if len(rows) >= 12 {
			break
		}
	}
	if len(rows) == 0 {
		rows = append(rows, []map[string]any{btn("(no approved edge)", "m:nvr", "")})
	}
	rows = append(rows, []map[string]any{btn(T("back"), "m:nvr", ""), btn(T("main_menu"), "m:menu", "primary")})
	return map[string]any{"inline_keyboard": rows}
}

func handleNVRCB(token string, chat int64, msgID int, data string) {
	switch {
	case data == "m:nvr" || data == "m:nvr:hub":
		reply(token, chat, msgID, nvrHubHTML(), nvrKeyboard())
	case data == "m:nvr:sites":
		reply(token, chat, msgID, "<b>NVR site (edge)</b>\nSelect device to load DHCP leases:", nvrSitesKeyboard())
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
			b.WriteString(fmt.Sprintf("<b>Leases · %s</b> (%d)\nTap to add camera:\n", esc(did), len(leases)))
			rows := [][]map[string]any{}
			for i, l := range leases {
				if i >= 15 {
					b.WriteString("...\n")
					break
				}
				host := l.Hostname
				if host == "" {
					host = l.MAC
				}
				b.WriteString(fmt.Sprintf("%d. <code>%s</code> %s %s\n", i+1, esc(l.IP), esc(l.MAC), esc(host)))
				rows = append(rows, []map[string]any{btn(fmt.Sprintf("%d · %s", i+1, trunc(host, 20)), fmt.Sprintf("m:nvr:add:%d", i), "")})
			}
			if len(leases) == 0 {
				b.WriteString("empty or bad agent payload:\n<pre>" + esc(trunc(raw, 400)) + "</pre>")
			}
			rows = append(rows, []map[string]any{btn(T("back"), "m:nvr:sites", ""), btn(T("main_menu"), "m:menu", "primary")})
			reply(token, chat, msgID, b.String(), map[string]any{"inline_keyboard": rows})
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
		var b strings.Builder
		b.WriteString("<b>Cameras</b>\n")
		rows := [][]map[string]any{}
		if len(cams) == 0 {
			b.WriteString("empty — From leases or CLI add\n")
		}
		payload, _ := json.Marshal(cams)
		setState(chat, "nvr_cam_cache", string(payload))
		for i, c := range cams {
			if i >= 10 {
				b.WriteString("...\n")
				break
			}
			b.WriteString(fmt.Sprintf("%d. <b>%s</b> <code>%s</code>\n   %s · %s · rec=%v\n",
				i+1, esc(c.Name), c.ID, esc(c.SiteID), c.LANIP, c.Record))
			rows = append(rows, []map[string]any{
				btn(fmt.Sprintf("P%d", i+1), fmt.Sprintf("m:nvr:probe:%d", i), ""),
				btn(fmt.Sprintf("R%d", i+1), fmt.Sprintf("m:nvr:rec:%d", i), ""),
				btn(fmt.Sprintf("S%d", i+1), fmt.Sprintf("m:nvr:stop:%d", i), ""),
			})
			rows = append(rows, []map[string]any{
				btn(fmt.Sprintf("%d◀", i+1), fmt.Sprintf("m:nvr:ptz:%d:left", i), ""),
				btn(fmt.Sprintf("%d▶", i+1), fmt.Sprintf("m:nvr:ptz:%d:right", i), ""),
				btn(fmt.Sprintf("%d▲", i+1), fmt.Sprintf("m:nvr:ptz:%d:up", i), ""),
				btn(fmt.Sprintf("%d▼", i+1), fmt.Sprintf("m:nvr:ptz:%d:down", i), ""),
			})
			rows = append(rows, []map[string]any{
				btn(fmt.Sprintf("%d night", i+1), fmt.Sprintf("m:nvr:ptz:%d:night:auto", i), ""),
				btn(fmt.Sprintf("%d priv", i+1), fmt.Sprintf("m:nvr:ptz:%d:privacy:off", i), ""),
				btn(fmt.Sprintf("%d cal", i+1), fmt.Sprintf("m:nvr:ptz:%d:calibrate", i), ""),
			})
		}
		rows = append(rows, []map[string]any{btn(T("back"), "m:nvr", ""), btn(T("main_menu"), "m:menu", "primary")})
		reply(token, chat, msgID, b.String()+"\nP=probe R=record S=stop", map[string]any{"inline_keyboard": rows})
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
		reply(token, chat, msgID, "<b>Motion / schedule</b>\n<pre>"+esc(string(raw))+"</pre>\nin_window="+fmt.Sprintf("%v", inw)+"\nCLI: <code>netductor nvr motion set enabled=true timezone=Europe/Moscow</code>", nvrKeyboard())
	case data == "m:nvr:cfg":
		cfg := nvr.LoadConfig()
		raw, _ := json.MarshalIndent(cfg, "", "  ")
		reply(token, chat, msgID, "<b>NVR config</b>\n<pre>"+esc(string(raw))+"</pre>\n"+
			"<code>netductor nvr config set retention_days=7 max_gb=40</code>", nvrKeyboard())
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
