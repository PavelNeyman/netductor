package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cmdDoneDir holds per-command markers:
//   <id>.started  — runCmd begun (best-effort crash signal)
//   <id>          — completed result text
func cmdDoneDir() string {
	return filepath.Join(agentDir(), "cmd_done")
}

func cmdDonePath(id string) string {
	id = strings.TrimSpace(id)
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

// cmdStarted reports whether a started marker exists without a completed result
// (process may have died mid-command — destructive actions must not be blindly re-run).
func cmdStarted(id string) bool {
	p := cmdDonePath(id)
	if p == "" {
		return false
	}
	if _, err := os.Stat(p); err == nil {
		return false // completed wins
	}
	_, err := os.Stat(p + ".started")
	return err == nil
}

func markCmdStarted(id string) error {
	p := cmdDonePath(id)
	if p == "" {
		return fmt.Errorf("invalid cmd id")
	}
	if err := os.MkdirAll(cmdDoneDir(), 0o700); err != nil {
		return err
	}
	tmp := p + ".started.tmp"
	if err := os.WriteFile(tmp, []byte("1\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p+".started")
}

func saveCmdDone(id, result string) error {
	p := cmdDonePath(id)
	if p == "" {
		return fmt.Errorf("invalid cmd id")
	}
	if err := os.MkdirAll(cmdDoneDir(), 0o700); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(result), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(p + ".started")
	pruneCmdDone()
	return nil
}

// pruneCmdDone keeps at most 200 completed markers; never deletes .started.
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
		name := e.Name()
		if strings.HasSuffix(name, ".started") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		all = append(all, item{name, info.ModTime()})
	}
	if len(all) <= 200 {
		return
	}
	// Selection sort oldest-first
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
