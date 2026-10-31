package aptguard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	pv.Token = previewToken(pv, policy, fsRoot)
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
		if len(r.SignedBy) == 0 {
			return "", fmt.Errorf("repo %s: SignedBy required for managed sources", r.ID)
		}
		b.WriteString("Types: deb\n")
		b.WriteString("URIs: " + r.URIs[0] + "\n")
		b.WriteString("Suites: " + suite + "\n")
		b.WriteString("Components: " + comps + "\n")
		b.WriteString("Signed-By: " + r.SignedBy[0] + "\n\n")
		wrote++
	}
	if wrote == 0 {
		return "", fmt.Errorf("no required suites match codename %q", codename)
	}
	return b.String(), nil
}

func policyFingerprint(policy Policy) string {
	b, err := json.Marshal(policy)
	if err != nil {
		return "err"
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// aptSourcesFingerprint hashes path+content of all APT source files under fsRoot.
func aptSourcesFingerprint(fsRoot string) string {
	h := sha256.New()
	paths := []string{filepath.Join(fsRoot, "etc/apt/sources.list")}
	dir := filepath.Join(fsRoot, "etc/apt/sources.list.d")
	ents, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(h, "DIRERR:%v\n", err)
		}
	} else {
		names := make([]string, 0, len(ents))
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			names = append(names, e.Name())
		}
		sort.Strings(names)
		for _, name := range names {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	for _, pth := range paths {
		fmt.Fprintf(h, "%s\n", pth)
		b, err := os.ReadFile(pth)
		if err != nil {
			fmt.Fprintf(h, "ERR:%v\n", err)
			continue
		}
		sum := sha256.Sum256(b)
		fmt.Fprintf(h, "%s\n", hex.EncodeToString(sum[:]))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func previewToken(pv Preview, policy Policy, fsRoot string) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "policy=%s\n", policyFingerprint(policy))
	_, _ = fmt.Fprintf(h, "sources=%s\n", aptSourcesFingerprint(fsRoot))
	for _, c := range pv.Changes {
		_, _ = fmt.Fprintf(h, "%s|%s|%s\n", c.Action, c.Path, c.Detail)
		sum := sha256.Sum256([]byte(c.Content))
		_, _ = fmt.Fprintf(h, "c=%s\n", hex.EncodeToString(sum[:]))
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

func durableWrite(path string, data []byte, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o600
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return fsyncDir(dir)
}

func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// mkdirDurable creates path and fsyncs the parent so the directory entry survives crash.
func mkdirDurable(path string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o700
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	if err := fsyncDir(path); err != nil {
		return err
	}
	parent := filepath.Dir(path)
	if parent != "" && parent != "." && parent != path {
		_ = fsyncDir(parent)
	}
	return nil
}

func ApplyPreview(stateRoot, fsRoot string, pv Preview, confirm string) (Report, error) {
	if confirm != "yes" {
		return Report{}, fmt.Errorf("aptguard: apply requires confirm=yes")
	}
	if fsRoot == "" {
		fsRoot = "/"
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

	if err := recoverAllIncompleteLocked(stateRoot); err != nil {
		return Report{}, fmt.Errorf("aptguard: recovery: %w", err)
	}

	policy, err := EnsurePolicy(stateRoot)
	if err != nil {
		return Report{}, err
	}

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
	if err := mkdirDurable(bakDir, 0o700); err != nil {
		return Report{}, err
	}
	// also durable-create parent tree for remediate-backup
	_ = fsyncDir(filepath.Dir(bakDir))
	journalPath := filepath.Join(bakDir, "journal.json")
	commitPath := filepath.Join(bakDir, "COMMIT")
	var journal []journalEntry

	persistJournal := func() error {
		jb, err := json.MarshalIndent(journal, "", "  ")
		if err != nil {
			return err
		}
		return durableWrite(journalPath, jb, 0o600)
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
			if err := durableWrite(dst, data, mode.Perm()); err != nil {
				return fail(fmt.Errorf("backup: %w", err))
			}
			q := path + ".nd-quarantine"
			if _, err := os.Stat(q); err == nil {
				return fail(fmt.Errorf("quarantine destination already exists: %s", q))
			} else if err != nil && !os.IsNotExist(err) {
				return fail(fmt.Errorf("stat quarantine dest: %w", err))
			}
			ent := journalEntry{
				Action: "quarantine", Path: path, BackupPath: dst,
				Existed: true, Mode: uint32(mode.Perm()), UID: uid, GID: gid, Quarantine: q,
			}
			journal = append(journal, ent)
			if err := persistJournal(); err != nil {
				return fail(fmt.Errorf("journal: %w", err))
			}
			if err := os.Link(path, q); err != nil {
				return fail(fmt.Errorf("quarantine link (no rename fallback): %w", err))
			}
			if err := os.Remove(path); err != nil {
				_ = os.Remove(q)
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
				if err := durableWrite(dst, data, st.Mode().Perm()); err != nil {
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
			if err := durableWrite(path, []byte(c.Content), 0o644); err != nil {
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
	// Forensic complete in journal (not sole commit signal).
	journal = append(journal, journalEntry{Action: "complete", Path: ""})
	if err := persistJournal(); err != nil {
		return fail(fmt.Errorf("commit journal complete: %w", err))
	}
	// Durable COMMIT marker is the only recovery "done" signal.
	if err := durableWrite(commitPath, []byte("ok\n"), 0o600); err != nil {
		return fail(fmt.Errorf("commit marker: %w", err))
	}
	audit.Log("aptguard", "remediate.apply", pv.ServerID, detail)
	notify.EnqueueAlert("aptguard:apply:"+pv.ServerID, fmt.Sprintf("APT Guard apply on %s: %s (%d changes)", pv.ServerID, rep.Status, len(pv.Changes)))
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
	dir := filepath.Dir(dest)
	if err := mkdirDurable(dir, 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if e.Mode != 0 {
		mode = os.FileMode(e.Mode)
	}
	// Write to temp in same dir, set metadata, fsync, then atomic rename.
	tmp, err := os.CreateTemp(dir, ".nd-restore-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if e.UID >= 0 && e.GID >= 0 {
		if err := tmp.Chown(e.UID, e.GID); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("chown %s: %w", tmpName, err)
		}
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return err
	}
	cleanup = false
	return fsyncDir(dir)
}

func truncateOut(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// RecoverIncompleteRemediation acquires the remediation lock and rolls back any incomplete journals.
func RecoverIncompleteRemediation(stateRoot string) error {
	if err := os.MkdirAll(Dir(stateRoot), 0o755); err != nil {
		return err
	}
	lockPath := filepath.Join(Dir(stateRoot), "remediate.lock")
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)
	return recoverAllIncompleteLocked(stateRoot)
}

// recoverAllIncompleteLocked assumes caller holds remediate.lock.
func recoverAllIncompleteLocked(stateRoot string) error {
	root := filepath.Join(Dir(stateRoot), "remediate-backup")
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var errs []string
	for _, name := range names {
		jp := filepath.Join(root, name, "journal.json")
		b, err := os.ReadFile(jp)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, name+": "+err.Error())
			continue
		}
		var journal []journalEntry
		if err := json.Unmarshal(b, &journal); err != nil {
			errs = append(errs, name+": "+err.Error())
			continue
		}
		// COMMIT is the durable done marker; journal "complete" alone is not sufficient
		// (rename of journal may succeed while fsyncDir fails).
		if _, err := os.Stat(filepath.Join(root, name, "COMMIT")); err == nil {
			continue
		}
		done := false
		for _, e := range journal {
			if e.Action == "recovered" {
				done = true
				break
			}
		}
		if done {
			continue
		}
		if err := rollbackJournal(journal); err != nil {
			errs = append(errs, name+": rollback: "+err.Error())
			continue
		}
		journal = append(journal, journalEntry{Action: "recovered", Path: name})
		jb, mErr := json.MarshalIndent(journal, "", "  ")
		if mErr != nil {
			errs = append(errs, name+": marshal recovered: "+mErr.Error())
			continue
		}
		if wErr := durableWrite(jp, jb, 0o600); wErr != nil {
			errs = append(errs, name+": write recovered: "+wErr.Error())
			continue
		}
		// Mark recovered with COMMIT so we never re-rollback this tx.
		if wErr := durableWrite(filepath.Join(root, name, "COMMIT"), []byte("recovered\n"), 0o600); wErr != nil {
			errs = append(errs, name+": write recovered COMMIT: "+wErr.Error())
			continue
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
