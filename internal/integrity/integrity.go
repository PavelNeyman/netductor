// Package integrity tracks hashes of critical paths and SSH authorized_keys fingerprints.
package integrity

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// DefaultPaths hashed on snapshot/check (missing files are noted, not fatal).
func DefaultPaths() []string {
	return []string{
		"/usr/local/bin/netductor",
		"/usr/local/bin/netductor-tg",
		"/usr/local/bin/netductor-agent",
		"/usr/local/bin/sing-box",
		"/usr/local/bin/blocky",
		"/etc/netductor/role",
		"/etc/netductor/VERSION",
		"/etc/systemd/system/netductor-api.service",
		"/etc/systemd/system/netductor-telegram-bot.service",
		"/etc/systemd/system/sing-box.service",
		"/etc/ssh/sshd_config",
	}
}

type Manifest struct {
	Version   int               `json:"version"`
	Created   string            `json:"created"`
	Files     map[string]string `json:"files"` // path → sha256 or "MISSING"
	SSHKeys   []SSHKey          `json:"ssh_keys"`
	SSHSource string            `json:"ssh_source"` // path of authorized_keys
}

type SSHKey struct {
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"` // sha256:…
	Comment     string `json:"comment,omitempty"`
}

type Drift struct {
	OK           bool     `json:"ok"`
	HaveManifest bool     `json:"have_manifest"`
	ChangedFiles []string `json:"changed_files,omitempty"`
	NewSSH       []string `json:"new_ssh,omitempty"`
	MissingSSH   []string `json:"missing_ssh,omitempty"`
	Notes        []string `json:"notes,omitempty"`
}

func manifestPath() string {
	return filepath.Join(paths.StateDir(), "baseline", "integrity.json")
}

func authKeysPath() string {
	// netductor SSH is typically root
	for _, p := range []string{"/root/.ssh/authorized_keys", "/etc/ssh/authorized_keys"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/root/.ssh/authorized_keys"
}

func fileSHA256(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ParseAuthorizedKeys returns key fingerprints (ssh-keygen -lf preferred).
func ParseAuthorizedKeys(path string) []SSHKey {
	var out []SSHKey
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// try ssh-keygen -lf via temp line
		fp, typ, comment := keyFingerprint(line)
		if fp == "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				sum := sha256.Sum256([]byte(parts[1]))
				fp = "sha256:" + hex.EncodeToString(sum[:8]) + "…" // truncated fallback
				typ = parts[0]
				if len(parts) >= 3 {
					comment = parts[len(parts)-1]
				}
			}
		}
		if fp != "" {
			out = append(out, SSHKey{Type: typ, Fingerprint: fp, Comment: comment})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fingerprint < out[j].Fingerprint })
	return out
}

func keyFingerprint(line string) (fp, typ, comment string) {
	tmp, err := os.CreateTemp("", "ak-*")
	if err != nil {
		return "", "", ""
	}
	path := tmp.Name()
	_, _ = tmp.WriteString(line + "\n")
	_ = tmp.Close()
	defer os.Remove(path)
	out, err := exec.Command("ssh-keygen", "-lf", path).CombinedOutput()
	if err != nil {
		return "", "", ""
	}
	// "256 SHA256:xxxx comment (ED25519)"
	fields := strings.Fields(string(out))
	if len(fields) < 2 {
		return "", "", ""
	}
	fp = fields[1]
	if len(fields) >= 3 {
		comment = fields[2]
	}
	if i := strings.Index(string(out), "("); i >= 0 {
		typ = strings.Trim(string(out)[i:], "()\n\r ")
	}
	return fp, typ, comment
}

// WriteManifest snapshots hashes + SSH keys (call after install / stack apply / host-baseline).
func WriteManifest() error {
	m := Manifest{
		Version:   1,
		Created:   time.Now().UTC().Format(time.RFC3339),
		Files:     map[string]string{},
		SSHSource: authKeysPath(),
	}
	for _, p := range DefaultPaths() {
		sum, err := fileSHA256(p)
		if err != nil {
			m.Files[p] = "MISSING"
			continue
		}
		m.Files[p] = sum
	}
	m.SSHKeys = ParseAuthorizedKeys(m.SSHSource)
	dir := filepath.Dir(manifestPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(manifestPath(), b, 0o600)
}

// Check compares live state to manifest.
func Check() Drift {
	d := Drift{OK: true}
	b, err := os.ReadFile(manifestPath())
	if err != nil {
		d.HaveManifest = false
		d.Notes = append(d.Notes, "no integrity manifest (written at install / host-baseline / stack apply)")
		return d
	}
	d.HaveManifest = true
	var m Manifest
	if json.Unmarshal(b, &m) != nil {
		d.OK = false
		d.Notes = append(d.Notes, "integrity manifest corrupt")
		return d
	}
	for p, want := range m.Files {
		sum, err := fileSHA256(p)
		if err != nil {
			if want != "MISSING" {
				d.ChangedFiles = append(d.ChangedFiles, p+" (now missing)")
				d.OK = false
			}
			continue
		}
		if want == "MISSING" || want != sum {
			d.ChangedFiles = append(d.ChangedFiles, p)
			d.OK = false
		}
	}
	live := ParseAuthorizedKeys(authKeysPath())
	wantFP := map[string]bool{}
	for _, k := range m.SSHKeys {
		wantFP[k.Fingerprint] = true
	}
	liveFP := map[string]bool{}
	for _, k := range live {
		liveFP[k.Fingerprint] = true
		if !wantFP[k.Fingerprint] {
			d.NewSSH = append(d.NewSSH, fmt.Sprintf("%s %s", k.Fingerprint, k.Comment))
			d.OK = false
		}
	}
	for _, k := range m.SSHKeys {
		if !liveFP[k.Fingerprint] {
			d.MissingSSH = append(d.MissingSSH, fmt.Sprintf("%s %s", k.Fingerprint, k.Comment))
			// missing key is informational — not necessarily attack
		}
	}
	sort.Strings(d.ChangedFiles)
	sort.Strings(d.NewSSH)
	return d
}

// EnforceSSHAllowed if /etc/netductor/ssh-allowed.pub exists, replace authorized_keys with it.
func EnforceSSHAllowed() error {
	src := "/etc/netductor/ssh-allowed.pub"
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("no %s — skip enforce", src)
	}
	dst := authKeysPath()
	_ = os.MkdirAll(filepath.Dir(dst), 0o700)
	if err := os.WriteFile(dst, b, 0o600); err != nil {
		return err
	}
	return WriteManifest()
}
