package integrity

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

const AllowlistPath = "/etc/netductor/ssh-allowed.pub"

type LiveKey struct {
	Host        string `json:"host"` // primary | secondary id
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
	Line        string `json:"line"` // full authorized_keys line
	InAllowlist bool   `json:"in_allowlist"`
}

type AllowlistState struct {
	Enforce   bool   `json:"enforce"`
	UpdatedAt string `json:"updated_at,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

func allowlistStatePath() string {
	return filepath.Join(paths.EtcDir(), "ssh-enforce.json")
}

// LiveKeysPrimary reads local authorized_keys with full lines.
func LiveKeysPrimary() []LiveKey {
	path := authKeysPath()
	return liveKeysFromPath(path, "primary")
}

func liveKeysFromPath(path, host string) []LiveKey {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	allow := map[string]bool{}
	for _, k := range ParseAuthorizedKeys(AllowlistPath) {
		allow[k.Fingerprint] = true
	}
	var out []LiveKey
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fp, typ, comment := keyFingerprint(line)
		if fp == "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				sum := sha256.Sum256([]byte(parts[1]))
				fp = "SHA256:" + hex.EncodeToString(sum[:])
				typ = parts[0]
				if len(parts) >= 3 {
					comment = parts[len(parts)-1]
				}
			}
		}
		if fp == "" {
			continue
		}
		out = append(out, LiveKey{
			Host: host, Type: typ, Fingerprint: fp, Comment: comment, Line: line,
			InAllowlist: allow[fp],
		})
	}
	return out
}

// ParseKeysFromText parses authorized_keys body (e.g. agent cmd log).
func ParseKeysFromText(host, body string) []LiveKey {
	_ = os.WriteFile(filepath.Join(os.TempDir(), "nd-ak-parse"), []byte(body), 0o600)
	// use line loop
	allow := map[string]bool{}
	for _, k := range ParseAuthorizedKeys(AllowlistPath) {
		allow[k.Fingerprint] = true
	}
	var out []LiveKey
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "ssh-") && !strings.HasPrefix(line, "ecdsa-") {
			continue
		}
		fp, typ, comment := keyFingerprint(line)
		if fp == "" {
			continue
		}
		out = append(out, LiveKey{Host: host, Type: typ, Fingerprint: fp, Comment: comment, Line: line, InAllowlist: allow[fp]})
	}
	return out
}

func ReadAllowlistLines() []string {
	b, err := os.ReadFile(AllowlistPath)
	if err != nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func ReadAllowlistState() AllowlistState {
	b, err := os.ReadFile(allowlistStatePath())
	if err != nil {
		return AllowlistState{}
	}
	var st AllowlistState
	_ = json.Unmarshal(b, &st)
	return st
}

func writeAllowlistState(st AllowlistState) error {
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = os.MkdirAll(filepath.Dir(allowlistStatePath()), 0o755)
	b, _ := json.MarshalIndent(st, "", "  ")
	return os.WriteFile(allowlistStatePath(), b, 0o600)
}

// SetAllowlistByFingerprints keeps lines whose fp is in selected (from candidates map fp→line).
func SetAllowlistByFingerprints(selected []string, candidates []LiveKey, enforce bool, by string) error {
	want := map[string]bool{}
	for _, s := range selected {
		want[s] = true
	}
	byLine := map[string]string{}
	for _, k := range candidates {
		if k.Line != "" {
			byLine[k.Fingerprint] = k.Line
		}
	}
	// also keep existing allowlist lines if selected
	for _, line := range ReadAllowlistLines() {
		fp, _, _ := keyFingerprint(line)
		if fp != "" {
			byLine[fp] = line
		}
	}
	var lines []string
	seen := map[string]bool{}
	for fp := range want {
		line := byLine[fp]
		if line == "" || seen[fp] {
			continue
		}
		seen[fp] = true
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return fmt.Errorf("no keys selected")
	}
	body := strings.Join(lines, "\n") + "\n"
	_ = os.MkdirAll(filepath.Dir(AllowlistPath), 0o755)
	if err := os.WriteFile(AllowlistPath, []byte(body), 0o600); err != nil {
		return err
	}
	st := ReadAllowlistState()
	st.Enforce = enforce
	st.UpdatedBy = by
	if err := writeAllowlistState(st); err != nil {
		return err
	}
	if enforce {
		if err := EnforceSSHAllowed(); err != nil {
			return err
		}
	}
	return WriteManifest()
}

// MergeAllowlist appends fingerprints not already present.
func MergeAllowlist(selected []string, candidates []LiveKey, by string) error {
	existing := map[string]bool{}
	for _, k := range ParseAuthorizedKeys(AllowlistPath) {
		existing[k.Fingerprint] = true
	}
	lines := ReadAllowlistLines()
	byLine := map[string]string{}
	for _, k := range candidates {
		byLine[k.Fingerprint] = k.Line
	}
	for _, fp := range selected {
		if existing[fp] {
			continue
		}
		line := byLine[fp]
		if line == "" {
			continue
		}
		lines = append(lines, line)
		existing[fp] = true
	}
	if len(lines) == 0 {
		return fmt.Errorf("nothing to merge")
	}
	body := strings.Join(lines, "\n") + "\n"
	_ = os.MkdirAll(filepath.Dir(AllowlistPath), 0o755)
	if err := os.WriteFile(AllowlistPath, []byte(body), 0o600); err != nil {
		return err
	}
	st := ReadAllowlistState()
	st.UpdatedBy = by
	_ = writeAllowlistState(st)
	return WriteManifest()
}

// AllowlistB64 returns base64 of ssh-allowed.pub for agent push.
func AllowlistB64() (string, error) {
	b, err := os.ReadFile(AllowlistPath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// ApplyAllowlistFromB64 writes allowlist and enforces (agent side).
func ApplyAllowlistFromB64(b64 string) error {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(AllowlistPath), 0o755)
	if err := os.WriteFile(AllowlistPath, raw, 0o600); err != nil {
		return err
	}
	return EnforceSSHAllowed()
}

// AuthorizedKeysBody returns raw authorized_keys for agent ssh_keys cmd.
func AuthorizedKeysBody() string {
	b, err := os.ReadFile(authKeysPath())
	if err != nil {
		return ""
	}
	return string(b)
}

func secondaryKeysCachePath(id string) string {
	return filepath.Join(paths.StateDir(), "ssh-keys-cache", id+".pub")
}

// CacheSecondaryKeys stores raw authorized_keys body from agent ssh_keys cmd.
func CacheSecondaryKeys(id, body string) error {
	id = strings.TrimSpace(id)
	if id == "" || strings.TrimSpace(body) == "" {
		return fmt.Errorf("empty")
	}
	dir := filepath.Dir(secondaryKeysCachePath(id))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(secondaryKeysCachePath(id), []byte(body), 0o600)
}

// LiveKeysSecondaryCached returns keys from last ssh_keys probe.
func LiveKeysSecondaryCached(id string) []LiveKey {
	b, err := os.ReadFile(secondaryKeysCachePath(id))
	if err != nil {
		return nil
	}
	return ParseKeysFromText(id, string(b))
}
