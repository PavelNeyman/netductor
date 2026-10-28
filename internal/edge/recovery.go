package edge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/statefile"
)

type recoveryCode struct {
	Code       string `json:"code"`
	SiteID     string `json:"site_id,omitempty"`
	Note       string `json:"note,omitempty"`
	Expires    int64  `json:"expires"`
	CreatedAt  int64  `json:"created_at"`
	UsedAt     int64  `json:"used_at,omitempty"`
	ReservedAt int64  `json:"reserved_at,omitempty"`
	ReserveID  string `json:"reserve_id,omitempty"`
	// Bound: enroll already persisted the device; never auto-release reservation.
	Bound bool `json:"bound,omitempty"`
}

func recoveryPath() string {
	return filepath.Join(paths.EdgeDir(), "recovery_codes.json")
}

func loadRecovery() ([]recoveryCode, error) {
	var list []recoveryCode
	err := statefile.LoadJSON(recoveryPath(), &list)
	if statefile.IsNotExist(err) {
		return []recoveryCode{}, nil
	}
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []recoveryCode{}
	}
	return list, nil
}

func saveRecovery(list []recoveryCode) error {
	return statefile.SaveJSON(recoveryPath(), list)
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
	list, err := loadRecovery()
	if err != nil {
		return "", time.Time{}, err
	}
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

// PeekRecoveryCode validates a recovery code without consuming it.
// Active reservations (ReservedAt or Bound) are never treated as free until
// Finalize, Abort, or the code's own Expires timestamp.
func PeekRecoveryCode(code string) (siteID string, ok bool) {
	code = strings.TrimSpace(code)
	if len(code) < 16 {
		return "", false
	}
	mu.Lock()
	defer mu.Unlock()
	list, err := loadRecovery()
	if err != nil {
		return "", false
	}
	now := time.Now().Unix()
	for i := range list {
		c := &list[i]
		if c.UsedAt != 0 || c.Expires < now {
			continue
		}
		// Any reservation (bound or in-flight) blocks until Commit, Abort, or code Expires.
		// No 120s auto-release — prevents reuse after Enroll succeeded but Bind/Commit failed.
		if c.ReservedAt != 0 || c.Bound {
			continue
		}
		if constEq(c.Code, code) {
			return c.SiteID, true
		}
	}
	return "", false
}

// ReserveRecoveryCode atomically claims a code for enroll (unused → reserved).
// Concurrent second claim fails. No TTL-based steal of an existing reservation.
// Caller must FinalizeRecoveryAfterEnroll or AbortRecoveryCode.
func ReserveRecoveryCode(code string) (siteID, reserveID string, ok bool) {
	code = strings.TrimSpace(code)
	if len(code) < 16 {
		return "", "", false
	}
	mu.Lock()
	defer mu.Unlock()
	list, err := loadRecovery()
	if err != nil {
		return "", "", false
	}
	now := time.Now().Unix()
	var b [8]byte
	_, _ = rand.Read(b[:])
	rid := hex.EncodeToString(b[:])
	for i := range list {
		c := &list[i]
		if c.UsedAt != 0 || c.Expires < now {
			continue
		}
		if c.ReservedAt != 0 || c.Bound {
			if constEq(c.Code, code) {
				return "", "", false
			}
			continue
		}
		if constEq(c.Code, code) {
			c.ReservedAt = now
			c.ReserveID = rid
			c.Bound = false
			if err := saveRecovery(list); err != nil {
				return "", "", false
			}
			return c.SiteID, rid, true
		}
	}
	return "", "", false
}

// FinalizeRecoveryAfterEnroll marks the code permanently used in one save after device persist.
// Combines bind+commit so a failed intermediate Bound cannot leave a freeable reservation.
func FinalizeRecoveryAfterEnroll(code, reserveID string) bool {
	code = strings.TrimSpace(code)
	reserveID = strings.TrimSpace(reserveID)
	if code == "" || reserveID == "" {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	list, err := loadRecovery()
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	for i := range list {
		c := &list[i]
		if !constEq(c.Code, code) {
			continue
		}
		if c.UsedAt != 0 {
			return true
		}
		if c.ReserveID != reserveID {
			return false
		}
		c.UsedAt = now
		c.Bound = true
		c.ReservedAt = 0
		c.ReserveID = ""
		return saveRecovery(list) == nil
	}
	return false
}

// BindRecoveryCode marks reservation as enroll-succeeded: never auto-released.
func BindRecoveryCode(code, reserveID string) bool {
	code = strings.TrimSpace(code)
	reserveID = strings.TrimSpace(reserveID)
	if code == "" || reserveID == "" {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	list, err := loadRecovery()
	if err != nil {
		return false
	}
	for i := range list {
		c := &list[i]
		if !constEq(c.Code, code) {
			continue
		}
		if c.UsedAt != 0 {
			return true
		}
		if c.ReserveID != reserveID {
			return false
		}
		c.Bound = true
		return saveRecovery(list) == nil
	}
	return false
}

// CommitRecoveryCode marks a reserved code as permanently used.
func CommitRecoveryCode(code, reserveID string) bool {
	code = strings.TrimSpace(code)
	reserveID = strings.TrimSpace(reserveID)
	if code == "" || reserveID == "" {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	list, err := loadRecovery()
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	for i := range list {
		c := &list[i]
		if !constEq(c.Code, code) {
			continue
		}
		if c.UsedAt != 0 {
			return true // already committed
		}
		if c.ReserveID != reserveID {
			return false
		}
		c.UsedAt = now
		c.ReservedAt = 0
		c.ReserveID = ""
		c.Bound = false
		return saveRecovery(list) == nil
	}
	return false
}

// AbortRecoveryCode releases an unbound reservation so the code can be retried.
// Bound reservations (enroll already persisted) are not released.
func AbortRecoveryCode(code, reserveID string) {
	code = strings.TrimSpace(code)
	reserveID = strings.TrimSpace(reserveID)
	mu.Lock()
	defer mu.Unlock()
	list, err := loadRecovery()
	if err != nil {
		return
	}
	for i := range list {
		c := &list[i]
		if !constEq(c.Code, code) {
			continue
		}
		if c.UsedAt != 0 || c.Bound {
			return
		}
		if c.ReserveID != "" && c.ReserveID != reserveID {
			return
		}
		c.ReservedAt = 0
		c.ReserveID = ""
		_ = saveRecovery(list)
		return
	}
}

// ConsumeRecoveryCode validates and marks used in one step (tests / simple callers).
func ConsumeRecoveryCode(code string) (siteID string, ok bool) {
	siteID, rid, ok := ReserveRecoveryCode(code)
	if !ok {
		return "", false
	}
	if !CommitRecoveryCode(code, rid) {
		AbortRecoveryCode(code, rid)
		return "", false
	}
	return siteID, true
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
	list, err := loadRecovery()
	if err != nil {
		return false
	}
	for _, c := range list {
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
	m, err := loadDevices()
	if err != nil {
		return fmt.Errorf("devices store: %w", err)
	}
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
	m, err := loadDevices()
	if err != nil {
		return nil, fmt.Errorf("devices store: %w", err)
	}
	return json.MarshalIndent(map[string]any{
		"devices":  m,
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
	m, err := loadDevices()
	if err != nil {
		return 0, fmt.Errorf("devices store: %w", err)
	}
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

// DeviceStatus returns pending/approved/… or empty if unknown.
func DeviceStatus(deviceID string) string {
	mu.Lock()
	defer mu.Unlock()
	m, err := loadDevices()
	if err != nil {
		return ""
	}
	d, ok := m[deviceID]
	if !ok {
		return ""
	}
	return d.Status
}

// DeviceSiteID returns Extra["site_id"] or empty.
func DeviceSiteID(deviceID string) string {
	mu.Lock()
	defer mu.Unlock()
	m, err := loadDevices()
	if err != nil {
		return ""
	}
	d, ok := m[deviceID]
	if !ok || d.Extra == nil {
		return ""
	}
	s, _ := d.Extra["site_id"].(string)
	return strings.TrimSpace(s)
}

// SetSiteID links device to a location/site id in Extra + returns device.
func SetSiteID(deviceID, siteID string) error {
	mu.Lock()
	defer mu.Unlock()
	m, err := loadDevices()
	if err != nil {
		return fmt.Errorf("devices store: %w", err)
	}
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
