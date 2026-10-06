// Package cleanup removes obsolete leftover files/units after upgrades (brew-style).
package cleanup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Item is one removable leftover.
type Item struct {
	Kind   string `json:"kind"` // file|dir|unit|iface
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Report is dry-run or apply result.
type Report struct {
	Apply      bool     `json:"apply"`
	Items      []Item   `json:"items"`
	FreedBytes int64    `json:"freed_bytes"`
	FreedHuman string   `json:"freed_human"`
	Notes      []string `json:"notes,omitempty"`
	At         string   `json:"at"`
}

// Run scans known leftovers. apply=false → dry-run only.
func Run(apply bool) Report {
	r := Report{Apply: apply, At: time.Now().UTC().Format(time.RFC3339)}
	addFile := func(kind, path, detail string) {
		var sz int64
		if st, err := os.Stat(path); err == nil {
			if st.IsDir() {
				sz = dirSize(path)
			} else {
				sz = st.Size()
			}
		} else if kind == "unit" || kind == "iface" {
			// may not exist as file
			sz = 0
		} else {
			return
		}
		it := Item{Kind: kind, Path: path, Bytes: sz, Detail: detail}
		if apply {
			switch kind {
			case "file":
				_ = os.Remove(path)
			case "dir":
				_ = os.RemoveAll(path)
			case "unit":
				_ = exec.Command("systemctl", "stop", path).Run()
				_ = exec.Command("systemctl", "disable", path).Run()
				_ = exec.Command("systemctl", "reset-failed", path).Run()
			case "iface":
				_ = exec.Command("wg-quick", "down", path).Run()
				_ = exec.Command("awg-quick", "down", path).Run()
				_ = exec.Command("ip", "link", "delete", path).Run()
			}
		}
		r.Items = append(r.Items, it)
		r.FreedBytes += sz
	}

	// Stack attempt after successful apply
	addFile("dir", filepath.Join(paths.StateDir(), "stack", "attempt"), "stack apply scratch")

	// sing-box atomic tmp
	addFile("file", "/usr/local/etc/sing-box/config.json.tmp", "incomplete sing-box write")

	// temp validate dirs
	if ents, err := os.ReadDir("/tmp"); err == nil {
		for _, e := range ents {
			name := e.Name()
			if strings.HasPrefix(name, "nd-sb-") {
				addFile("dir", filepath.Join("/tmp", name), "sing-box validate temp")
			}
		}
	}

	// Legacy test plane (never product)
	for _, u := range []string{
		"nd-backbone-iperf-s", "nd-backbone-iperf-c", "nd-awg-iperf-s", "nd-awg-iperf-c",
		"nd-backbone-soak", "wg-quick@nd-backbone", "wg-quick@nd-wgios",
		"strongswan", "strongswan-starter",
	} {
		if unitExists(u) {
			addFile("unit", u, "legacy test unit")
		}
	}
	for _, iface := range []string{"nd-backbone", "nd-awg", "nd-wgios"} {
		if ifaceExists(iface) {
			addFile("iface", iface, "legacy test iface")
		}
	}
	for _, f := range []string{
		"/etc/wireguard/nd-backbone.conf", "/etc/wireguard/nd-wgios.conf",
		"/etc/amnezia/amneziawg/nd-awg.conf",
	} {
		if _, err := os.Stat(f); err == nil {
			addFile("file", f, "legacy test conf")
		}
	}

	r.FreedHuman = humanBytes(r.FreedBytes)
	if !apply {
		r.Notes = append(r.Notes, "dry-run; pass --apply to remove")
	}
	r.Notes = append(r.Notes, "keeps: stack/prev, backups/, baseline/, live sing-box config, nd-svc-sp/ps")
	return r
}

func unitExists(u string) bool {
	out, _ := exec.Command("systemctl", "cat", u).CombinedOutput()
	s := string(out)
	return !strings.Contains(s, "No files found") && len(strings.TrimSpace(s)) > 0 && !strings.Contains(s, "not found")
}

func ifaceExists(name string) bool {
	out, err := exec.Command("ip", "link", "show", name).CombinedOutput()
	return err == nil && len(out) > 0
}

func dirSize(root string) int64 {
	var n int64
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		n += info.Size()
		return nil
	})
	return n
}

func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	f := float64(n)
	for _, u := range []string{"KiB", "MiB", "GiB"} {
		f /= 1024
		if f < 1024 {
			return fmt.Sprintf("%.1f %s", f, u)
		}
	}
	return fmt.Sprintf("%.1f TiB", f/1024)
}

// FormatText is CLI/TG-friendly summary.
func FormatText(r Report) string {
	var b strings.Builder
	mode := "dry-run"
	if r.Apply {
		mode = "applied"
	}
	fmt.Fprintf(&b, "cleanup (%s): %d item(s), free %s\n", mode, len(r.Items), r.FreedHuman)
	for _, it := range r.Items {
		fmt.Fprintf(&b, "  %-5s %s", it.Kind, it.Path)
		if it.Bytes > 0 {
			fmt.Fprintf(&b, " (%s)", humanBytes(it.Bytes))
		}
		if it.Detail != "" {
			fmt.Fprintf(&b, " — %s", it.Detail)
		}
		b.WriteByte('\n')
	}
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "note: %s\n", n)
	}
	return b.String()
}
