package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/integrity"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

var sshWizardSelected = map[string]bool{}

func handleSSHCB(token string, chat int64, msgID int, data string) bool {
	// Prefix m:sshkeys: only — do not swallow m:ssh:rm (SSH hosts TOFU)
	if data != "m:sshkeys" && !strings.HasPrefix(data, "m:sshkeys:") {
		return false
	}

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
			{btn(T("sshkeys_enforce_btn"), "m:sshkeys:enforce", "")},
			{btn(T("cancel"), "m:sshkeys", "")},
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
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n"+msg, sshHubKB())
		return true
	}
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

func keyLabel(k integrity.LiveKey) string {
	c := strings.TrimSpace(k.Comment)
	if c == "" {
		c = k.Fingerprint
		if len(c) > 20 {
			c = c[:20] + "…"
		}
	}
	if len(c) > 28 {
		c = c[:28] + "…"
	}
	return c
}

func formatSSHHubHTML() string {
	st := integrity.ReadAllowlistState()
	nl := "\n"
	var b strings.Builder
	b.WriteString(T("sshkeys_title") + nl)
	b.WriteString(T("sshkeys_what") + nl + nl)

	prot := T("sshkeys_prot_off")
	if st.Enforce {
		prot = T("sshkeys_prot_on")
	}
	nAllow := len(integrity.ReadAllowlistLines())
	nSel := len(selectedFPs())
	b.WriteString(fmt.Sprintf(T("sshkeys_status"), prot, nAllow, nSel) + nl)
	b.WriteString(T("sshkeys_path") + nl + nl)

	// Primary
	pk := integrity.LiveKeysPrimary()
	b.WriteString("<b>" + T("sshkeys_host_primary") + "</b>" + nl)
	if len(pk) == 0 {
		b.WriteString("<i>" + T("sshkeys_no_primary") + "</i>" + nl)
	} else {
		for _, k := range pk {
			b.WriteString(formatKeyLine(k) + nl)
		}
	}

	// Secondaries
	secs := secondary.List()
	if len(secs) == 0 {
		b.WriteString(nl + "<i>" + T("sshkeys_no_secondary") + "</i>" + nl)
	} else {
		for _, d := range secs {
			name := d.Name
			if name == "" {
				name = d.ID
			}
			on := secondary.Online(d, 2*time.Minute)
			onLab := T("sshkeys_offline")
			if on {
				onLab = T("sshkeys_online")
			}
			b.WriteString(nl + "<b>" + T("sshkeys_host_secondary") + " " + esc(name) + "</b> · " + onLab + nl)
			sk := integrity.LiveKeysSecondaryCached(d.ID)
			if len(sk) == 0 {
				b.WriteString("<i>" + T("sshkeys_sec_empty") + "</i>" + nl)
			} else {
				for _, k := range sk {
					b.WriteString(formatKeyLine(k) + nl)
				}
			}
		}
	}

	b.WriteString(nl + T("sshkeys_steps") + nl)
	b.WriteString("<i>" + T("sshkeys_hint") + "</i>")
	return b.String()
}

func formatKeyLine(k integrity.LiveKey) string {
	mark := "☐"
	if sshWizardSelected[k.Fingerprint] {
		mark = "☑"
	}
	al := ""
	if k.InAllowlist {
		al = " · " + T("sshkeys_in_baseline")
	}
	// comment + short fingerprint ends so two similar comments stay distinguishable
	return fmt.Sprintf("%s <code>%s</code> <i>%s</i>%s", mark, esc(keyLabel(k)), esc(fpShort(k.Fingerprint)), al)
}

func fpShort(fp string) string {
	fp = strings.TrimPrefix(fp, "SHA256:")
	fp = strings.TrimSpace(fp)
	if len(fp) <= 14 {
		return fp
	}
	return fp[:6] + "…" + fp[len(fp)-6:]
}

func pubKeyEnds(line string) (head, tail string) {
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) < 2 {
		s := strings.TrimSpace(line)
		if len(s) <= 24 {
			return s, ""
		}
		return s[:12], s[len(s)-12:]
	}
	blob := parts[1] // base64 body
	if len(blob) <= 20 {
		return blob, ""
	}
	return blob[:10], blob[len(blob)-10:]
}

