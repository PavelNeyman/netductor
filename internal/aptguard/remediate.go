package aptguard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/notify"
)

// Change describes one planned file mutation (preview / apply).
type Change struct {
	Action  string `json:"action"` // quarantine|ensure_file
	Path    string `json:"path"`
	Detail  string `json:"detail"`
	Content string `json:"content,omitempty"`
}

// Preview holds planned remediation without mutating the system.
type Preview struct {
	ServerID      string    `json:"server_id"`
	PolicyVersion int       `json:"policy_version"`
	At            time.Time `json:"at"`
	Changes       []Change  `json:"changes"`
	Findings      []Finding `json:"findings"`
	Blocked       string    `json:"blocked,omitempty"`
	Token         string    `json:"token,omitempty"` // hash binding preview to policy+files
}

type journalEntry struct {
	Action     string `json:"action"`
	Path       string `json:"path"`
	BackupPath string `json:"backup_path,omitempty"`
	Existed    bool   `json:"existed"`
	Mode       uint32 `json:"mode,omitempty"`
	UID        int    `json:"uid,omitempty"`
	GID        int    `json:"gid,omitempty"`
	Quarantine string `json:"quarantine,omitempty"`
}

// BuildPreview scans and proposes safe changes.
// Mixed files (allowed + unknown in same path) block Apply rather than quarantine the whole file.
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

	// Group enabled entries by file
	byFile := map[string][]SourceEntry{}
	for _, s := range rep.Sources {
		if !s.Enabled {
			continue
		}
		byFile[s.File] = append(byFile[s.File], s)
	}

	for path, ents := range byFile {
		if !strings.Contains(path, "sources.list.d") {
			continue
		}
		if strings.HasSuffix(path, ".nd-quarantine") {
			continue
		}
		var bad, allowed int
		for _, e := range ents {
			switch matchPolicy(policy, e) {
			case "ok":
				allowed++
			default:
				bad++
			}
		}
		if bad > 0 && allowed > 0 {
			pv.Blocked = fmt.Sprintf("mixed allowed+noncompliant in %s — refuse whole-file quarantine", path)
			continue
		}
		if bad > 0 && allowed == 0 {
			pv.Changes = append(pv.Changes, Change{
				Action: "quarantine",
				Path:   path,
				Detail: "rename to " + path + ".nd-quarantine (not deleted)",
			})
		}
	}

	needEnsure := false
	for _, f := range rep.Findings {
		if f.Code == "MISSING_REQUIRED" {
			needEnsure = true
			break
		}
	}
	if needEnsure {
		codename := DetectDebianCodename(fsRoot)
		switch {
		case codename == "":
			// Do not block quarantine-only plans; record finding-style detail
			pv.Findings = append(pv.Findings, Finding{
				Code: "CODENAME_UNKNOWN", Severity: SevWarn,
				Message: "cannot detect Debian codename — skip managed sources ensure",
			})
		case !releaseAllowed(policy, codename):
			pv.Blocked = fmt.Sprintf("codename %q not in policy allowed_releases", codename)
		default:
			managed := filepath.Join(fsRoot, "etc/apt/sources.list.d/netductor-policy.sources")
			content, err := managedSourcesContent(policy, codename)
			if err != nil {
				pv.Findings = append(pv.Findings, Finding{Code: "ENSURE_SKIP", Severity: SevWarn, Message: err.Error()})
			} else {
				pv.Changes = append(pv.Changes, Change{
					Action:  "ensure_file",
					Path:    managed,
					Detail:  "write managed required Debian sources for " + codename,
					Content: content,
				})
			}
		}
	}

	if (fsRoot == "/" || fsRoot == "") && aptBusy() {
		if pv.Blocked == "" {
			pv.Blocked = "apt/dpkg lock busy"
		}
	}

	pv.Token = previewToken(pv, policy)
	return pv, nil
}

func releaseAllowed(p Policy, codename string) bool {
	for _, r := range p.Debian.AllowedReleases {
		if r == codename {
			return true
		}
	}
	return false
}

// DetectDebianCodename reads /etc/os-release VERSION_CODENAME (under fsRoot).
func DetectDebianCodename(fsRoot string) string {
	if fsRoot == "" {
		fsRoot = "/"
	}
	b, err := os.ReadFile(filepath.Join(fsRoot, "etc/os-release"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			v := strings.TrimPrefix(line, "VERSION_CODENAME=")
			return strings.Trim(v, "\"'")
		}
	}
	return ""
}

