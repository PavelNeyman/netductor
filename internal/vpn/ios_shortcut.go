package vpn

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// BuildIOSShortcutOpenURL returns an unsigned Apple Shortcuts workflow (XML plist)
// that only runs "Open URL". Filename should end with .shortcut.
//
// Modern iOS (15+) typically requires a signed file (AEA). Unsigned files may import
// only after Settings → Shortcuts → Allow Untrusted / Private Sharing, or after
// signing on a Mac: shortcuts sign --mode anyone -i in.shortcut -o out.shortcut
// (needs iCloud login on that Mac).
func BuildIOSShortcutOpenURL(name, openURL string) []byte {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "netductor"
	}
	openURL = strings.TrimSpace(openURL)
	uid := shortcutUUID()
	// Escape XML special chars in URL and name
	esc := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, `"`, "&quot;")
		return s
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0"><dict>` + "\n")
	b.WriteString(`<key>WFWorkflowClientVersion</key><string>1300.0.0</string>` + "\n")
	b.WriteString(`<key>WFWorkflowClientRelease</key><string>18.0</string>` + "\n")
	b.WriteString(`<key>WFWorkflowMinimumClientVersion</key><integer>900</integer>` + "\n")
	b.WriteString(`<key>WFWorkflowMinimumClientVersionString</key><string>900</string>` + "\n")
	b.WriteString(`<key>WFWorkflowName</key><string>` + esc(name) + `</string>` + "\n")
	b.WriteString(`<key>WFWorkflowIcon</key><dict>`)
	b.WriteString(`<key>WFWorkflowIconStartColor</key><integer>4282601983</integer>`)
	b.WriteString(`<key>WFWorkflowIconGlyphNumber</key><integer>59511</integer>`)
	b.WriteString(`</dict>` + "\n")
	b.WriteString(`<key>WFWorkflowImportQuestions</key><array/>` + "\n")
	b.WriteString(`<key>WFWorkflowTypes</key><array/>` + "\n")
	b.WriteString(`<key>WFWorkflowHasOutputFallback</key><false/>` + "\n")
	b.WriteString(`<key>WFWorkflowHasShortcutInputVariables</key><false/>` + "\n")
	b.WriteString(`<key>WFWorkflowInputContentItemClasses</key><array>`)
	b.WriteString(`<string>WFStringContentItem</string><string>WFURLContentItem</string>`)
	b.WriteString(`</array>` + "\n")
	b.WriteString(`<key>WFWorkflowOutputContentItemClasses</key><array/>` + "\n")
	b.WriteString(`<key>WFWorkflowActions</key><array>` + "\n")
	b.WriteString(`<dict>`)
	b.WriteString(`<key>WFWorkflowActionIdentifier</key><string>is.workflow.actions.openurl</string>`)
	b.WriteString(`<key>WFWorkflowActionParameters</key><dict>`)
	b.WriteString(`<key>UUID</key><string>` + uid + `</string>`)
	b.WriteString(`<key>WFInput</key><string>` + esc(openURL) + `</string>`)
	b.WriteString(`</dict></dict>` + "\n")
	b.WriteString(`</array>` + "\n")
	b.WriteString(`</dict></plist>` + "\n")
	return []byte(b.String())
}

// BuildIOSShortcutKaringImport builds Open URL → karing://install-config?...
func BuildIOSShortcutKaringImport(profile, yamlURL string) (filename string, data []byte) {
	profile = NormalizeKaringProfile(profile)
	name := "nd-karing-" + profile
	deep := fmt.Sprintf("karing://install-config?url=%s&name=%s",
		pathEscapeQuery(yamlURL), pathEscapeQuery("nd-"+profile))
	return name + ".shortcut", BuildIOSShortcutOpenURL(name, deep)
}

// BuildIOSShortcutKaringReconnect opens karing://reconnect.
func BuildIOSShortcutKaringReconnect() (filename string, data []byte) {
	return "nd-karing-reconnect.shortcut", BuildIOSShortcutOpenURL("nd-karing-reconnect", "karing://reconnect")
}

func pathEscapeQuery(s string) string {
	return url.QueryEscape(s)
}

func shortcutUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	// RFC 4122 version 4
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return strings.ToUpper(fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32]))
}
