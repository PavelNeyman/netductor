package main

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/integrity"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

var sshWizardSelected = map[string]bool{}

func handleSSHCB(token string, chat int64, msgID int, data string) bool {
	if data != "m:ssh" && !strings.HasPrefix(data, "m:ssh:") {
		return false
	}
	ru := getLang() != "en"

	if data == "m:ssh" || data == "m:ssh:hub" {
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if data == "m:ssh:probe" {
		n := 0
		for _, d := range secondary.List() {
			_ = secondary.EnqueueCmd(d.ID, "ssh_keys")
			n++
		}
		msg := fmt.Sprintf("queued ssh_keys on %d secondary — refresh in ~30s", n)
		if ru {
			msg = fmt.Sprintf("ssh_keys на %d secondary — обновите через ~30с", n)
		}
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n✅ "+msg, sshHubKB())
		return true
	}
	if data == "m:ssh:clear-sel" {
		sshWizardSelected = map[string]bool{}
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if data == "m:ssh:all" {
		for _, k := range allCandidateKeys() {
			sshWizardSelected[k.Fingerprint] = true
		}
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if strings.HasPrefix(data, "m:ssh:tog:") {
		fp := strings.TrimPrefix(data, "m:ssh:tog:")
		if sshWizardSelected[fp] {
			delete(sshWizardSelected, fp)
		} else {
			sshWizardSelected[fp] = true
		}
		reply(token, chat, msgID, formatSSHHubHTML(), sshHubKB())
		return true
	}
	if data == "m:ssh:save" {
		fps := selectedFPs()
		err := integrity.SetAllowlistByFingerprints(fps, allCandidateKeys(), false, "tg")
		msg := "✅ allowlist saved (enforce off)"
		if ru {
			msg = "✅ allowlist сохранён (enforce выкл)"
		}
		if err != nil {
			msg = "❌ " + err.Error()
		}
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n"+msg, sshHubKB())
		return true
	}
	if data == "m:ssh:merge" {
		fps := selectedFPs()
		err := integrity.MergeAllowlist(fps, allCandidateKeys(), "tg")
		msg := "✅ merged into allowlist"
		if ru {
			msg = "✅ добавлено в allowlist"
		}
		if err != nil {
			msg = "❌ " + err.Error()
		}
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n"+msg, sshHubKB())
		return true
	}
	if data == "m:ssh:enforce-ask" {
		warn := "⚠️ Enforce will replace authorized_keys with allowlist only.\nHoster recovery keys not in the list will lose SSH.\nContinue?"
		if ru {
			warn = "⚠️ Enforce заменит authorized_keys только на allowlist.\nКлючи хостера вне списка потеряют SSH.\nПродолжить?"
		}
		kb := map[string]any{"inline_keyboard": [][]map[string]any{
			{btn("✅ Enforce", "m:ssh:enforce", ""), btn(T("cancel"), "m:ssh", "")},
		}}
		reply(token, chat, msgID, warn, kb)
		return true
	}
	if data == "m:ssh:enforce" {
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
		msg := "✅ enforced on primary"
		if ru {
			msg = "✅ enforce на primary"
		}
		if err != nil {
			reply(token, chat, msgID, "❌ "+err.Error(), sshHubKB())
			return true
		}
		if b64, err := integrity.AllowlistB64(); err == nil {
			for _, d := range secondary.List() {
				_ = secondary.EnqueueCmd(d.ID, "ssh_allowlist:"+b64)
			}
			msg += " · secondary queued"
		}
		reply(token, chat, msgID, formatSSHHubHTML()+"\n\n"+msg, sshHubKB())
		return true
	}
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
	ru := getLang() != "en"
	st := integrity.ReadAllowlistState()
	var b strings.Builder
	if ru {
		b.WriteString("🔐 <b>SSH allowlist</b>\n")
		b.WriteString("Эталон → <code>/etc/netductor/ssh-allowed.pub</code>\n")
		b.WriteString(fmt.Sprintf("enforce=<b>%v</b> · lines=<b>%d</b>\n\n", st.Enforce, len(integrity.ReadAllowlistLines())))
	} else {
		b.WriteString("🔐 <b>SSH allowlist</b>\n")
		b.WriteString("Canonical → <code>/etc/netductor/ssh-allowed.pub</code>\n")
		b.WriteString(fmt.Sprintf("enforce=<b>%v</b> · lines=<b>%d</b>\n\n", st.Enforce, len(integrity.ReadAllowlistLines())))
	}
	keys := allCandidateKeys()
	if len(keys) == 0 {
		b.WriteString("<i>No keys yet. Probe secondary or check authorized_keys.</i>\n")
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
	if ru {
		b.WriteString("\n<i>Save=эталон без enforce. Merge=добавить. Enforce=применить+secondary.</i>")
	} else {
		b.WriteString("\n<i>Save=allowlist only. Merge=append. Enforce=apply+secondary.</i>")
	}
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
		rows = append(rows, []map[string]any{btn(mark+" "+label, "m:ssh:tog:"+k.Fingerprint, "")})
	}
	rows = append(rows, []map[string]any{
		btn("Probe sec", "m:ssh:probe", ""), btn("All", "m:ssh:all", ""), btn("Clear", "m:ssh:clear-sel", ""),
	})
	rows = append(rows, []map[string]any{
		btn("Save", "m:ssh:save", ""), btn("Merge", "m:ssh:merge", ""), btn("Enforce…", "m:ssh:enforce-ask", ""),
	})
	rows = append(rows, []map[string]any{btn(T("main_menu"), "m:menu", "primary")})
	return map[string]any{"inline_keyboard": rows}
}
