package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cmdDoneDir holds results of completed command IDs so re-leased commands
// are not re-executed (reboot/sysupgrade/storage_restore etc.).
func cmdDoneDir() string {
	return filepath.Join(agentDir(), "cmd_done")
}

func cmdDonePath(id string) string {
	id = strings.TrimSpace(id)
	// Restrict filename to safe charset.
	safe := make([]rune, 0, len(id))
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			safe = append(safe, r)
		}
	}
	if len(safe) == 0 {
		return ""
	}
	return filepath.Join(cmdDoneDir(), string(safe))
}

func loadCmdDone(id string) (result string, ok bool) {
	p := cmdDonePath(id)
	if p == "" {
		return "", false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func saveCmdDone(id, result string) {
	p := cmdDonePath(id)
	if p == "" {
		return
	}
	_ = os.MkdirAll(cmdDoneDir(), 0o700)
	_ = os.WriteFile(p, []byte(result), 0o600)
	pruneCmdDone()
}

// pruneCmdDone keeps at most 200 newest markers (by mtime).
func pruneCmdDone() {
	ents, err := os.ReadDir(cmdDoneDir())
	if err != nil || len(ents) <= 200 {
		return
	}
	type item struct {
		name string
		mod  time.Time
	}
	var all []item
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		all = append(all, item{e.Name(), info.ModTime()})
	}
	if len(all) <= 200 {
		return
	}
	// Drop oldest.
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].mod.Before(all[i].mod) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	for _, it := range all[:len(all)-200] {
		_ = os.Remove(filepath.Join(cmdDoneDir(), it.name))
	}
}
