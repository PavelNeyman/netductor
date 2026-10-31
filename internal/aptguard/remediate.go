package aptguard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Change describes one planned file mutation (preview / apply).
type Change struct {
	Action  string `json:"action"` // quarantine|ensure_file|restore
	Path    string `json:"path"`
	Detail  string `json:"detail"`
	Content string `json:"content,omitempty"` // for ensure_file
}

// Preview holds planned remediation without mutating the system.
type Preview struct {
	ServerID      string    `json:"server_id"`
	PolicyVersion int       `json:"policy_version"`
	At            time.Time `json:"at"`
	Changes       []Change  `json:"changes"`
	Findings      []Finding `json:"findings"`
	Blocked       string    `json:"blocked,omitempty"` // non-empty = must not apply
}

// BuildPreview scans and proposes quarantine of unknown/forbidden sources
// and ensure of missing required Debian entries via a managed sources file.
// Does not delete files; quarantine = rename to *.nd-quarantine.
func BuildPreview(serverID, fsRoot string, policy Policy) (Preview, error) {
	if fsRoot == "" {
		fsRoot = "/"
	}
	if err := policy.Validate(); err != nil {
		return Preview{}, err
	}
	rep, err := ScanHost(serverID, policy, fsRoot)
	if err != nil {
		return Preview{}, err
	}
	pv := Preview{
		ServerID:      serverID,
		PolicyVersion: policy.Version,
		At:            time.Now().UTC(),
		Findings:      rep.Findings,
	}
	seenFiles := map[string]bool{}
	for _, f := range rep.Findings {
		if f.Code == "UNKNOWN_REPOSITORY" || f.Code == "FORBIDDEN_REPOSITORY" {
			path := f.SourceFile
			if path == "" || seenFiles[path] {
				continue
			}
			// only quarantine files under sources.list.d (never wholesale sources.list without care)
			if strings.Contains(path, "sources.list.d") && !strings.HasSuffix(path, ".nd-quarantine") {
				seenFiles[path] = true
				pv.Changes = append(pv.Changes, Change{
					Action: "quarantine",
					Path:   path,
					Detail: "rename to " + path + ".nd-quarantine (not deleted)",
				})
			}
		}
	}
	// Missing required → managed Deb822 file
	needEnsure := false
	for _, f := range rep.Findings {
		if f.Code == "MISSING_REQUIRED" {
			needEnsure = true
			break
		}
	}
	if needEnsure {
		managed := filepath.Join(fsRoot, "etc/apt/sources.list.d/netductor-policy.sources")
		pv.Changes = append(pv.Changes, Change{
			Action:  "ensure_file",
			Path:    managed,
			Detail:  "write managed required Debian sources from policy",
			Content: managedSourcesContent(policy),
		})
	}
	// Only probe live dpkg locks when scanning the real root filesystem.
	if (fsRoot == "/" || fsRoot == "") && aptBusy() {
		pv.Blocked = "apt/dpkg lock busy"
	}
	return pv, nil
}

func managedSourcesContent(policy Policy) string {
	var b strings.Builder
	b.WriteString("# Managed by netductor aptguard — do not edit by hand\n")
	for _, r := range policy.Repos {
		if !r.Required || !r.Enabled {
			continue
		}
		if len(r.URIs) == 0 || len(r.Suites) == 0 {
			continue
		}
		// Prefer first suite matching allowed release if possible
		suite := r.Suites[0]
		for _, s := range r.Suites {
			for _, rel := range policy.Debian.AllowedReleases {
				if strings.HasPrefix(s, rel) {
					suite = s
					break
				}
			}
		}
		comps := "main"
		if len(r.Components) > 0 {
			comps = strings.Join(r.Components, " ")
		}
		b.WriteString("Types: deb\n")
		b.WriteString("URIs: " + r.URIs[0] + "\n")
		b.WriteString("Suites: " + suite + "\n")
		b.WriteString("Components: " + comps + "\n")
		b.WriteString("Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg\n\n")
	}
	return b.String()
}

func aptBusy() bool {
	for _, p := range []string{
		"/var/lib/dpkg/lock-frontend",
		"/var/lib/apt/lists/lock",
		"/var/lib/dpkg/lock",
	} {
		out, _ := exec.Command("fuser", p).CombinedOutput()
		if len(strings.TrimSpace(string(out))) > 0 {
			return true
		}
	}
	return false
}

const remediateLockName = "remediate.lock"

