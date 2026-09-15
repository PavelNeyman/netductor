package vpn

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/session"
)

type UserRecord struct {
	Name        string `json:"name"`
	UUID        string `json:"uuid"`
	Hy2Password string `json:"hy2_password"`
	Enabled     bool   `json:"enabled"`
	Note        string `json:"note"`
	Created     string `json:"created"`
}

type registry struct {
	Users []UserRecord `json:"users"`
}

func usersFile() string {
	return filepath.Join(paths.EtcDir(), "vpn-users.json")
}

func EnsureDirs() error {
	etc := paths.EtcDir()
	for _, d := range []string{
		filepath.Join(etc, "secrets"),
		Clients(),
		filepath.Join(paths.EtcDir(), "sessions"),
	} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	if _, err := os.Stat(usersFile()); err != nil {
		return writeRegistry(&registry{Users: []UserRecord{}})
	}
	return nil
}

func loadRegistry() (*registry, error) {
	_ = EnsureDirs()
	b, err := os.ReadFile(usersFile())
	if err != nil {
		return &registry{Users: []UserRecord{}}, nil
	}
	var r registry
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	if r.Users == nil {
		r.Users = []UserRecord{}
	}
	return &r, nil
}

func writeRegistry(r *registry) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := usersFile() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, usersFile())
}

func secret(name string) string {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}


// advertiseHost is the host put into client share links (domain preferred).
func advertiseHost() string {
	if h := strings.TrimSpace(os.Getenv("NETDUCTOR_VPN_HOST")); h != "" {
		return h
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "vpn_hostname")); err == nil {
		if h := strings.TrimSpace(string(b)); h != "" {
			return h
		}
	}
	// core-facing hostname (not RU vpn) for core-only links
	if h := strings.TrimSpace(os.Getenv("NETDUCTOR_PUBLIC_HOSTNAME")); h != "" {
		return h
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_hostname")); err == nil {
		if h := strings.TrimSpace(string(b)); h != "" {
			return h
		}
	}
	return publicIP()
}

func publicIP() string {
	if v := os.Getenv("PUBLIC_IP"); v != "" {
		return v
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_ip")); err == nil {
		return strings.TrimSpace(string(b))
	}
	out, err := exec.Command("curl", "-4", "-fsS", "--max-time", "5", "https://ifconfig.me").Output()
	if err != nil {
		return "YOUR_IP"
	}
	return strings.TrimSpace(string(out))
}

func sni() string {
	if v := os.Getenv("SINGBOX_REALITY_SNI"); v != "" {
		return v
	}
	// persisted at install / vpn set-sni
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", "singbox_reality_sni")); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v
		}
	}
	// RU whitelist-oriented default (home ISP). Mobile often needs RU relay.
	// Override: SINGBOX_REALITY_SNI or netductor vpn set-sni.
	return DefaultRealitySNI
}

func vlessPort() int {
	if v := os.Getenv("SINGBOX_VLESS_PORT"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		if n > 0 {
			return n
		}
	}
	return 443
}

func hy2Port() int {
	if v := os.Getenv("SINGBOX_HY2_PORT"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		if n > 0 {
			return n
		}
	}
	return 8443
}

func genUUID() string {
	if Bin() != "" {
		if out, err := exec.Command(Bin(), "generate", "uuid").Output(); err == nil {
			u := strings.TrimSpace(string(out))
			if u != "" {
				return u
			}
		}
	}
	b, err := os.ReadFile("/proc/sys/kernel/random/uuid")
	if err == nil {
		return strings.TrimSpace(string(b))
	}
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}

func genHy2Pass() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func VLESSLink(name, uuid string) string {
	return fmt.Sprintf(
		"vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=%s&pbk=%s&sid=%s&type=tcp#nd-core",
		uuid, advertiseHost(), vlessPort(), sni(), DefaultUTLSFingerprint, secret("singbox_reality_public"), secret("singbox_short_id"),
	)
}

// PreferredVLESSLink uses online RU relay when available (primary under carrier WL).
func PreferredVLESSLink(name, uuid string) string {
	e := ResolveClientEndpoints(name, uuid)
	if e.RelayHost != "" && e.RelayPBK != "" {
		return ClientLinkForRelayLocal(name, uuid, e.RelayHost, e.RelayPBK, e.RelaySID, e.RelaySNI)
	}
	return VLESSLink(name, uuid)
}

func Hy2Link(name, pass string) string {
	return fmt.Sprintf("hysteria2://%s@%s:%d?sni=%s&insecure=1#nd-hy2",
		pass, advertiseHost(), hy2Port(), sni())
}

