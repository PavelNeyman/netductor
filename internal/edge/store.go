package edge

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var mu sync.Mutex

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusDenied   = "denied"
	StatusRevoked  = "revoked"
)

func ensure() error {
	return os.MkdirAll(paths.EdgeDir(), 0o700)
}

// BootstrapToken — shared secret only for enroll (not full access).
func BootstrapToken() string {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", "edge_bootstrap_token"))
	if err != nil {
		// fallback to legacy edge_token for transition
		return Token()
	}
	return string(bytesTrim(b))
}

func Token() string {
	b, err := os.ReadFile(paths.EdgeTokenFile())
	if err != nil {
		return ""
	}
	return string(bytesTrim(b))
}

func bytesTrim(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}

func constEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func bearerRaw(auth string) string {
	const p = "Bearer "
	if len(auth) > len(p) && auth[:len(p)] == p {
		return auth[len(p):]
	}
	return ""
}

// ValidBearer: approved per-device token.
// Global edge_token is denied unless NETDUCTOR_EDGE_LEGACY_TOKEN=1 (migration only).
func ValidBearer(auth string) bool {
	raw := bearerRaw(auth)
	if raw == "" || len(raw) < 32 {
		return false
	}
	if os.Getenv("NETDUCTOR_EDGE_LEGACY_TOKEN") == "1" {
		if tok := Token(); tok != "" && constEq(raw, tok) {
			return true
		}
	}
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	for _, d := range m {
		if d.DeviceToken != "" && constEq(raw, d.DeviceToken) && d.Status == StatusApproved {
			return true
		}
	}
	return false
}

// DeviceIDFromAuth returns device_id if auth is a device token.
func DeviceIDFromAuth(auth string) string {
	raw := bearerRaw(auth)
	if raw == "" {
		return ""
	}
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	for id, d := range m {
		if d.DeviceToken != "" && constEq(raw, d.DeviceToken) {
			return id
		}
	}
	return ""
}

func ValidBootstrap(auth string) bool {
	raw := bearerRaw(auth)
	bt := BootstrapToken()
	if raw == "" || bt == "" || len(bt) < 32 {
		return false
	}
	return constEq(raw, bt)
}

// RequireApproved: device token must map to approved device_id matching claim.
func RequireApproved(auth, deviceID string) bool {
	raw := bearerRaw(auth)
	if raw == "" || deviceID == "" {
		return false
	}
	if os.Getenv("NETDUCTOR_EDGE_LEGACY_TOKEN") == "1" {
		if tok := Token(); tok != "" && constEq(raw, tok) {
			return true
		}
	}
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return false
	}
	return d.Status == StatusApproved && d.DeviceToken != "" && constEq(raw, d.DeviceToken)
}


// Device is a registered edge host (OpenWrt / MikroTik / …).
type Device struct {
	DeviceID    string         `json:"device_id,omitempty"`
	Status      string         `json:"status,omitempty"`
	DeviceToken string         `json:"device_token,omitempty"`
	Board       string         `json:"board,omitempty"`
	Hostname    string         `json:"hostname,omitempty"`
	WANIP       string         `json:"wan_ip,omitempty"`
	TemplateID  string         `json:"template_id,omitempty"`
	Overlay     map[string]any `json:"overlay,omitempty"`
	EnrolledAt  int64          `json:"enrolled_at,omitempty"`
	ApprovedAt  int64          `json:"approved_at,omitempty"`
	DeniedAt    int64          `json:"denied_at,omitempty"`
	RevokedAt   int64          `json:"revoked_at,omitempty"`
	LastSeen    int64          `json:"last_seen,omitempty"`
	Healthy     bool           `json:"healthy,omitempty"`
	Agent       string         `json:"agent,omitempty"`
	UptimeSec   float64        `json:"uptime_sec,omitempty"`
	MemPct      float64        `json:"mem_pct,omitempty"`
	// Extra keeps unknown agent fields without losing them on disk.
	Extra map[string]any `json:"extra,omitempty"`
}