// ApplyPreview applies changes with backup; on failure restores backup.
// confirm must be "yes". Never runs apt-get upgrade.
func ApplyPreview(stateRoot, fsRoot string, pv Preview, confirm string) (Report, error) {
	if confirm != "yes" {
		return Report{}, fmt.Errorf("aptguard: apply requires confirm=yes")
	}
	if pv.Blocked != "" {
		return Report{}, fmt.Errorf("aptguard: blocked: %s", pv.Blocked)
	}
	if len(pv.Changes) == 0 {
		return ScanHost(pv.ServerID, mustPolicy(stateRoot), fsRoot)
	}
	if fsRoot == "" {
		fsRoot = "/"
	}
	lockPath := filepath.Join(Dir(stateRoot), remediateLockName)
	if err := os.MkdirAll(Dir(stateRoot), 0o755); err != nil {
		return Report{}, err
	}
	if _, err := os.Stat(lockPath); err == nil {
		return Report{}, fmt.Errorf("aptguard: remediation already in progress")
	}
	if err := os.WriteFile(lockPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0o600); err != nil {
		return Report{}, err
	}
	defer os.Remove(lockPath)

	bakDir := filepath.Join(Dir(stateRoot), "remediate-backup", time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(bakDir, 0o700); err != nil {
		return Report{}, err
	}

	type bakItem struct{ src, dst string }
	var bak []bakItem

	rollback := func() error {
		for i := len(bak) - 1; i >= 0; i-- {
			item := bak[i]
			// if dst is quarantine target that didn't exist before — remove
			if strings.HasSuffix(item.src, ".nd-quarantine") {
				_ = os.Remove(item.src)
			}
			data, err := os.ReadFile(item.dst)
			if err != nil {
				continue
			}
			_ = os.MkdirAll(filepath.Dir(item.src), 0o755)
			if err := os.WriteFile(item.src, data, 0o644); err != nil {
				return err
			}
		}
		return nil
	}

	for _, c := range pv.Changes {
		switch c.Action {
		case "quarantine":
			src := c.Path
			if !strings.HasPrefix(src, fsRoot) && fsRoot != "/" {
				src = filepath.Join(fsRoot, strings.TrimPrefix(c.Path, "/"))
			}
			data, err := os.ReadFile(src)
			if err != nil {
				_ = rollback()
				return Report{}, fmt.Errorf("read %s: %w", src, err)
			}
			dst := filepath.Join(bakDir, filepath.Base(src))
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				_ = rollback()
				return Report{}, err
			}
			bak = append(bak, bakItem{src: src, dst: dst})
			q := src + ".nd-quarantine"
			if err := os.Rename(src, q); err != nil {
				_ = rollback()
				return Report{}, fmt.Errorf("quarantine %s: %w", src, err)
			}
		case "ensure_file":
			path := c.Path
			if !strings.HasPrefix(path, fsRoot) && fsRoot != "/" {
				path = filepath.Join(fsRoot, strings.TrimPrefix(c.Path, "/"))
			}
			if prev, err := os.ReadFile(path); err == nil {
				dst := filepath.Join(bakDir, filepath.Base(path)+".prev")
				_ = os.WriteFile(dst, prev, 0o644)
				bak = append(bak, bakItem{src: path, dst: dst})
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				_ = rollback()
				return Report{}, err
			}
			if err := os.WriteFile(path, []byte(c.Content), 0o644); err != nil {
				_ = rollback()
				return Report{}, err
			}
		default:
			_ = rollback()
			return Report{}, fmt.Errorf("unknown action %q", c.Action)
		}
	}

	// apt-get update to validate indexes (no upgrade)
	if fsRoot == "/" {
		cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteractive; apt-get update -y -o Acquire::Retries=2")
		out, err := cmd.CombinedOutput()
		if err != nil {
			_ = rollback()
			return Report{}, fmt.Errorf("apt-get update failed after remediate (rolled back): %w (%s)", err, truncateOut(string(out), 300))
		}
	}

	policy := mustPolicy(stateRoot)
	rep, err := ScanHost(pv.ServerID, policy, fsRoot)
	if err != nil {
		return Report{}, err
	}
	_ = SaveReport(stateRoot, rep)
	return rep, nil
}

func mustPolicy(stateRoot string) Policy {
	p, err := EnsurePolicy(stateRoot)
	if err != nil {
		return DefaultPolicy()
	}
	return p
}

func truncateOut(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
