package edge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type recoveryCode struct {
	Code      string `json:"code"`
	SiteID    string `json:"site_id,omitempty"`
	Note      string `json:"note,omitempty"`
	Expires   int64  `json:"expires"`
	CreatedAt int64  `json:"created_at"`
	UsedAt    int64  `json:"used_at,omitempty"`
}

func recoveryPath() string {
	return filepath.Join(paths.EdgeDir(), "recovery_codes.json")
}

func loadRecovery() []recoveryCode {
	b, err := os.ReadFile(recoveryPath())
	if err != nil {
		return nil
	}
	var list []recoveryCode
	_ = json.Unmarshal(b, &list)
	return list
}

func saveRecovery(list []recoveryCode) error {
	_ = os.MkdirAll(filepath.Dir(recoveryPath()), 0o700)
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(recoveryPath(), append(raw, '\n'), 0o600)
}

// IssueRecoveryCode creates a one-time code (default TTL 24h) for LAN recovery page.
func IssueRecoveryCode(siteID, note string, ttl time.Duration) (code string, expires time.Time, err error) {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", time.Time{}, err
	}
	code = hex.EncodeToString(b[:])
	now := time.Now().UTC()
	exp := now.Add(ttl)
	mu.Lock()
	defer mu.Unlock()
	list := loadRecovery()
	// drop expired
	var keep []recoveryCode
	for _, c := range list {
		if c.UsedAt == 0 && c.Expires > now.Unix() {
			keep = append(keep, c)
		}
	}
	keep = append(keep, recoveryCode{
		Code: code, SiteID: siteID, Note: note,
		Expires: exp.Unix(), CreatedAt: now.Unix(),
	})
	if err := saveRecovery(keep); err != nil {
		return "", time.Time{}, err
	}
	return code, exp, nil
}

// ConsumeRecoveryCode validates and marks used. Returns siteID hint.
func ConsumeRecoveryCode(code string) (siteID string, ok bool) {
	code = strings.TrimSpace(code)
	if len(code) < 16 {
		return "", false
	}
	mu.Lock()
	defer mu.Unlock()
	list := loadRecovery()
	now := time.Now().Unix()
	for i := range list {
		c := &list[i]
		if c.UsedAt != 0 || c.Expires < now {
			continue
		}
		if constEq(c.Code, code) {
			c.UsedAt = now
			siteID = c.SiteID
			_ = saveRecovery(list)
			return siteID, true
		}
	}
	return "", false
}

// ValidRecoveryOrBootstrap — enroll auth: recovery code OR long-lived bootstrap.
func ValidRecoveryOrBootstrap(auth string) bool {
	raw := bearerRaw(auth)
	if raw == "" {
		return false
	}
	if ValidBootstrap(auth) {
		return true
	}
	// recovery codes checked without consume here — consume on enroll success path
	mu.Lock()
	defer mu.Unlock()
	now := time.Now().Unix()
	for _, c := range loadRecovery() {
		if c.UsedAt == 0 && c.Expires >= now && constEq(c.Code, raw) {
			return true
		}
	}
	return false
}

// RegisterPending creates/updates a pending device row (operator pre-declare).
func RegisterPending(deviceID, siteID, note string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return fmt.Errorf("device_id required")
	}
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	now := time.Now().Unix()
	d, ok := m[deviceID]
	if !ok {
		d = Device{DeviceID: deviceID, Status: StatusPending, EnrolledAt: now}
	}
	if d.Status == StatusApproved {
		return fmt.Errorf("already approved — revoke first if re-bind needed")
	}
	d.Status = StatusPending
	d.LastSeen = now
	if siteID != "" {
		if d.Extra == nil {
			d.Extra = map[string]any{}
		}
		d.Extra["site_id"] = siteID
	}
	if note != "" {
		if d.Extra == nil {
			d.Extra = map[string]any{}
		}
		d.Extra["note"] = note
	}
	m[deviceID] = d
	return saveDevices(m)
}

// ExportDevices snapshot for backup/migrate.
func ExportDevices() ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	return json.MarshalIndent(map[string]any{
		"devices": m,
		"exported": time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
}

// ImportDevices merges devices (does not delete existing unless replace).
func ImportDevices(raw []byte, replace bool) (n int, err error) {
	var wrap struct {
		Devices map[string]Device `json:"devices"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return 0, err
	}
	if wrap.Devices == nil {
		return 0, fmt.Errorf("no devices in import")
	}
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	if replace {
		m = map[string]Device{}
	}
	for id, d := range wrap.Devices {
		d.DeviceID = id
		m[id] = d
		n++
	}
	return n, saveDevices(m)
}

// SetSiteID links device to a location/site id in Extra + returns device.
func SetSiteID(deviceID, siteID string) error {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return fmt.Errorf("unknown device")
	}
	if d.Extra == nil {
		d.Extra = map[string]any{}
	}
	d.Extra["site_id"] = siteID
	m[deviceID] = d
	return saveDevices(m)
}