func writeArtifacts(name, uuid, hy2pass string) error {
	dir := filepath.Join(Clients(), name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	vless := PreferredVLESSLink(name, uuid)
	vlessCore := VLESSLink(name, uuid)
	hy2 := Hy2Link(name, hy2pass)
	nl := string([]byte{10})
	// Primary subscription: VLESS only (relay-first + core). HY2 optional under WL.
	sub := vless + nl + vlessCore + nl
	_ = os.WriteFile(filepath.Join(dir, "link-vless.txt"), []byte(vless+nl), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "link-vless-core.txt"), []byte(vlessCore+nl), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "link-hy2.txt"), []byte(hy2+nl), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "subscription.txt"), []byte(sub), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "subscription-full.txt"), []byte(sub+hy2+nl), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "link.txt"), []byte(vless+nl), 0o600)
	_ = qrcode.WriteFile(vless, qrcode.Medium, 512, filepath.Join(dir, "qr.png"))
	_ = qrcode.WriteFile(vless, qrcode.Medium, 512, filepath.Join(dir, "qr-vless.png"))
	_ = qrcode.WriteFile(hy2, qrcode.Medium, 512, filepath.Join(dir, "qr-hy2.png"))
	_ = qrcode.WriteFile(sub, qrcode.Medium, 512, filepath.Join(dir, "qr-subscription.png"))
	_ = os.Chmod(filepath.Join(dir, "qr.png"), 0o600)
	_ = os.Chmod(filepath.Join(dir, "qr-subscription.png"), 0o600)
	_ = WriteClientConfigs(name, uuid)
	return nil
}


// RenameNative changes display name only. UUID and credentials stay the same; client links keep working.
func RenameNative(oldName, newName string) error {
	if !ValidName(oldName) || !ValidName(newName) {
		return fmt.Errorf("bad name")
	}
	if oldName == newName {
		return nil
	}
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	u := findUser(r, oldName)
	if u == nil {
		return fmt.Errorf("user not found: %s", oldName)
	}
	if findUser(r, newName) != nil {
		return fmt.Errorf("user already exists: %s", newName)
	}
	uuid, hy2 := u.UUID, u.Hy2Password
	u.Name = newName
	if err := writeRegistry(r); err != nil {
		return err
	}
	oldDir := filepath.Join(Clients(), oldName)
	newDir := filepath.Join(Clients(), newName)
	if st, err := os.Stat(oldDir); err == nil && st.IsDir() {
		_ = os.RemoveAll(newDir)
		if err := os.Rename(oldDir, newDir); err != nil {
			// fallback: rewrite into new dir
			_ = os.MkdirAll(newDir, 0o700)
		}
	}
	if err := writeArtifacts(newName, uuid, hy2); err != nil {
		return err
	}
	// UUID unchanged — ApplyConfig optional (may fail in tests without sing-box)
	_ = ApplyConfig()
	return nil
}

func findUser(r *registry, name string) *UserRecord {
	for i := range r.Users {
		if r.Users[i].Name == name {
			return &r.Users[i]
		}
	}
	return nil
}

func AddNative(name, note string) (string, error) {
	if !ValidName(name) {
		return "", fmt.Errorf("bad name")
	}
	r, err := loadRegistry()
	if err != nil {
		return "", err
	}
	if findUser(r, name) != nil {
		return "", fmt.Errorf("user already exists: %s", name)
	}
	uuid := genUUID()
	hy2 := genHy2Pass()
	r.Users = append(r.Users, UserRecord{
		Name: name, UUID: uuid, Hy2Password: hy2, Enabled: true, Note: note, Created: time.Now().Format(time.RFC3339),
	})
	if err := writeRegistry(r); err != nil {
		return "", err
	}
	if err := writeArtifacts(name, uuid, hy2); err != nil {
		return "", err
	}
	if err := ApplyConfig(); err != nil {
		return name + " " + uuid, err
	}
	return name + " " + uuid, nil
}

func SetNoteNative(name, note string) error {
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	u := findUser(r, name)
	if u == nil {
		return fmt.Errorf("user not found: %s", name)
	}
	u.Note = note
	return writeRegistry(r)
}

func SetEnabledNative(name string, enabled bool) error {
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	u := findUser(r, name)
	if u == nil {
		return fmt.Errorf("user not found: %s", name)
	}
	u.Enabled = enabled
	if err := writeRegistry(r); err != nil {
		return err
	}
	return ApplyConfig()
}

func RevokeNative(name string) error {
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	out := r.Users[:0]
	found := false
	for _, u := range r.Users {
		if u.Name == name {
			found = true
			continue
		}
		out = append(out, u)
	}
	if !found {
		return fmt.Errorf("user not found: %s", name)
	}
	r.Users = out
	if err := writeRegistry(r); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(Clients(), name))
	return ApplyConfig()
}

func ListNative() ([]User, error) {
	r, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(r.Users))
	for _, u := range r.Users {
		users = append(users, User{Name: u.Name, Enabled: u.Enabled, UUID: u.UUID, Note: u.Note, Created: u.Created})
	}
	return users, nil
}

func CreateSession(hours int) (token string, exp int64, err error) {
	return session.Create(hours, "cli", "")
}


// SetSNI stores Reality/HY2 SNI and rewrites client links + server config.
func SetSNI(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty sni")
	}
	dir := filepath.Join(paths.EtcDir(), "secrets")
	_ = os.MkdirAll(dir, 0o700)
	line := value + string([]byte{10})
	if err := os.WriteFile(filepath.Join(dir, "singbox_reality_sni"), []byte(line), 0o600); err != nil {
		return err
	}
	if err := ApplyConfig(); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "restart", "sing-box").Run()
	if _, err := ExportRelayBundle(value); err != nil {
		// core-only OK if no relay users yet
		_ = err
	}
	if err := RewriteAllLinks(); err != nil {
		return err
	}
	return nil
}

func RewriteAllLinks() error {
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	for _, u := range r.Users {
		if err := writeArtifacts(u.Name, u.UUID, u.Hy2Password); err != nil {
			return err
		}
	}
	return nil
}