func devicesPath() string  { return filepath.Join(paths.EdgeDir(), "devices.json") }
func commandsPath() string { return filepath.Join(paths.EdgeDir(), "commands.json") }
func resultsPath() string  { return filepath.Join(paths.EdgeDir(), "results.jsonl") }

func loadDevices() map[string]Device {
	_ = ensure()
	b, err := os.ReadFile(devicesPath())
	if err != nil {
		return map[string]Device{}
	}
	var wrap struct {
		Devices map[string]Device `json:"devices"`
	}
	if json.Unmarshal(b, &wrap) != nil || wrap.Devices == nil {
		return map[string]Device{}
	}
	return wrap.Devices
}

func saveDevices(m map[string]Device) error {
	_ = ensure()
	b, _ := json.MarshalIndent(map[string]any{"devices": m}, "", "  ")
	tmp := devicesPath() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, devicesPath())
}

func strFrom(payload map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := payload[k]; ok {
			switch t := v.(type) {
			case string:
				return t
			}
		}
	}
	return ""
}

func floatFrom(payload map[string]any, key string) float64 {
	v, ok := payload[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	}
	return 0
}

func mergePayload(d *Device, payload map[string]any) {
	if s := strFrom(payload, "board"); s != "" {
		d.Board = s
	}
	if s := strFrom(payload, "hostname"); s != "" {
		d.Hostname = s
	}
	if s := strFrom(payload, "wan_ip"); s != "" {
		d.WANIP = s
	}
	if s := strFrom(payload, "agent"); s != "" {
		d.Agent = s
	}
	if v := floatFrom(payload, "uptime_sec"); v > 0 {
		d.UptimeSec = v
	}
	if v := floatFrom(payload, "mem_pct"); v > 0 {
		d.MemPct = v
	}
	// stash unknown keys
	known := map[string]bool{
		"device_id": true, "status": true, "device_token": true, "board": true,
		"hostname": true, "wan_ip": true, "template_id": true, "overlay": true,
		"enrolled_at": true, "approved_at": true, "denied_at": true, "revoked_at": true,
		"last_seen": true, "healthy": true, "agent": true, "uptime_sec": true, "mem_pct": true,
		"extra": true,
	}
	for k, v := range payload {
		if known[k] {
			continue
		}
		if d.Extra == nil {
			d.Extra = map[string]any{}
		}
		d.Extra[k] = v
	}
}

// Enroll registers or refreshes pending device. Returns status + optional token if already approved.
func Enroll(payload map[string]any) (status, deviceToken string, isNew bool) {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	did, _ := payload["device_id"].(string)
	if did == "" {
		did = "unknown"
	}
	existing, ok := m[did]
	now := time.Now().Unix()
	if ok {
		st := existing.Status
		if st == StatusApproved {
			mergePayload(&existing, payload)
			existing.LastSeen = now
			existing.DeviceID = did
			m[did] = existing
			_ = saveDevices(m)
			return StatusApproved, existing.DeviceToken, false
		}
		if st == StatusDenied || st == StatusRevoked {
			existing.LastSeen = now
			m[did] = existing
			_ = saveDevices(m)
			return st, "", false
		}
		mergePayload(&existing, payload)
		existing.Status = StatusPending
		existing.LastSeen = now
		existing.DeviceID = did
		m[did] = existing
		_ = saveDevices(m)
		return StatusPending, "", false
	}
	row := Device{DeviceID: did, Status: StatusPending, EnrolledAt: now, LastSeen: now}
	mergePayload(&row, payload)
	m[did] = row
	_ = saveDevices(m)
	return StatusPending, "", true
}

func Approve(deviceID string) (deviceToken string, err error) {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return "", fmt.Errorf("unknown device")
	}
	if d.Status == StatusApproved {
		return d.DeviceToken, nil
	}
	tok := randomToken(32)
	d.Status = StatusApproved
	d.DeviceToken = tok
	d.ApprovedAt = time.Now().Unix()
	d.DeviceID = deviceID
	m[deviceID] = d
	_ = saveDevices(m)
	return tok, nil
}

