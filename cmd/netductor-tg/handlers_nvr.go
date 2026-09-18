package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/nvr"
)

func nvrHubHTML() string {
	cfg := nvr.LoadConfig()
	cams := nvr.ListCameras()
	last, ok := nvr.LastRetentionReport()
	var b strings.Builder
	b.WriteString("<b>🎥 NVR</b>\n")
	b.WriteString(fmt.Sprintf("record=%v path=<code>%s</code>\n", cfg.RecordEnabled, esc(cfg.Path)))
	b.WriteString(fmt.Sprintf("retention: %dd · max %.0fGB · free≥%.0fGB · segment %ds\n",
		cfg.RetentionDays, cfg.MaxGB, cfg.MinFreeGB, cfg.SegmentSec))
	b.WriteString(fmt.Sprintf("cameras: <b>%d</b>\n", len(cams)))
	if ok {
		b.WriteString(fmt.Sprintf("last rotate: deleted=%d bytes=%d kept=%d\n", last.Deleted, last.DeletedBytes, last.Kept))
	}
	for i, c := range cams {
		if i >= 8 {
			b.WriteString("…\n")
			break
		}
		b.WriteString(fmt.Sprintf("• <code>%s</code> %s (%s) %s\n", c.ID, esc(c.Name), esc(c.SiteID), c.LANIP))
	}
	return b.String()
}

func nvrKeyboard() map[string]any {
	return map[string]any{
		"inline_keyboard": [][]map[string]any{
			{btn("📋 Cameras", "m:nvr:cams", ""), btn("⚙️ Config", "m:nvr:cfg", "")},
			{btn("🔄 Rotate now", "m:nvr:rotate", ""), btn("📁 Segments", "m:nvr:segs", "")},
			{btn(T("back"), "m:tools", ""), btn(T("main_menu"), "m:menu", "primary")},
		},
	}
}

func handleNVRCB(token string, chat int64, msgID int, data string) {
	switch {
	case data == "m:nvr" || data == "m:nvr:hub":
		reply(token, chat, msgID, nvrHubHTML(), nvrKeyboard())
	case data == "m:nvr:cams":
		cams := nvr.ListCameras()
		var b strings.Builder
		b.WriteString("<b>Cameras</b>\n")
		if len(cams) == 0 {
			b.WriteString("empty — use CLI: <code>netductor nvr cameras add …</code>\n")
		}
		for _, c := range cams {
			b.WriteString(fmt.Sprintf("• <b>%s</b> id=<code>%s</code>\nsite=%s mac=%s ip=%s rec=%v\n",
				esc(c.Name), c.ID, esc(c.SiteID), c.MAC, c.LANIP, c.Record))
		}
		reply(token, chat, msgID, b.String(), nvrKeyboard())
	case data == "m:nvr:cfg":
		cfg := nvr.LoadConfig()
		raw, _ := json.MarshalIndent(cfg, "", "  ")
		reply(token, chat, msgID, "<b>NVR config</b>\n<pre>"+esc(string(raw))+"</pre>\n"+
			"Change: <code>netductor nvr config set retention_days=7 max_gb=40</code>", nvrKeyboard())
	case data == "m:nvr:rotate":
		rep, err := nvr.RunRetention(nvr.LoadConfig())
		if err != nil {
			reply(token, chat, msgID, "⚠️ "+esc(err.Error()), nvrKeyboard())
			return
		}
		reply(token, chat, msgID, fmt.Sprintf("🔄 retention done\ndeleted=%d (%d bytes)\nkept=%d total=%d",
			rep.Deleted, rep.DeletedBytes, rep.Kept, rep.TotalBytes), nvrKeyboard())
	case data == "m:nvr:segs":
		files, err := nvr.ListSegmentFiles(nvr.LoadConfig().Path)
		if err != nil {
			reply(token, chat, msgID, "⚠️ "+esc(err.Error()), nvrKeyboard())
			return
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("<b>Segments</b> (%d)\n", len(files)))
		// show newest last 12
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
