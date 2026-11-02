package svcpaths

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	bootstrapJournalVersion = 2
	maxBootstrapJournalBytes = 16 << 20
	maxBootstrapImageBytes = 4 << 20
	maxBootstrapJournalFiles = 128
)

var bootstrapJournalRoot = "/var/lib/netductor/bootstrap-transactions"

// Snapshot targets are constrained to Netductor-managed configuration trees.
var bootstrapAllowedFileRoots = []string{
	"/etc/wireguard", "/etc/systemd/system", "/etc/netductor",
	"/usr/local/bin", "/var/lib/netductor",
}

type bootstrapFileImage struct {
	Path   string      `json:"path"`
	Exists bool        `json:"exists"`
	Data   []byte      `json:"data,omitempty"`
	Mode   os.FileMode `json:"mode,omitempty"`
	UID    int         `json:"uid,omitempty"`
	GID    int         `json:"gid,omitempty"`
}

type bootstrapUnitImage struct {
	Name       string `json:"name"`
	LoadState  string `json:"load_state"`
	FileState  string `json:"file_state"`
	ActiveState string `json:"active_state"`
}

type bootstrapJournal struct {
	Version   int                  `json:"version"`
	TxID      string               `json:"tx_id"`
	Role      string               `json:"role"`
	PeerID    string               `json:"peer_id,omitempty"`
	Slot      int                  `json:"slot"`
	Phase     string               `json:"phase"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	Files     []bootstrapFileImage `json:"files"`
	Units        []bootstrapUnitImage `json:"units,omitempty"`
	TouchedUnits []string             `json:"touched_units,omitempty"`
}

type bootstrapFileTransaction struct {
	journal bootstrapJournal
	path    string
}

func beginBootstrapFileTransaction(role, peerID string, slot int) (*bootstrapFileTransaction, error) {
	if role != "primary" && role != "secondary" {
		return nil, fmt.Errorf("invalid bootstrap transaction role %q", role)
	}
	if slot < 0 || slot >= len(SlotTable()) {
		return nil, fmt.Errorf("invalid bootstrap transaction slot %d", slot)
	}
	if err := ensureBootstrapJournalDirectory(bootstrapJournalRoot); err != nil {
		return nil, err
	}
	txid, err := randomBootstrapTransactionID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	j := bootstrapJournal{Version: bootstrapJournalVersion, TxID: txid, Role: role,
		PeerID: peerID, Slot: slot, Phase: "applying", CreatedAt: now, UpdatedAt: now}
	tx := &bootstrapFileTransaction{journal: j, path: filepath.Join(bootstrapJournalRoot, txid+".json")}
	if err := tx.persist(); err != nil {
		return nil, err
	}
	return tx, nil
}

func randomBootstrapTransactionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate bootstrap transaction id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func (tx *bootstrapFileTransaction) persist() error {
	tx.journal.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(tx.journal, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(tx.path)
	f, err := os.CreateTemp(dir, ".bootstrap-journal-*")
	if err != nil {
		return fmt.Errorf("create bootstrap journal temp: %w", err)
	}
	tmp := f.Name()
	cleanup := func() { _ = f.Close(); _ = os.Remove(tmp) }
	if err := f.Chmod(0o600); err != nil { cleanup(); return err }
	if n, err := f.Write(b); err != nil { cleanup(); return err } else if n != len(b) { cleanup(); return fmt.Errorf("short bootstrap journal write") }
	if err := f.Sync(); err != nil { cleanup(); return err }
	if err := f.Close(); err != nil { _ = os.Remove(tmp); return err }
	if err := os.Rename(tmp, tx.path); err != nil { _ = os.Remove(tmp); return fmt.Errorf("publish bootstrap journal: %w", err) }
	d, err := os.Open(dir)
	if err != nil { return fmt.Errorf("open bootstrap journal directory for sync: %w", err) }
	defer d.Close()
	if err := d.Sync(); err != nil { return fmt.Errorf("sync bootstrap journal directory: %w", err) }
	return nil
}

// SnapshotFile durably records a file's before-image before its caller mutates it.
// Repeated snapshots are ignored so rollback always returns to transaction start.
func (tx *bootstrapFileTransaction) SnapshotFile(path string) error {
	if tx == nil || tx.journal.Phase != "applying" {
		return fmt.Errorf("bootstrap file transaction is not applying")
	}
	if err := validateBootstrapSnapshotPathForIdentity(tx.journal.Role, tx.journal.Slot, path); err != nil { return err }
	for _, image := range tx.journal.Files {
		if image.Path == path { return nil }
	}
	image := bootstrapFileImage{Path: path}
	info, err := os.Lstat(path)
	if err != nil {
		if !os.IsNotExist(err) { return fmt.Errorf("inspect bootstrap target %s: %w", path, err) }
	} else {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("bootstrap target %s must be a regular file, not symlink/special", path)
		}
		data, err := os.ReadFile(path)
		if err != nil { return fmt.Errorf("read bootstrap target %s: %w", path, err) }
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok { return fmt.Errorf("stat metadata unavailable for bootstrap target %s", path) }
		image.Exists, image.Data, image.Mode = true, data, info.Mode().Perm()
		image.UID, image.GID = int(st.Uid), int(st.Gid)
	}
	tx.journal.Files = append(tx.journal.Files, image)
	if err := tx.persist(); err != nil {
		tx.journal.Files = tx.journal.Files[:len(tx.journal.Files)-1]
		return fmt.Errorf("persist before-image for %s: %w", path, err)
	}
	return nil
}


var bootstrapSystemctl = func(args ...string) ([]byte, error) {
	return exec.Command("systemctl", args...).CombinedOutput()
}

// SnapshotUnits records unit load, enablement and runtime state before any
// bootstrap file or service mutation. Repeated snapshots preserve transaction start.
func (tx *bootstrapFileTransaction) SnapshotUnits(names ...string) error {
	if tx == nil || tx.journal.Phase != "applying" {
		return fmt.Errorf("bootstrap file transaction is not applying")
	}
	seen := make(map[string]bool, len(tx.journal.Units)+len(names))
	for _, image := range tx.journal.Units { seen[image.Name] = true }
	pending := make([]bootstrapUnitImage, 0, len(names))
	for _, name := range names {
		if !safeUnitName(name) {
			return fmt.Errorf("invalid systemd unit name %q", name)
		}
		if seen[name] { continue }
		out, err := bootstrapSystemctl("show", "--property=LoadState", "--property=UnitFileState", "--property=ActiveState", name)
		text := strings.TrimSpace(string(out))
		if err != nil && !strings.Contains(text, "LoadState=not-found") && !strings.Contains(text, "LoadState=masked") {
			return fmt.Errorf("snapshot systemd unit %s: %w: %s", name, err, text)
		}
		values := map[string]string{}
		for _, line := range strings.Split(text, "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
			if ok { values[key] = value }
		}
		if values["LoadState"] == "" {
			return fmt.Errorf("snapshot systemd unit %s: missing LoadState in systemctl output %q", name, text)
		}
		image := bootstrapUnitImage{Name:name, LoadState:values["LoadState"], FileState:values["UnitFileState"], ActiveState:values["ActiveState"]}
		if image.LoadState != "not-found" && (image.FileState == "" || image.ActiveState == "") {
			return fmt.Errorf("snapshot systemd unit %s: incomplete state in systemctl output %q", name, text)
		}
		pending = append(pending, image)
		seen[name] = true
	}
	if len(pending) == 0 { return nil }
	oldLen := len(tx.journal.Units)
	tx.journal.Units = append(tx.journal.Units, pending...)
	if err := tx.persist(); err != nil {
		tx.journal.Units = tx.journal.Units[:oldLen]
		return fmt.Errorf("persist systemd before-images: %w", err)
	}
	return nil
}

func (tx *bootstrapFileTransaction) MarkUnitsTouched(names ...string) error {
	if tx == nil || tx.journal.Phase != "applying" {
		return fmt.Errorf("bootstrap file transaction is not applying")
	}
	known := make(map[string]bool, len(tx.journal.Units))
	for _, image := range tx.journal.Units { known[image.Name] = true }
	seen := make(map[string]bool, len(tx.journal.TouchedUnits)+len(names))
	for _, name := range tx.journal.TouchedUnits { seen[name] = true }
	pending := make([]string, 0, len(names))
	for _, name := range names {
		if !safeUnitName(name) || !known[name] {
			return fmt.Errorf("systemd unit %q must be safely snapshotted before mutation", name)
		}
		if !seen[name] { pending = append(pending, name); seen[name] = true }
	}
	if len(pending) == 0 { return nil }
	oldLen := len(tx.journal.TouchedUnits)
	tx.journal.TouchedUnits = append(tx.journal.TouchedUnits, pending...)
	if err := tx.persist(); err != nil {
		tx.journal.TouchedUnits = tx.journal.TouchedUnits[:oldLen]
		return fmt.Errorf("persist systemd mutation intent: %w", err)
	}
	return nil
}

func restoreBootstrapUnits(images []bootstrapUnitImage, touched []string) error {
	wanted := make(map[string]bool, len(touched))
	for _, name := range touched { wanted[name] = true }
	var errs []error
	for i := len(images)-1; i >= 0; i-- {
		image := images[i]
		if !safeUnitName(image.Name) {
			errs = append(errs, fmt.Errorf("unsafe systemd unit name %q in rollback journal", image.Name))
			continue
		}
		if !wanted[image.Name] { continue }
		if image.LoadState == "not-found" {
			for _, action := range [][]string{{"stop", image.Name}, {"disable", image.Name}} {
				out, err := bootstrapSystemctl(action...)
				if err != nil && !unitMissing(string(out)) {
					errs = append(errs, fmt.Errorf("restore absent unit %s (%s): %w: %s", image.Name, action[0], err, strings.TrimSpace(string(out))))
				}
			}
			continue
		}
		isMasked := image.FileState == "masked" || image.FileState == "masked-runtime"
		out, err := bootstrapSystemctl("unmask", image.Name)
		if err != nil { errs = append(errs, fmt.Errorf("unmask unit %s for state restoration: %w: %s", image.Name, err, strings.TrimSpace(string(out)))) }
		if !isMasked {
			switch image.FileState {
			case "enabled", "enabled-runtime":
				args := []string{"enable"}
				if image.FileState == "enabled-runtime" { args = append(args, "--runtime") }
				args = append(args, image.Name)
				out, err = bootstrapSystemctl(args...)
			case "disabled":
				out, err = bootstrapSystemctl("disable", image.Name)
			default:
				out, err = nil, nil // static/indirect/generated/linked/transient units are not enableable
			}
			if err != nil { errs = append(errs, fmt.Errorf("restore enablement for %s: %w: %s", image.Name, err, strings.TrimSpace(string(out)))) }
		}
		if image.ActiveState == "active" || image.ActiveState == "activating" || image.ActiveState == "reloading" {
			out, err = bootstrapSystemctl("restart", image.Name)
			if err != nil { errs = append(errs, fmt.Errorf("restart previously active unit %s: %w: %s", image.Name, err, strings.TrimSpace(string(out)))) }
		} else {
			out, err = bootstrapSystemctl("stop", image.Name)
			if err != nil && !unitMissing(string(out)) { errs = append(errs, fmt.Errorf("stop previously inactive unit %s: %w: %s", image.Name, err, strings.TrimSpace(string(out)))) }
		}
		if isMasked {
			args := []string{"mask"}
			if image.FileState == "masked-runtime" { args = append(args, "--runtime") }
			args = append(args, image.Name)
			out, err = bootstrapSystemctl(args...)
			if err != nil { errs = append(errs, fmt.Errorf("restore masked unit %s: %w: %s", image.Name, err, strings.TrimSpace(string(out)))) }
		}
	}
	return errors.Join(errs...)
}

func (tx *bootstrapFileTransaction) writeFile(path, body string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return err }
	if tx != nil {
		if err := tx.SnapshotFile(path); err != nil { return err }
	}
	return atomicWritePeerFile(path, []byte(body), mode)
}

func (tx *bootstrapFileTransaction) atomicWritePeerFile(path string, body []byte, mode os.FileMode) error {
	if tx != nil {
		if err := tx.SnapshotFile(path); err != nil { return err }
	}
	return atomicWritePeerFile(path, body, mode)
}

func (tx *bootstrapFileTransaction) Rollback() error {
	if tx == nil { return nil }
	tx.journal.Phase = "rolling_back"
	if err := tx.persist(); err != nil { return fmt.Errorf("checkpoint rollback: %w", err) }
	var errs []error
	for i := len(tx.journal.Files)-1; i >= 0; i-- {
		if err := restoreBootstrapFile(tx.journal.Files[i]); err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", tx.journal.Files[i].Path, err))
		}
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}
	if tx.journalTouchesSystemdUnits() || len(tx.journal.Units) > 0 {
		out, err := bootstrapSystemctl("daemon-reload")
		if err != nil {
			return fmt.Errorf("reload systemd after bootstrap file rollback: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	if err := restoreBootstrapUnits(tx.journal.Units, tx.journal.TouchedUnits); err != nil {
		return fmt.Errorf("restore systemd runtime/enablement state: %w", err)
	}
	return tx.finish("rolled_back")
}

func (tx *bootstrapFileTransaction) journalTouchesSystemdUnits() bool {
	if tx == nil { return false }
	for _, image := range tx.journal.Files {
		if strings.HasPrefix(image.Path, "/etc/systemd/system/") { return true }
	}
	return false
}


func recoverBootstrapFileTransactions() error {
	if err := validateBootstrapJournalDirectory(bootstrapJournalRoot); err != nil {
		if os.IsNotExist(err) { return nil }
		return err
	}
	entries, err := os.ReadDir(bootstrapJournalRoot)
	if err != nil {
		if os.IsNotExist(err) { return nil }
		return fmt.Errorf("read bootstrap transaction directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(bootstrapJournalRoot, name)
		info, err := os.Lstat(path)
		if err != nil { return fmt.Errorf("inspect bootstrap journal entry %s: %w", path, err) }
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
			st.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("bootstrap journal entry %s is not a private regular file owned by euid %d", path, os.Geteuid())
		}
		if strings.HasPrefix(name, ".bootstrap-journal-") {
			if err := os.Remove(path); err != nil { return fmt.Errorf("remove abandoned journal temp %s: %w", path, err) }
			if err := syncBootstrapDirectory(bootstrapJournalRoot); err != nil { return err }
			continue
		}
		if !strings.HasSuffix(name, ".json") || len(name) != 37 {
			return fmt.Errorf("unexpected entry %q in bootstrap journal directory; refusing recovery", name)
		}
		if info.Size() <= 0 || info.Size() > maxBootstrapJournalBytes {
			return fmt.Errorf("bootstrap journal %s has invalid size %d", path, info.Size())
		}
		b, err := os.ReadFile(path)
		if err != nil { return fmt.Errorf("read bootstrap journal %s: %w", path, err) }
		var j bootstrapJournal
		if err := json.Unmarshal(b, &j); err != nil {
			return fmt.Errorf("bootstrap journal %s is corrupt; refusing bootstrap: %w", path, err)
		}
		if err := validateBootstrapJournal(j, name); err != nil {
			return fmt.Errorf("bootstrap journal %s is invalid; refusing recovery: %w", path, err)
		}
		seen := make(map[string]struct{}, len(j.Files))
		for _, image := range j.Files {
			if _, exists := seen[image.Path]; exists {
				return fmt.Errorf("bootstrap journal %s contains duplicate target %q", path, image.Path)
			}
			seen[image.Path] = struct{}{}
			if err := validateBootstrapSnapshotPathForIdentity(j.Role, j.Slot, image.Path); err != nil {
				return fmt.Errorf("bootstrap journal %s contains unsafe target: %w", path, err)
			}
		}
		tx := &bootstrapFileTransaction{journal: j, path: path}
		switch j.Phase {
		case "applying", "rolling_back":
			if err := tx.Rollback(); err != nil {
				return fmt.Errorf("recover bootstrap transaction %s (role=%s peer=%s slot=%d): %w", j.TxID, j.Role, j.PeerID, j.Slot, err)
			}
		case "committed", "rolled_back":
			if err := tx.removeJournal(); err != nil { return fmt.Errorf("clean completed bootstrap journal %s: %w", path, err) }
		default:
			return fmt.Errorf("bootstrap journal %s has unknown phase %q; refusing bootstrap", path, j.Phase)
		}
	}
	return nil
}

func validateBootstrapJournal(j bootstrapJournal, filename string) error {
	if (j.Version != 1 && j.Version != bootstrapJournalVersion) || j.Role != "primary" && j.Role != "secondary" ||
		j.Slot < 0 || j.Slot >= len(SlotTable()) {
		return fmt.Errorf("unsupported version or invalid role/slot")
	}
	raw, err := hex.DecodeString(j.TxID)
	if err != nil || len(raw) != 16 || filename != j.TxID+".json" {
		return fmt.Errorf("invalid transaction id or filename")
	}
	if len(j.PeerID) > 128 || strings.ContainsAny(j.PeerID, "/\\\\\x00\r\n") {
		return fmt.Errorf("invalid peer identity")
	}
	now := time.Now().UTC()
	if j.CreatedAt.IsZero() || j.UpdatedAt.IsZero() ||
		j.UpdatedAt.Before(j.CreatedAt) || j.CreatedAt.After(now.Add(5*time.Minute)) ||
		j.UpdatedAt.After(now.Add(5*time.Minute)) {
		return fmt.Errorf("invalid transaction timestamps")
	}
	switch j.Phase {
	case "applying", "rolling_back", "committed", "rolled_back":
	default:
		return fmt.Errorf("unknown phase %q", j.Phase)
	}
	if len(j.Files) > maxBootstrapJournalFiles {
		return fmt.Errorf("too many file images: %d", len(j.Files))
	}
	if len(j.Units) > 32 {
		return fmt.Errorf("too many systemd unit images: %d", len(j.Units))
	}
	seenUnits := make(map[string]bool, len(j.Units))
	for _, unit := range j.Units {
		if !safeUnitName(unit.Name) || seenUnits[unit.Name] {
			return fmt.Errorf("invalid or duplicate systemd unit image %q", unit.Name)
		}
		seenUnits[unit.Name] = true
		if unit.LoadState != "loaded" && unit.LoadState != "masked" && unit.LoadState != "not-found" {
			return fmt.Errorf("invalid LoadState for systemd unit %s: %q", unit.Name, unit.LoadState)
		}
		if unit.LoadState == "loaded" && (unit.FileState == "" || unit.ActiveState == "") {
			return fmt.Errorf("incomplete systemd state for unit %s", unit.Name)
		}
	}
	seenTouched := make(map[string]bool, len(j.TouchedUnits))
	for _, name := range j.TouchedUnits {
		if !safeUnitName(name) || !seenUnits[name] || seenTouched[name] {
			return fmt.Errorf("invalid, duplicate, or unsnapshotted touched unit %q", name)
		}
		seenTouched[name] = true
	}
	for _, image := range j.Files {
		if image.Path == "" { return fmt.Errorf("empty snapshot path") }
		if len(image.Data) > maxBootstrapImageBytes {
			return fmt.Errorf("snapshot %s exceeds %d bytes", image.Path, maxBootstrapImageBytes)
		}
		if image.Exists {
			if image.Mode&^os.FileMode(0o777) != 0 || image.Mode.Perm() == 0 ||
				image.UID < 0 || image.GID < 0 ||
				uint64(image.UID) > uint64(^uint32(0)) || uint64(image.GID) > uint64(^uint32(0)) {
				return fmt.Errorf("invalid metadata for snapshot %s", image.Path)
			}
		} else if len(image.Data) != 0 || image.Mode != 0 || image.UID != 0 || image.GID != 0 {
			return fmt.Errorf("non-existent snapshot %s carries file metadata", image.Path)
		}
	}
	return nil
}

func (tx *bootstrapFileTransaction) removeJournal() error {
	if err := os.Remove(tx.path); err != nil && !os.IsNotExist(err) { return err }
	return syncBootstrapDirectory(filepath.Dir(tx.path))
}

func (tx *bootstrapFileTransaction) Commit() error {
	if tx == nil { return fmt.Errorf("nil bootstrap file transaction") }
	return tx.finish("committed")
}

func (tx *bootstrapFileTransaction) finish(phase string) error {
	tx.journal.Phase = phase
	if err := tx.persist(); err != nil { return fmt.Errorf("checkpoint %s: %w", phase, err) }
	if err := tx.removeJournal(); err != nil { return fmt.Errorf("remove completed bootstrap journal: %w", err) }
	return nil
}

// validateBootstrapSnapshotPathForIdentity binds every journal image to the
// exact files that the recorded role and slot may mutate. Test-only roots remain
// injectable through bootstrapAllowedFileRoots; production paths are exact.
func validateBootstrapSnapshotPathForIdentity(role string, slot int, path string) error {
	if slot < 0 || slot >= len(SlotTable()) {
		return fmt.Errorf("invalid bootstrap transaction slot %d", slot)
	}
	if err := validateBootstrapSnapshotPath(path); err != nil { return err }
	for _, root := range bootstrapAllowedFileRoots {
		if root != "/etc/wireguard" && root != "/etc/systemd/system" &&
			root != "/etc/netductor" && root != "/usr/local/bin" && root != "/var/lib/netductor" {
			rel, err := filepath.Rel(root, path)
			if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				return nil
			}
		}
	}
	s := SlotTable()[slot]
	allowed := map[string]bool{
		filepath.Join("/etc/wireguard", s.IfaceSP+".conf"): true,
		filepath.Join("/etc/wireguard", s.IfacePS+".conf"): true,
		"/usr/local/bin/nd-svc-paths-health": true,
		"/etc/systemd/system/nd-svc-paths-health.service": true,
		"/etc/systemd/system/nd-svc-paths-health.timer": true,
	}
	if role == "primary" {
		spUnit := "nd-wss-sp-server"
		psUnit := "nd-wss-ps-client"
		if s.Index != 0 {
			spUnit += "-" + s.IfaceSP
			psUnit += "-" + s.IfacePS
		}
		allowed[filepath.Join("/etc/systemd/system", spUnit+".service")] = true
		allowed[filepath.Join("/etc/systemd/system", psUnit+".service")] = true
		if s.Index == 0 {
			allowed[filepath.Join(KeysDir, "material.json")] = true
		}
		allowed[policyPath] = true
	} else if role == "secondary" {
		allowed["/etc/systemd/system/nd-wss-sp-client.service"] = true
		allowed["/etc/systemd/system/nd-wss-ps-server.service"] = true
		allowed[filepath.Join(KeysDir, "material.json")] = true
		allowed["/etc/netductor/secrets/secondary_core_url.public"] = true
		allowed["/etc/netductor/secrets/secondary_core_url"] = true
		allowed[policyPath] = true
	} else {
		return fmt.Errorf("invalid bootstrap transaction role %q", role)
	}
	if !allowed[path] {
		return fmt.Errorf("bootstrap %s slot %d may not snapshot %q", role, slot, path)
	}
	return nil
}

func validateBootstrapSnapshotPath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("bootstrap snapshot path must be clean and absolute: %q", path)
	}
	allowed := false
	for _, root := range bootstrapAllowedFileRoots {
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			allowed = true
			break
		}
	}
	if !allowed { return fmt.Errorf("bootstrap snapshot path is outside Netductor-managed roots: %q", path) }
	if rel, err := filepath.Rel(bootstrapJournalRoot, path); err == nil &&
		(rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))) {
		return fmt.Errorf("bootstrap snapshot path must not target the journal store: %q", path)
	}
	return validateExistingDirectoryChain(filepath.Dir(path))
}

func validateExistingDirectoryChain(dir string) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir { return fmt.Errorf("directory path must be clean and absolute: %q", dir) }
	current := string(os.PathSeparator)
	for _, part := range strings.Split(strings.TrimPrefix(dir, current), current) {
		if part == "" { continue }
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil { return err }
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() { return fmt.Errorf("path component %s is not a real directory", current) }
	}
	return nil
}

// ensureBootstrapJournalDirectory never creates or traverses an unchecked parent.
// The parent must already exist as a private directory owned by the bootstrap euid.
func ensureBootstrapJournalDirectory(dir string) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || dir == string(os.PathSeparator) {
		return fmt.Errorf("bootstrap journal path must be clean, absolute, and non-root: %q", dir)
	}
	parent := filepath.Dir(dir)
	if err := validateExistingDirectoryChain(parent); err != nil {
		return fmt.Errorf("unsafe bootstrap journal parent: %w", err)
	}
	info, err := os.Lstat(parent)
	if err != nil {
		return fmt.Errorf("inspect bootstrap journal parent: %w", err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		st.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("bootstrap journal parent must be a real private directory owned by euid %d", os.Geteuid())
	}

	if err := os.Mkdir(dir, 0o700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("create bootstrap journal directory: %w", err)
	}
	if err := validateBootstrapJournalDirectory(dir); err != nil {
		return err
	}
	return nil
}

func validateBootstrapJournalDirectory(dir string) error {
	if err := validateExistingDirectoryChain(dir); err != nil { return fmt.Errorf("unsafe bootstrap journal directory: %w", err) }
	info, err := os.Lstat(dir)
	if err != nil { return fmt.Errorf("inspect bootstrap journal directory: %w", err) }
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		st.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("bootstrap journal directory must be a real private directory owned by euid %d", os.Geteuid())
	}
	return nil
}

func restoreBootstrapFile(image bootstrapFileImage) error {
	if err := validateBootstrapSnapshotPath(image.Path); err != nil { return err }
	dir := filepath.Dir(image.Path)
	if !image.Exists {
		if err := validateExistingDirectoryChain(dir); err != nil {
			if os.IsNotExist(err) { return nil }
			return err
		}
		if err := os.Remove(image.Path); err != nil && !os.IsNotExist(err) { return err }
		return syncBootstrapDirectory(dir)
	}
	if err := validateExistingDirectoryChain(dir); err != nil { return fmt.Errorf("unsafe rollback parent: %w", err) }
	if err := validateAtomicWriteTarget(image.Path); err != nil { return err }
	f, err := os.CreateTemp(dir, ".netductor-rollback-*")
	if err != nil { return err }
	tmp := f.Name()
	cleanup := func() { _ = f.Close(); _ = os.Remove(tmp) }
	if err := f.Chown(image.UID, image.GID); err != nil { cleanup(); return err }
	if err := f.Chmod(image.Mode); err != nil { cleanup(); return err }
	if n, err := f.Write(image.Data); err != nil { cleanup(); return err } else if n != len(image.Data) { cleanup(); return fmt.Errorf("short restore write") }
	if err := f.Sync(); err != nil { cleanup(); return err }
	if err := f.Close(); err != nil { _ = os.Remove(tmp); return err }
	if err := os.Rename(tmp, image.Path); err != nil { _ = os.Remove(tmp); return err }
	return syncBootstrapDirectory(dir)
}

func syncBootstrapDirectory(path string) error {
	d, err := os.Open(path)
	if err != nil { return err }
	defer d.Close()
	return d.Sync()
}
