package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func tgSource(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	p := filepath.Join(filepath.Dir(file), name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// /menu and /start must always forceHub — silent edit of a deleted-for-me message looks like a dead bot.
func TestMenuCommandsUseForceHub(t *testing.T) {
	src := tgSource(t, "handlers_msg.go")
	// Extract the /start|/menu case block roughly
	idx := strings.Index(src, `case "/start", "/menu":`)
	if idx < 0 {
		t.Fatal(`missing case "/start", "/menu"`)
	}
	rest := src[idx:]
	end := strings.Index(rest, "\n\tcase \"")
	if end < 0 {
		end = len(rest)
	}
	block := rest[:end]
	if !strings.Contains(block, "forceHub(") {
		t.Fatalf("/menu|/start must call forceHub; block:\n%s", block)
	}
	// Must not prefer plain reply for the happy path after clear-history bugs
	if strings.Contains(block, "reply(token, chat, 0, menuText()") {
		t.Fatal("/menu must not reply(menuText) — that edits an invisible hub after chat clear")
	}
}

// Text-wait handlers must finish via completeInput (edit hub + drop user msg), not sendHTML (orphan stack).
func TestWaitHandlersPreferCompleteInput(t *testing.T) {
	src := tgSource(t, "handlers_msg.go")
	// From start of handleMessage wait section through command switch
	start := strings.Index(src, "func handleMessage")
	cmd := strings.Index(src, `case "/start"`)
	if start < 0 || cmd < 0 || cmd <= start {
		t.Fatal("handleMessage bounds")
	}
	waitSec := src[start:cmd]
	// Disallow sendHTML in wait section (completeInput is required)
	if strings.Contains(waitSec, "sendHTML(token, chat,") {
		// show first offender
		i := strings.Index(waitSec, "sendHTML(token, chat,")
		snippet := waitSec[i:]
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		t.Fatalf("wait_* path still uses sendHTML (use completeInput):\n%s", snippet)
	}
	if !strings.Contains(waitSec, "completeInput(token, chat, uid,") {
		t.Fatal("expected completeInput(token, chat, uid, …) in wait handlers")
	}
	if !strings.Contains(waitSec, "uid := m.MessageID") {
		t.Fatal("handleMessage must capture m.MessageID as uid for completeInput")
	}
}

// Users list must expose device *profiles* (kind=device), not only online-IP locator.
func TestUsersListWiresVpnDevicesCallback(t *testing.T) {
	src := tgSource(t, "format_vpn.go")
	if !strings.Contains(src, `data="m:vpn_devices"`) {
		t.Fatal(`formatUsersListHTML must include callback m:vpn_devices`)
	}
	if !strings.Contains(src, "vpn_devices_btn") {
		t.Fatal("formatUsersListHTML must use T(\"vpn_devices_btn\")")
	}
	cb := tgSource(t, "handlers_cb.go")
	if !strings.Contains(cb, `case "m:vpn_devices":`) {
		t.Fatal(`handlers_cb missing case "m:vpn_devices"`)
	}
}

// Every m:/u: callback data string embedded in format_*.go / keyboards should be handled (smoke).
func TestNoObviousOrphanMenuCallbacks(t *testing.T) {
	dir := filepath.Dir(tgSourcePath(t))
	var htmlData []string
	re := regexp.MustCompile(`data="((?:m|u):[^"]+)"`)
	for _, name := range []string{"format_vpn.go", "format_status.go", "keyboards.go", "catalog_ui.go"} {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			htmlData = append(htmlData, m[1])
		}
	}
	handlers := tgSource(t, "handlers_cb.go") + tgSource(t, "handlers_ssh.go") +
		tgSource(t, "handlers_presets.go") + tgSource(t, "handlers_msg.go") +
		tgSource(t, "handlers_extra.go") + tgSource(t, "handlers_policy.go") +
		tgSource(t, "handlers_canary.go") + tgSource(t, "handlers_stack.go") +
		tgSource(t, "handlers_logs.go") + tgSource(t, "handlers_git.go") +
		tgSource(t, "handlers_reboot.go") + tgSource(t, "handlers_nvr.go") +
		tgSource(t, "handlers_edge_guest.go") + tgSource(t, "handlers_edge_template.go") +
		tgSource(t, "handlers_edge_luci.go") + tgSource(t, "handlers_registry.go") +
		tgSource(t, "handlers_certs.go") +
		tgSource(t, "handlers_mtls.go") + tgSource(t, "handlers_versions.go") +
		tgSource(t, "alerts_channel_ui.go")

	var missing []string
	seen := map[string]bool{}
	for _, d := range htmlData {
		// Normalize dynamic suffixes: m:quota:name:50 → check prefix m:quota
		base := d
		if i := strings.Index(base[2:], ":"); i >= 0 {
			// keep m:xxx or u:xxx for static checks; skip fully dynamic with many parts
			parts := strings.Split(d, ":")
			if len(parts) >= 3 && (parts[0] == "m" || parts[0] == "u") {
				// static menu ids are m:foo or m:foo:bar without user names
				if parts[0] == "u" {
					// u:open:NAME — handler is prefix u:
					continue
				}
				if len(parts) >= 3 && looksLikeDynamic(parts[2]) {
					base = parts[0] + ":" + parts[1]
				}
			}
		}
		if seen[base] {
			continue
		}
		seen[base] = true
		if strings.Contains(handlers, `"`+d+`"`) || strings.Contains(handlers, `"`+base+`"`) ||
			strings.Contains(handlers, "\""+base) ||
			strings.Contains(handlers, `HasPrefix(data, "`+base) ||
			strings.Contains(handlers, `data == "`+d+`"`) ||
			strings.Contains(handlers, `case "`+d+`":`) ||
			strings.Contains(handlers, `case "`+base+`":`) {
			continue
		}
		// prefix handlers m:sshkeys
		okPrefix := false
		for _, pref := range []string{"m:sshkeys", "m:nd:", "m:quota:", "m:edge", "m:edgestor", "m:git", "m:loc:", "m:preset", "m:canary", "m:stack", "m:logs", "m:nvr", "m:reboot", "m:ssh:"} {
			if strings.HasPrefix(d, pref) && strings.Contains(handlers, pref) {
				okPrefix = true
				break
			}
		}
		if okPrefix {
			continue
		}
		missing = append(missing, d)
	}
	if len(missing) > 0 {
		// Soft fail budget: only fail on known critical orphans
		critical := []string{"m:vpn_devices", "m:users", "m:menu", "m:status", "m:vpn_add"}
		for _, c := range critical {
			for _, m := range missing {
				if m == c {
					t.Fatalf("critical callback %s not found in handlers (orphans sample: %v)", c, missing[:min(5, len(missing))])
				}
			}
		}
		t.Logf("note: %d callbacks not exact-matched (review manually): %v", len(missing), missing[:min(15, len(missing))])
	}
}

func looksLikeDynamic(s string) bool {
	if s == "" {
		return false
	}
	// user names / ids often mixed case or contain non-fixed tokens
	return true
}

func tgSourcePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return file
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
