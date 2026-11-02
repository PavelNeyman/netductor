package svcpaths

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const bootstrapJournalVersion = 1

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
	if err := os.MkdirAll(bootstrapJournalRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create bootstrap journal directory: %w", err)
	}
	if err := validateBootstrapJournalDirectory(bootstrapJournalRoot); err != nil { return nil, err }
	info, err := os.Lstat(bootstrapJournalRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect bootstrap journal directory: %w", err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		st.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("bootstrap journal directory must be a real private directory owned by euid %d", os.Geteuid())
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
	return tx.finish("rolled_back")
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
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" { continue }
		path := filepath.Join(bootstrapJournalRoot, entry.Name())
		info, err := os.Lstat(path)
		if err != nil { return fmt.Errorf("inspect bootstrap journal %s: %w", path, err) }
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
			st.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("bootstrap journal %s is not a private regular file owned by euid %d", path, os.Geteuid())
		}
		b, err := os.ReadFile(path)
		if err != nil { return fmt.Errorf("read bootstrap journal %s: %w", path, err) }
		var j bootstrapJournal
		if err := json.Unmarshal(b, &j); err != nil {
			return fmt.Errorf("bootstrap journal %s is corrupt; refusing bootstrap: %w", path, err)
		}
		if j.Version != bootstrapJournalVersion || j.TxID == "" ||
			filepath.Base(path) != j.TxID+".json" ||
			(j.Role != "primary" && j.Role != "secondary") ||
			j.Slot < 0 || j.Slot >= len(SlotTable()) {
			return fmt.Errorf("bootstrap journal %s has unsupported or invalid identity; refusing bootstrap", path)
		}
		for _, image := range j.Files {
			if err := validateBootstrapSnapshotPathForIdentity(j.Role, j.Slot, image.Path); err != nil {
				return fmt.Errorf("bootstrap journal %s contains unsafe target: %w", path, err)
			}
			if image.Exists && (image.Mode&^os.FileMode(0o777) != 0 || image.Mode.Perm() == 0) {
				return fmt.Errorf("bootstrap journal %s contains invalid file metadata for %s", path, image.Path)
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
