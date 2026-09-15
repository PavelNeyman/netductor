package secondary

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type Device struct {
	ID         string    `json:"id"`
	Token      string    `json:"token"` // agent auth
	Name       string    `json:"name"`
	PublicIP   string    `json:"public_ip"`
	PBK        string    `json:"pbk"`
	SID        string    `json:"sid"`
	SNI        string    `json:"sni"`
	Version    string    `json:"version"`
	SingBoxOK  bool      `json:"singbox_ok"`
	ConfigVer  int       `json:"config_ver"`
	CPUPercent float64   `json:"cpu_percent"`
	MemUsedMB  int64     `json:"mem_used_mb"`
	MemTotalMB int64     `json:"mem_total_mb"`
	Load1      float64   `json:"load1"`
	MismatchTotal int            `json:"mismatch_total,omitempty"`
	MismatchByIP  map[string]int `json:"mismatch_by_ip,omitempty"`
	PendingCmds []string  `json:"pending_cmds,omitempty"`
	LastCmd     string    `json:"last_cmd,omitempty"`
	LastCmdAt   time.Time `json:"last_cmd_at,omitempty"`
	LastCmdOK   bool      `json:"last_cmd_ok,omitempty"`
	LastCmdLog  string    `json:"last_cmd_log,omitempty"`
	LastSeen   time.Time `json:"last_seen"`
	CreatedAt  time.Time `json:"created_at"`
}

type registry struct {
	ConfigVer   int      `json:"config_ver"`
	ExitEnabled bool     `json:"exit_enabled"` // abroad → exit via RU
	Devices     []Device `json:"devices"`
}

var mu sync.Mutex

func path() string {
	return filepath.Join(paths.StateDir(), "secondary", "devices.json")
}

func legacyPath() string {
	return filepath.Join(paths.StateDir(), "relay", "devices.json")
}

func load() (*registry, error) {
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	b, err := os.ReadFile(path())
	if err != nil && os.IsNotExist(err) {
		// migrate from legacy relay/ path
		if lb, e2 := os.ReadFile(legacyPath()); e2 == nil {
			_ = os.WriteFile(path(), lb, 0o600)
			b, err = lb, nil
		}
	}
	if err != nil {
		if os.IsNotExist(err) {
			return &registry{ConfigVer: 1}, nil
		}
		return nil, err
	}
	var r registry
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func save(r *registry) error {
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := path() + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path())
}

func BumpConfigVer() int {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return 0
	}
	r.ConfigVer++
	_ = save(r)
	return r.ConfigVer
}

func ConfigVer() int {
	mu.Lock()
	defer mu.Unlock()
	r, _ := load()
	if r == nil {
		return 1
	}
	return r.ConfigVer
}

func IssueToken(name string) (id, token string, err error) {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return "", "", err
	}
	var tb, ib [16]byte
	_, _ = rand.Read(tb[:])
	_, _ = rand.Read(ib[:])
	token = hex.EncodeToString(tb[:])
	id = "secondary-" + hex.EncodeToString(ib[:8])
	if name == "" {
		name = id
	}
	r.Devices = append(r.Devices, Device{
		ID: id, Token: token, Name: name, CreatedAt: time.Now().UTC(), SNI: "api.vk.me",
	})
	if r.ConfigVer < 1 {
		r.ConfigVer = 1
	}
	return id, token, save(r)
}

func FindByToken(token string) *Device {
	if token == "" || len(token) < 16 {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil
	}
	for i := range r.Devices {
		t := r.Devices[i].Token
		if t == "" || len(t) != len(token) {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			d := r.Devices[i]
			return &d
		}
	}
	return nil
}

type HeartbeatIn struct {
	PublicIP   string  `json:"public_ip"`
	PBK        string  `json:"pbk"`
	SID        string  `json:"sid"`
	SNI        string  `json:"sni"`
	Version    string  `json:"version"`
	SingBoxOK  bool    `json:"singbox_ok"`
	ConfigVer  int     `json:"config_ver"`
	CPUPercent float64 `json:"cpu_percent"`
	MemUsedMB  int64   `json:"mem_used_mb"`
	MemTotalMB int64   `json:"mem_total_mb"`
	Load1      float64 `json:"load1"`
	CmdDone    string  `json:"cmd_done,omitempty"`
	CmdOK      bool    `json:"cmd_ok,omitempty"`
	CmdLog     string  `json:"cmd_log,omitempty"`
	MismatchTotal int            `json:"mismatch_total,omitempty"`
	MismatchByIP  map[string]int `json:"mismatch_by_ip,omitempty"`
}

