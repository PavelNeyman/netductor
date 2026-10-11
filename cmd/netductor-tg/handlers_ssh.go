package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/integrity"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

var sshWizardSelected = map[string]bool{}

func handleSSHCB(token string, chat int64, msgID int, data string) bool {
	// Prefix m:sshkeys: only — do not swallow m:ssh:rm (SSH hosts TOFU)
	if data != "m:sshkeys" && !strings.HasPrefix(data, "m:sshkeys:") {
		return false
	}
	ru := getLang() != "en"

	if data == "m:sshkeys" || data == "m:sshkeys:hub" {
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if data == "m:sshkeys:probe" {
		n := 0
		for _, d := range secondary.List() {
			_ = secondary.EnqueueCmd(d.ID, "ssh_keys")
			n++
		}
		msg := fmt.Sprintf(T("sshkeys_probe_ok"), n)
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n✅ "+msg, sshHubKB())
		return true
	}
	if data == "m:sshkeys:clear-sel" {
		sshWizardSelected = map[string]bool{}
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if data == "m:sshkeys:all" {
		for _, k := range allCandidateKeys() {
			sshWizardSelected[k.Fingerprint] = true
		}
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if strings.HasPrefix(data, "m:sshkeys:tog:") {
		fp := strings.TrimPrefix(data, "m:sshkeys:tog:")
		if sshWizardSelected[fp] {
			delete(sshWizardSelected, fp)
		} else {
			sshWizardSelected[fp] = true
		}
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if data == "m:sshkeys:save" {
		fps := selectedFPs()
		err := integrity.SetAllowlistByFingerprints(fps, allCandidateKeys(), false, "tg")
		msg := T("sshkeys_saved")
		if err != nil {
			msg = "❌ " + err.Error()
		}
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n"+msg, sshHubKB())
		return true
	}
	if data == "m:sshkeys:merge" {
		fps := selectedFPs()
		err := integrity.MergeAllowlist(fps, allCandidateKeys(), "tg")
		msg := T("sshkeys_merged")
		if err != nil {
			msg = "❌ " + err.Error()
		}
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n"+msg, sshHubKB())
		return true
	}
	if data == "m:sshkeys:enforce-ask" {
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn(T("sshkeys_enforce_btn"), "m:sshkeys:enforce", ""), btn(T("cancel"), "m:sshkeys", "")},
		}}
		reply(token, chat, msgID, T("sshkeys_enforce_warn"), kb)
		return true
	}
	if data == "m:sshkeys:enforce" {
		fps := selectedFPs()
		if len(fps) == 0 {
			for _, line := range integrity.ReadAllowlistLines() {
				ks := integrity.ParseKeysFromText("x", line)
				if len(ks) > 0 {
					fps = append(fps, ks[0].Fingerprint)
				}
			}
		}
		err := integrity.SetAllowlistByFingerprints(fps, allCandidateKeys(), true, "tg")
		msg := T("sshkeys_enforced")
		if err != nil {
			reply(token, chat, msgID, "❌ "+err.Error(), sshHubKB())
			return true
		}
		if b64, err := integrity.AllowlistB64(); err == nil {
			for _, d := range secondary.List() {
				_ = secondary.EnqueueCmd(d.ID, "ssh_allowlist:"+b64)
			}
			msg += " · " + T("sshkeys_sec_queued")
		}
		_ = ru
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n"+msg, sshHubKB())
		return true
	}
	// unknown m:sshkeys:* — show hub, do not claim success
	reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
	return true
}

func allCandidateKeys() []integrity.LiveKey {
	c := integrity.LiveKeysPrimary()
	for _, d := range secondary.List() {
		c = append(c, integrity.LiveKeysSecondaryCached(d.ID)...)
	}
	return c
}

func selectedFPs() []string {
	var out []string
	for fp, on := range sshWizardSelected {
		if on {
			out = append(out, fp)
		}
	}
	return out
}

func formatSSHHubHTML() string {
	st := integrity.ReadAllowlistState()
	var b strings.Builder
	b.WriteString(T("sshkeys_title") + "\n")
	b.WriteString(T("sshkeys_path") + "\n")
	b.WriteString(fmt.Sprintf(T("sshkeys_meta"), st.Enforce, len(integrity.ReadAllowlistLines())) + "\n\n")
	keys := allCandidateKeys()
	if len(keys) == 0 {
		b.WriteString("<i>" + T("sshkeys_empty") + "</i>\n")
	}
	for _, k := range keys {
		mark := "☐"
		if sshWizardSelected[k.Fingerprint] {
			mark = "☑"
		}
		al := ""
		if k.InAllowlist {
			al = " · allowlist"
		}
		c := k.Comment
		if c == "" {
			c = k.Fingerprint
			if len(c) > 24 {
				c = c[:24] + "…"
			}
		}
		b.WriteString(fmt.Sprintf("%s <code>%s</code> <i>%s</i>%s\n", mark, esc(c), esc(k.Host), al))
	}
	b.WriteString("\n<i>" + T("sshkeys_hint") + "</i>")
	return b.String()
}

func sshHubKB() map[string]any {
	var rows [][]map[string]any
	for _, k := range allCandidateKeys() {
		label := k.Comment
		if label == "" {
			label = k.Fingerprint
		}
		if len(label) > 24 {
			label = label[:24] + "…"
		}
		mark := "☐"
		if sshWizardSelected[k.Fingerprint] {
			mark = "☑"
		}
		rows = append(rows, []map[string]any{btn(mark+" "+label, "m:sshkeys:tog:"+k.Fingerprint, "")})
	}
	rows = append(rows, []map[string]any{
		btn(T("sshkeys_probe"), "m:sshkeys:probe", ""),
		btn(T("sshkeys_all"), "m:sshkeys:all", ""),
		btn(T("sshkeys_clear"), "m:sshkeys:clear-sel", ""),
	})
	rows = append(rows, []map[string]any{
		btn(T("sshkeys_save"), "m:sshkeys:save", ""),
		btn(T("sshkeys_merge"), "m:sshkeys:merge", ""),
		btn(T("sshkeys_enforce"), "m:sshkeys:enforce-ask", ""),
	})
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
	return map[string]any{"inline_keyboard": rows}
}
