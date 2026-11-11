package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"testing"
)

func TestAppWidgetHTMLSizeLimits(t *testing.T) {
	html := appWidgetHTML()
	raw := len(html)
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(html)); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	comp := buf.Len()
	t.Logf("app HTML raw=%d zlib9=%d limit raw=32768 comp=4096", raw, comp)
	if raw > 32768 {
		t.Fatalf("raw %d > 32768", raw)
	}
	if comp > 4096 {
		t.Fatalf("compressed %d > 4096 — Telegram will reject sendWidget", comp)
	}
}

func TestAppShellHTMLSizes(t *testing.T) {
	for _, tc := range []struct{ name string; html string }{
		{"hub", appWidgetHTMLHub()},
		{"spark", appWidgetHTMLSpark()},
	} {
		var buf bytes.Buffer
		w, _ := zlib.NewWriterLevel(&buf, 9)
		_, _ = w.Write([]byte(tc.html))
		_ = w.Close()
		comp := buf.Len()
		t.Logf("%s raw=%d zlib=%d", tc.name, len(tc.html), comp)
		if len(tc.html) > 32768 || comp > 4096 {
			t.Fatalf("%s over limit raw=%d comp=%d", tc.name, len(tc.html), comp)
		}
	}
}

func TestAppHomeChatDataSize(t *testing.T) {
	cd, err := buildAppHome()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("home chat_data len=%d data=%s", len(cd), cd)
	if len(cd) > 1024 {
		t.Fatalf("home chat_data %d > 1024", len(cd))
	}
}

func TestAppScreensChatData(t *testing.T) {
	screens := []string{"home", "status", "fleet", "users", "nodes", "channel", "versions",
		"secondary", "probes", "netmap", "digest", "operator", "addons", "backup", "ssh",
		"devices", "sites", "lampac", "sessions", "audit", "actions", "routers", "canary", "svcpaths", "pending", "seclist"}
	for _, s := range screens {
		cd, err := buildAppScreen(s)
		if err != nil {
			t.Logf("WARN %s build err: %v", s, err)
			continue
		}
		if len(cd) > 1024 {
			t.Errorf("%s chat_data %d > 1024", s, len(cd))
		} else {
			t.Logf("OK %s len=%d", s, len(cd))
		}
	}
	_ = fmt.Sprintf
}

func TestSPAContentRoutingMaps(t *testing.T) {
	for _, id := range []string{"status", "users", "channel", "fleet", "ssh"} {
		if !spaContentScreens[id] && spaScreenAlias[id] == "" {
			// ssh is content; sshhosts is alias
			if id != "ssh" {
				t.Fatalf("missing spa content %s", id)
			}
		}
	}
	if spaScreenAlias["sshhosts"] != "ssh" {
		t.Fatalf("sshhosts alias: %q", spaScreenAlias["sshhosts"])
	}
	if !spaContentScreens["status"] {
		t.Fatal("status should be SPA")
	}
}

func TestSPAMenuParityScreensBuild(t *testing.T) {
	screens := []string{
		"home", "status", "users", "ulist", "fleet", "tools", "operator", "help",
		"channel", "netmap", "actions", "sessions", "digest", "incident", "reboot",
		"nodes", "secondary", "addons", "seclist",
		"edge", "routers", "sites", "pending",
		"media", "lampac", "canary", "nvr",
		"data", "backup", "logs", "dns",
		"adv", "probes", "stack", "doctor", "certs", "fw", "cleanup",
		"versions", "ssh", "git",
		"session", "alerts", "admin", "audit", "svcpaths",
		"devices",
	}
	for _, s := range screens {
		cd, err := buildAppScreen(s)
		if err != nil {
			t.Errorf("%s: %v", s, err)
			continue
		}
		if len(cd) > 1024 {
			t.Errorf("%s chat_data %d > 1024", s, len(cd))
		}
	}
}

func TestAppInfoHTMLSize(t *testing.T) {
	for _, tc := range []struct{ name, html string }{
		{"info", appInfoHTML()},
		{"info-spark", appInfoHTMLSpark()},
	} {
		var buf bytes.Buffer
		w, _ := zlib.NewWriterLevel(&buf, 9)
		_, _ = w.Write([]byte(tc.html))
		_ = w.Close()
		if len(tc.html) > 32768 || buf.Len() > 4096 {
			t.Fatalf("%s raw=%d zlib=%d", tc.name, len(tc.html), buf.Len())
		}
		t.Logf("%s raw=%d zlib=%d", tc.name, len(tc.html), buf.Len())
	}
}

func TestDualFlagOffByDefault(t *testing.T) {
	if dualEnabled() {
		t.Fatal("dual must default off without NETDUCTOR_TG_DUAL=1")
	}
}

func TestBuildInfoDataSize(t *testing.T) {
	cd, err := buildInfoData("home")
	if err != nil {
		t.Fatal(err)
	}
	if len(cd) > 1024 {
		t.Fatalf("info chat_data %d", len(cd))
	}
}