func Heartbeat(token string, in HeartbeatIn) (*Device, int, error) {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil, 0, err
	}
	if token == "" || len(token) < 16 {
		return nil, 0, fmt.Errorf("unauthorized")
	}
	for i := range r.Devices {
		t := r.Devices[i].Token
		if t == "" || len(t) != len(token) || subtle.ConstantTimeCompare([]byte(t), []byte(token)) != 1 {
			continue
		}
		r.Devices[i].PublicIP = in.PublicIP
		r.Devices[i].PBK = in.PBK
		r.Devices[i].SID = in.SID
		if in.SNI != "" {
			r.Devices[i].SNI = in.SNI
		}
		r.Devices[i].Version = in.Version
		r.Devices[i].SingBoxOK = in.SingBoxOK
		r.Devices[i].ConfigVer = in.ConfigVer
		r.Devices[i].CPUPercent = in.CPUPercent
		r.Devices[i].MemUsedMB = in.MemUsedMB
		r.Devices[i].MemTotalMB = in.MemTotalMB
		r.Devices[i].Load1 = in.Load1
		r.Devices[i].MismatchTotal = in.MismatchTotal
		r.Devices[i].MismatchByIP = in.MismatchByIP
		if in.CmdDone != "" {
			r.Devices[i].LastCmd = in.CmdDone
			r.Devices[i].LastCmdOK = in.CmdOK
			r.Devices[i].LastCmdLog = in.CmdLog
			r.Devices[i].LastCmdAt = time.Now().UTC()
		}
		r.Devices[i].LastSeen = time.Now().UTC()
		_ = save(r)
		d := r.Devices[i]
		return &d, r.ConfigVer, nil
	}
	return nil, 0, os.ErrNotExist
}

func List() []Device {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil
	}
	out := make([]Device, len(r.Devices))
	copy(out, r.Devices)
	return out
}

func Online(d Device, within time.Duration) bool {
	if d.LastSeen.IsZero() {
		return false
	}
	return time.Since(d.LastSeen) < within
}

func ExitEnabled() bool {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil || r == nil {
		return false
	}
	return r.ExitEnabled
}

func SetExitEnabled(on bool) error {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return err
	}
	r.ExitEnabled = on
	r.ConfigVer++
	return save(r)
}

// PruneDuplicates keeps the newest online device per PublicIP (or per token family).
func PruneDuplicates() int {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil || r == nil {
		return 0
	}
	best := map[string]Device{}
	order := []string{}
	for _, d := range r.Devices {
		key := d.PublicIP
		if key == "" {
			key = d.ID
		}
		prev, ok := best[key]
		if !ok {
			best[key] = d
			order = append(order, key)
			continue
		}
		// prefer online (later LastSeen) and higher config
		if d.LastSeen.After(prev.LastSeen) {
			best[key] = d
		}
	}
	var out []Device
	for _, k := range order {
		if d, ok := best[k]; ok {
			out = append(out, d)
			delete(best, k)
		}
	}
	for _, d := range best {
		out = append(out, d)
	}
	before := len(r.Devices)
	r.Devices = out
	_ = save(r)
	return before - len(out)
}

// PruneStale removes devices not seen within maxAge (and empty-IP ghosts).
func PruneStale(maxAge time.Duration) int {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil || r == nil {
		return 0
	}
	var keep []Device
	removed := 0
	now := time.Now().UTC()
	for _, d := range r.Devices {
		stale := d.LastSeen.IsZero() || now.Sub(d.LastSeen) > maxAge
		ghost := d.PublicIP == "" && stale
		if ghost || (stale && d.PublicIP == "") {
			removed++
			continue
		}
		if stale && maxAge > 0 {
			// keep one offline record only if it has IP (for re-join hint) — still drop if > maxAge*2
			if now.Sub(d.LastSeen) > maxAge*2 || d.LastSeen.IsZero() {
				removed++
				continue
			}
		}
		keep = append(keep, d)
	}
	if removed > 0 {
		r.Devices = keep
		_ = save(r)
	}
	return removed
}

// RemoveByPublicIP drops all devices for ip except optional keepID (empty = drop all).
func RemoveByPublicIP(ip, keepID string) int {
	if ip == "" {
		return 0
	}
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil || r == nil {
		return 0
	}
	var out []Device
	n := 0
	for _, d := range r.Devices {
		if d.PublicIP == ip && d.ID != keepID {
			n++
			continue
		}
		out = append(out, d)
	}
	if n > 0 {
		r.Devices = out
		_ = save(r)
	}
	return n
}

// WaitOnlinePBK waits until a device for publicIP reports PBK (heartbeat after join).
func WaitOnlinePBK(publicIP string, timeout time.Duration) *Device {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, d := range List() {
			if d.PublicIP != publicIP {
				continue
			}
			if d.PBK != "" && Online(d, 2*time.Minute) {
				return &d
			}
		}
		time.Sleep(3 * time.Second)
	}
	return nil
}

// RemoveDevice deletes by id.
func RemoveDevice(id string) error {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return err
	}
	var out []Device
	found := false
	for _, d := range r.Devices {
		if d.ID == id {
			found = true
			continue
		}
		out = append(out, d)
	}
	if !found {
		return fmt.Errorf("not found")
	}
	r.Devices = out
	return save(r)
}

func Rename(id, name string) error {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return err
	}
	for i := range r.Devices {
		if r.Devices[i].ID == id {
			r.Devices[i].Name = name
			return save(r)
		}
	}
	return os.ErrNotExist
}

func EnqueueCmd(id, cmd string) error {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return err
	}
	for i := range r.Devices {
		if r.Devices[i].ID != id {
			continue
		}
		for _, c := range r.Devices[i].PendingCmds {
			if c == cmd {
				return save(r)
			}
		}
		r.Devices[i].PendingCmds = append(r.Devices[i].PendingCmds, cmd)
		return save(r)
	}
	return os.ErrNotExist
}

func TakeCmds(id string) []string {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil
	}
	for i := range r.Devices {
		if r.Devices[i].ID != id {
			continue
		}
		cmds := append([]string{}, r.Devices[i].PendingCmds...)
		r.Devices[i].PendingCmds = nil
		_ = save(r)
		return cmds
	}
	return nil
}