func RotateToken(deviceID string) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return "", fmt.Errorf("unknown device")
	}
	if d.Status != StatusApproved {
		return "", fmt.Errorf("not approved")
	}
	tok := randomToken(32)
	d.DeviceToken = tok
	m[deviceID] = d
	_ = saveDevices(m)
	return tok, nil
}

func Deny(deviceID string) error {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return fmt.Errorf("unknown device")
	}
	d.Status = StatusDenied
	d.DeviceToken = ""
	d.DeniedAt = time.Now().Unix()
	m[deviceID] = d
	return saveDevices(m)
}

func Revoke(deviceID string) error {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return fmt.Errorf("unknown device")
	}
	d.Status = StatusRevoked
	d.DeviceToken = ""
	d.RevokedAt = time.Now().Unix()
	m[deviceID] = d
	return saveDevices(m)
}

func StatusOf(deviceID string) string {
	mu.Lock()
	defer mu.Unlock()
	d, ok := loadDevices()[deviceID]
	if !ok {
		return ""
	}
	return d.Status
}

func Heartbeat(payload map[string]any) {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	did, _ := payload["device_id"].(string)
	if did == "" {
		did = "unknown"
	}
	cur, ok := m[did]
	if !ok {
		return
	}
	if cur.Status != StatusApproved {
		cur.LastSeen = time.Now().Unix()
		m[did] = cur
		_ = saveDevices(m)
		return
	}
	mergePayload(&cur, payload)
	cur.LastSeen = time.Now().Unix()
	m[did] = cur
	_ = saveDevices(m)
}

func ListDevices() []Device {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	now := time.Now().Unix()
	out := make([]Device, 0, len(m))
	for id, d := range m {
		d.DeviceID = id
		d.DeviceToken = "" // never leak
		d.Healthy = d.LastSeen > 0 && (now-d.LastSeen) < 120
		out = append(out, d)
	}
	return out
}

func ListPending() []Device {
	var out []Device
	for _, d := range ListDevices() {
		if d.Status == StatusPending {
			out = append(out, d)
		}
	}
	return out
}

var allowedEdgeActions = map[string]bool{
	"ping": true, "status": true, "metrics": true, "logread": true,
	"wifi_reload": true, "network_reload": true,
	"uci_get": true, "uci_show": true, "uci_set": true, "uci_commit": true, "uci_batch": true,
	"config_backup": true, "config_restore": true, "apply_template": true, "bootstrap_apply": true,
	"reboot": true, "agent_update": true, "mtls_refresh": true, "sysupgrade": true, "apply_rsc": true,
	"dhcp_leases": true, "wifi_clients": true, "dhcp_static": true, "rtsp_probe": true, "nvr_record_start": true, "nvr_record_stop": true, "nvr_record_status": true, "nvr_disk_info": true, "camera_ptz": true,
	"guest_status": true, "guest_grant": true, "guest_revoke": true, "guest_apply_template": true,
}

func EnqueueCmd(deviceID, action, arg string) string {
	mu.Lock()
	defer mu.Unlock()
	_ = ensure()
	if !allowedEdgeActions[action] {
		return ""
	}
	if st := statusUnlocked(deviceID); st != StatusApproved && st != "" {
		return ""
	}
	var wrap struct {
		Pending []map[string]any `json:"pending"`
	}
	if b, err := os.ReadFile(commandsPath()); err == nil {
		_ = json.Unmarshal(b, &wrap)
	}
	id := randomID()
	wrap.Pending = append(wrap.Pending, map[string]any{
		"id": id, "device_id": deviceID, "action": action, "arg": arg, "created": time.Now().Unix(),
	})
	b, _ := json.MarshalIndent(wrap, "", "  ")
	tmp := commandsPath() + ".tmp"
	_ = os.WriteFile(tmp, append(b, '\n'), 0o600)
	_ = os.Rename(tmp, commandsPath())
	return id
}

func statusUnlocked(deviceID string) string {
	d, ok := loadDevices()[deviceID]
	if !ok {
		return ""
	}
	return d.Status
}