func managedSourcesContent(policy Policy, codename string) (string, error) {
	var b strings.Builder
	b.WriteString("# Managed by netductor aptguard — do not edit by hand\n")
	wrote := 0
	for _, r := range policy.Repos {
		if !r.Required || !r.Enabled {
			continue
		}
		if len(r.URIs) == 0 {
			continue
		}
		suite := ""
		for _, s := range r.Suites {
			if s == codename || strings.HasPrefix(s, codename+"-") {
				suite = s
				break
			}
		}
		if suite == "" {
			continue
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
		wrote++
	}
	if wrote == 0 {
		return "", fmt.Errorf("no required suites match codename %q", codename)
	}
	return b.String(), nil
}

func previewToken(pv Preview, policy Policy) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "v=%d\n", policy.Version)
	for _, c := range pv.Changes {
		_, _ = fmt.Fprintf(h, "%s|%s|%s\n", c.Action, c.Path, c.Detail)
		sum := sha256.Sum256([]byte(c.Content))
		_, _ = fmt.Fprintf(h, "c=%s\n", hex.EncodeToString(sum[:]))
		if st, err := os.Stat(c.Path); err == nil {
			_, _ = fmt.Fprintf(h, "sz=%d m=%d\n", st.Size(), st.ModTime().UnixNano())
		} else {
			_, _ = fmt.Fprintf(h, "missing\n")
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
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

// ApplyPreview applies changes with flock + journal; on failure restores files and metadata.
// confirm must be "yes". previewToken must match a freshly rebuilt plan (or empty only if no changes).
func ApplyPreview(stateRoot, fsRoot string, pv Preview, confirm string) (Report, error) {
	if confirm != "yes" {
		return Report{}, fmt.Errorf("aptguard: apply requires confirm=yes")
	}
	if fsRoot == "" {
		fsRoot = "/"
	}
	policy, err := EnsurePolicy(stateRoot)
	if err != nil {
		return Report{}, err
	}
	if err := os.MkdirAll(Dir(stateRoot), 0o755); err != nil {
		return Report{}, err
	}
	lockPath := filepath.Join(Dir(stateRoot), "remediate.lock")
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return Report{}, err
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return Report{}, fmt.Errorf("aptguard: remediation already in progress")
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

	// Under lock: re-scan, re-token, then mutate
	fresh, err := BuildPreview(pv.ServerID, fsRoot, policy)
	if err != nil {
		return Report{}, err
	}
	if fresh.Blocked != "" {
		return Report{}, fmt.Errorf("aptguard: blocked: %s", fresh.Blocked)
	}
	if len(fresh.Changes) == 0 {
		return ScanHost(pv.ServerID, policy, fsRoot)
	}
	if pv.Token == "" || pv.Token != fresh.Token {
		return Report{}, fmt.Errorf("aptguard: preview token mismatch — re-run preview")
	}
	pv = fresh

	if fsRoot == "/" && aptBusy() {
		return Report{}, fmt.Errorf("aptguard: apt/dpkg lock busy")
	}

	bakDir := filepath.Join(Dir(stateRoot), "remediate-backup", time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(bakDir, 0o700); err != nil {
		return Report{}, err
	}
	journalPath := filepath.Join(bakDir, "journal.json")
	var journal []journalEntry

	persistJournal := func() error {
		jb, err := json.MarshalIndent(journal, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(journalPath, jb, 0o600); err != nil {
			return err
		}
		f, err := os.OpenFile(journalPath, os.O_RDWR, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		return f.Sync()
	}

	fail := func(op error) (Report, error) {
		rbErr := rollbackJournal(journal)
		if rbErr != nil {
			return Report{}, fmt.Errorf("%w; rollback: %v", op, rbErr)
		}
		return Report{}, op
	}

	for i, c := range pv.Changes {
		path := c.Path
		switch c.Action {
		case "quarantine":
			st, err := os.Stat(path)
			if err != nil {
				return fail(fmt.Errorf("stat %s: %w", path, err))
			}
			mode := st.Mode()
			uid, gid := -1, -1
			if sys, ok := st.Sys().(*syscall.Stat_t); ok {
				uid, gid = int(sys.Uid), int(sys.Gid)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fail(err)
			}
			dst := filepath.Join(bakDir, fmt.Sprintf("%d-%s.bak", i, filepath.Base(path)))
			if err := os.WriteFile(dst, data, mode.Perm()); err != nil {
				return fail(fmt.Errorf("backup: %w", err))
			}
			q := path + ".nd-quarantine"
			ent := journalEntry{
				Action: "quarantine", Path: path, BackupPath: dst,
				Existed: true, Mode: uint32(mode.Perm()), UID: uid, GID: gid, Quarantine: q,
			}
			journal = append(journal, ent)
			if err := persistJournal(); err != nil {
				return fail(fmt.Errorf("journal: %w", err))
			}
			if err := os.Rename(path, q); err != nil {
				return fail(err)
			}
		case "ensure_file":
			ent := journalEntry{Action: "ensure_file", Path: path, UID: -1, GID: -1}
			if st, err := os.Stat(path); err == nil {
				ent.Existed = true
				ent.Mode = uint32(st.Mode().Perm())
				if sys, ok := st.Sys().(*syscall.Stat_t); ok {
					ent.UID, ent.GID = int(sys.Uid), int(sys.Gid)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return fail(err)
				}
				dst := filepath.Join(bakDir, fmt.Sprintf("%d-%s.prev", i, filepath.Base(path)))
				if err := os.WriteFile(dst, data, st.Mode().Perm()); err != nil {
					return fail(fmt.Errorf("backup existing: %w", err))
				}
				ent.BackupPath = dst
			}
			journal = append(journal, ent)
			if err := persistJournal(); err != nil {
				return fail(fmt.Errorf("journal: %w", err))
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fail(err)
			}
			if err := os.WriteFile(path, []byte(c.Content), 0o644); err != nil {
				return fail(err)
			}
		default:
			return fail(fmt.Errorf("unknown action %q", c.Action))
		}
	}

	if fsRoot == "/" {
		if aptBusy() {
			return fail(fmt.Errorf("aptguard: apt/dpkg lock busy before update"))
		}
		cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteractive; apt-get update -y -o Acquire::Retries=2")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fail(fmt.Errorf("apt-get update failed: %v (%s)", err, truncateOut(string(out), 300)))
		}
	}

	rep, err := ScanHost(pv.ServerID, policy, fsRoot)
	if err != nil {
		return fail(fmt.Errorf("post-apply scan failed: %w", err))
	}
	if err := SaveReport(stateRoot, rep); err != nil {
		audit.Log("aptguard", "remediate.apply.save_warn", pv.ServerID, err.Error())
	}
	detail := fmt.Sprintf("changes=%d status=%s token=%s", len(pv.Changes), rep.Status, pv.Token)
	audit.Log("aptguard", "remediate.apply", pv.ServerID, detail)
	notify.EnqueueAlert("aptguard:apply:"+pv.ServerID, fmt.Sprintf("APT Guard apply on %s: %s (%d changes)", pv.ServerID, rep.Status, len(pv.Changes)))
	// mark journal complete
	journal = append(journal, journalEntry{Action: "complete", Path: ""})
	_ = persistJournal()
	return rep, nil
}

func rollbackJournal(journal []journalEntry) error {
	var errs []string
	for i := len(journal) - 1; i >= 0; i-- {
		e := journal[i]
		if e.Action == "complete" {
			continue
		}
		switch e.Action {
		case "quarantine":
			if e.Quarantine != "" {
				if err := os.Remove(e.Quarantine); err != nil && !os.IsNotExist(err) {
					errs = append(errs, "remove quarantine: "+err.Error())
				}
			}
			if e.Existed && e.BackupPath != "" {
				if err := restoreFile(e.BackupPath, e.Path, e); err != nil {
					errs = append(errs, err.Error())
				}
			}
		case "ensure_file":
			if !e.Existed {
				if err := os.Remove(e.Path); err != nil && !os.IsNotExist(err) {
					errs = append(errs, "remove ensure: "+err.Error())
				}
			} else if e.BackupPath != "" {
				if err := restoreFile(e.BackupPath, e.Path, e); err != nil {
					errs = append(errs, err.Error())
				}
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func restoreFile(backup, dest string, e journalEntry) error {
	data, err := os.ReadFile(backup)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if e.Mode != 0 {
		mode = os.FileMode(e.Mode)
	}
	if err := os.WriteFile(dest, data, mode); err != nil {
		return err
	}
	if e.UID >= 0 && e.GID >= 0 {
		if err := os.Chown(dest, e.UID, e.GID); err != nil {
			return fmt.Errorf("chown %s: %w", dest, err)
		}
	}
	if err := os.Chmod(dest, mode); err != nil {
		return err
	}
	return nil
}

func truncateOut(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// RecoverIncompleteRemediation rolls back the newest incomplete journal under remediate-backup.
func RecoverIncompleteRemediation(stateRoot string) error {
	root := filepath.Join(Dir(stateRoot), "remediate-backup")
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var latest string
	for _, e := range ents {
		if e.IsDir() {
			name := e.Name()
			if name > latest {
				latest = name
			}
		}
	}
	if latest == "" {
		return nil
	}
	jp := filepath.Join(root, latest, "journal.json")
	b, err := os.ReadFile(jp)
	if err != nil {
		return err
	}
	var journal []journalEntry
	if err := json.Unmarshal(b, &journal); err != nil {
		return err
	}
	for _, e := range journal {
		if e.Action == "complete" {
			return nil // finished ok
		}
	}
	return rollbackJournal(journal)
}