func formatSSHKeysDetailHTML() string {
	nl := "\n"
	var b strings.Builder
	b.WriteString(T("sshkeys_show_title") + nl)
	b.WriteString("<i>" + T("sshkeys_show_hint") + "</i>" + nl + nl)
	keys := allCandidateKeys()
	if len(keys) == 0 {
		b.WriteString("<i>" + T("sshkeys_empty") + "</i>")
		return b.String()
	}
	for i, k := range keys {
		b.WriteString(formatOneKeyBlock(i+1, k))
		b.WriteString(nl)
	}
	return b.String()
}

func formatOneSSHKeyDetailHTML(fp string) string {
	nl := "\n"
	for _, k := range allCandidateKeys() {
		if k.Fingerprint == fp {
			var b strings.Builder
			b.WriteString(T("sshkeys_show_title") + nl)
			b.WriteString(formatOneKeyBlock(1, k))
			return b.String()
		}
	}
	return T("sshkeys_show_title") + nl + "<i>" + T("sshkeys_empty") + "</i>"
}

func formatOneKeyBlock(n int, k integrity.LiveKey) string {
	nl := "\n"
	mark := "☐"
	if sshWizardSelected[k.Fingerprint] {
		mark = "☑"
	}
	al := ""
	if k.InAllowlist {
		al = " · " + T("sshkeys_in_baseline")
	}
	head, tail := pubKeyEnds(k.Line)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>%d.</b> %s <code>%s</code>%s%s", n, mark, esc(keyLabel(k)), al, nl))
	b.WriteString(fmt.Sprintf("· host: <code>%s</code>%s", esc(k.Host), nl))
	if k.Type != "" {
		b.WriteString(fmt.Sprintf("· type: <code>%s</code>%s", esc(k.Type), nl))
	}
	b.WriteString(fmt.Sprintf("· fp: <code>%s</code>%s", esc(k.Fingerprint), nl))
	if head != "" {
		if tail != "" {
			b.WriteString(fmt.Sprintf("· key: <code>%s…%s</code>%s", esc(head), esc(tail), nl))
		} else {
			b.WriteString(fmt.Sprintf("· key: <code>%s</code>%s", esc(head), nl))
		}
	}
	return b.String()
}

func sshDetailKB() map[string]any {
	return map[string]any{"inline_keyboard": [][]map[string]any{
		{btn(T("sshkeys_refresh"), "m:sshkeys:hub", ""), btn(T("sshkeys_show"), "m:sshkeys:show", "")},
		{btn(T("main_menu"), "m:menu", "primary")},
	}}
}

func sshHubKB() map[string]any {
	var rows [][]map[string]any
	// Toggle keys: label includes host so primary vs secondary is obvious
	for _, k := range allCandidateKeys() {
		mark := "☐"
		if sshWizardSelected[k.Fingerprint] {
			mark = "☑"
		}
		host := k.Host
		if host == "" {
			host = "?"
		}
		if len(host) > 12 {
			host = host[:12]
		}
		label := mark + " " + host + ": " + keyLabel(k) + " " + fpShort(k.Fingerprint)
		if len(label) > 48 {
			label = label[:48] + "…"
		}
		rows = append(rows, []map[string]any{
			btn(label, "m:sshkeys:tog:"+k.Fingerprint, ""),
			btn("👁", "m:sshkeys:show:"+k.Fingerprint, ""),
		})
	}

	rows = append(rows, []map[string]any{
		btn(T("sshkeys_probe"), "m:sshkeys:probe", ""),
		btn(T("sshkeys_refresh"), "m:sshkeys:hub", ""),
	})
	rows = append(rows, []map[string]any{
		btn(T("sshkeys_show"), "m:sshkeys:show", "primary"),
	})
	rows = append(rows, []map[string]any{
		btn(T("sshkeys_all"), "m:sshkeys:all", ""),
		btn(T("sshkeys_clear"), "m:sshkeys:clear-sel", ""),
	})
	rows = append(rows, []map[string]any{
		btn(T("sshkeys_save"), "m:sshkeys:save", "primary"),
	})
	rows = append(rows, []map[string]any{
		btn(T("sshkeys_merge"), "m:sshkeys:merge", ""),
	})
	rows = append(rows, []map[string]any{
		btn(T("sshkeys_enforce"), "m:sshkeys:enforce-ask", ""),
	})
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
	return map[string]any{"inline_keyboard": rows}
}