func PollCommands(deviceID string) []map[string]any {
	mu.Lock()
	defer mu.Unlock()
	if st := statusUnlocked(deviceID); st != StatusApproved {
		return []map[string]any{}
	}
	var wrap struct {
		Pending []map[string]any `json:"pending"`
	}
	if b, err := os.ReadFile(commandsPath()); err == nil {
		_ = json.Unmarshal(b, &wrap)
	}
	var mine, rest []map[string]any
	for _, c := range wrap.Pending {
		if c["device_id"] == deviceID {
			mine = append(mine, c)
		} else {
			rest = append(rest, c)
		}
	}
	wrap.Pending = rest
	b, _ := json.MarshalIndent(wrap, "", "  ")
	tmp := commandsPath() + ".tmp"
	_ = os.WriteFile(tmp, append(b, '\n'), 0o600)
	_ = os.Rename(tmp, commandsPath())
	if mine == nil {
		mine = []map[string]any{}
	}
	return mine
}


// WaitCmdResult polls results log until cmd_id appears or timeout.
func WaitCmdResult(cmdID string, timeout time.Duration) (map[string]any, error) {
	if cmdID == "" {
		return nil, fmt.Errorf("empty cmd id")
	}
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, m := range ListResults() {
			id, _ := m["cmd_id"].(string)
			if id == cmdID {
				return m, nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("timeout waiting for cmd %s", cmdID)
}

// FindResult returns latest result for cmd_id if present.
func FindResult(cmdID string) (map[string]any, bool) {
	var last map[string]any
	ok := false
	for _, m := range ListResults() {
		id, _ := m["cmd_id"].(string)
		if id == cmdID {
			last = m
			ok = true
		}
	}
	return last, ok
}


func ListResults() []map[string]any {
	mu.Lock()
	defer mu.Unlock()
	b, err := os.ReadFile(resultsPath())
	if err != nil {
		return []map[string]any{}
	}
	var out []map[string]any
	lines := strings.Split(string(b), "\n")
	start := 0
	if len(lines) > 100 {
		start = len(lines) - 100
	}
	for _, line := range lines[start:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func CmdResult(payload map[string]any) {
	mu.Lock()
	defer mu.Unlock()
	_ = ensure()
	b, _ := json.Marshal(payload)
	f, err := os.OpenFile(resultsPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

func deviceDir(id string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, id)
	return filepath.Join(paths.EdgeDir(), safe)
}

func ListBackups(deviceID string) []map[string]any {
	dir := filepath.Join(deviceDir(deviceID), "backups")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []map[string]any{}
	}
	var out []map[string]any
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, map[string]any{
			"name": e.Name(),
			"size": info.Size(),
			"mod":  info.ModTime().Unix(),
		})
	}
	return out
}

func BackupPath(deviceID, name string) (string, error) {
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, ":") {
		return "", fmt.Errorf("invalid name")
	}
	for _, r := range name {
		if r < 32 {
			return "", fmt.Errorf("invalid name")
		}
	}
	path := filepath.Join(deviceDir(deviceID), "backups", name)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func ListMetricsTail(deviceID string, max int) []map[string]any {
	if max <= 0 {
		max = 50
	}
	path := filepath.Join(deviceDir(deviceID), "metrics.jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		return []map[string]any{}
	}
	lines := strings.Split(string(b), "\n")
	start := 0
	if len(lines) > max {
		start = len(lines) - max
	}
	var out []map[string]any
	for _, line := range lines[start:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func SaveBackup(deviceID string, r io.Reader) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	dir := filepath.Join(deviceDir(deviceID), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := time.Now().UTC().Format("20060102-150405") + ".tar.gz"
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// cap backup upload (50 MiB)
	if _, err := io.Copy(f, io.LimitReader(r, 50<<20)); err != nil {
		return "", err
	}
	return path, nil
}

func SaveMetrics(deviceID string, payload map[string]any) error {
	mu.Lock()
	defer mu.Unlock()
	dir := deviceDir(deviceID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "metrics.jsonl")
	b, _ := json.Marshal(payload)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}


func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = io.ReadFull(rand.Reader, b)
	return hex.EncodeToString(b)
}
