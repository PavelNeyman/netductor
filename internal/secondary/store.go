package secondary

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
	"time"

	"github.com/PavelNeyman/netductor/internal/integrity"
	"github.com/PavelNeyman/netductor/internal/paths"
)

type UserTraffic struct {
	User string `json:"user,omitempty"`
	UUID string `json:"uuid,omitempty"`
	Up   int64  `json:"up_bytes,omitempty"`
	Down int64  `json:"down_bytes,omitempty"`
}

type Device struct {
	ID            string         `json:"id"`
	Token         string         `json:"token"` // agent auth
	Name          string         `json:"name"`
	Region        string         `json:"region,omitempty"` // CF-14 geo label e.g. eu, ru
	PublicIP      string         `json:"public_ip"`
	PBK           string         `json:"pbk"`
	SID           string         `json:"sid"`
	SNI           string         `json:"sni"`
	Version       string         `json:"version"`
	SingBoxOK     bool           `json:"singbox_ok"`
	UplinkOK      bool           `json:"uplink_ok"`
	UplinkRTTMs   float64        `json:"uplink_rtt_ms,omitempty"` // T2: S→P :443 dial RTT
	EgressRxBps   float64        `json:"egress_rx_bps,omitempty"` // T2: default-route iface rate
	EgressTxBps   float64        `json:"egress_tx_bps,omitempty"`
	UserTraffic   []UserTraffic  `json:"user_traffic,omitempty"` // C11 from local clash_api
	ConfigVer     int            `json:"config_ver"`
	CPUPercent    float64        `json:"cpu_percent"`
	MemUsedMB     int64          `json:"mem_used_mb"`
	MemTotalMB    int64          `json:"mem_total_mb"`
	Load1         float64        `json:"load1"`
	MismatchTotal int            `json:"mismatch_total,omitempty"`
	MismatchByIP  map[string]int `json:"mismatch_by_ip,omitempty"`
	RecoveryFP    string         `json:"recovery_fp,omitempty"`
	PendingCmds   []string       `json:"pending_cmds,omitempty"`
	LastCmd       string         `json:"last_cmd,omitempty"`
	LastCmdAt     time.Time      `json:"last_cmd_at,omitempty"`
	LastCmdOK     bool           `json:"last_cmd_ok,omitempty"`
	LastCmdLog    string         `json:"last_cmd_log,omitempty"`
	LastSeen      time.Time      `json:"last_seen"`
	MTLSSerial    string         `json:"mtls_serial,omitempty"`
	MTLSSeenAt    time.Time      `json:"mtls_seen_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type registry struct {
	ConfigVer   int      `json:"config_ver"`
	ExitEnabled bool     `json:"exit_enabled"` // abroad → exit via RU
	Devices     []Device `json:"devices"`
}

var (
	mu                sync.Mutex
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidCommand = errors.New("secondary command is not allowed")
)

func path() string {
	return filepath.Join(paths.StateDir(), "secondary", "devices.json")
}

// acquireRegistryLock serializes registry transactions across Netductor processes.
// The lock file lives beside devices.json and is independent of the registry inode,
// which is atomically replaced by save().
func acquireRegistryLock() (func(), error) {
	dir := filepath.Dir(path())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create secondary registry lock directory: %w", err)
	}
	lockPath := filepath.Join(dir, "devices.lock")
	fd, err := unix.Open(lockPath, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open secondary registry lock: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("inspect secondary registry lock: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o077 != 0 {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("secondary registry lock must be a private regular file owned by the current user")
	}
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("acquire secondary registry lock: %w", err)
	}
	return func() {
		_ = unix.Flock(fd, unix.LOCK_UN)
		_ = unix.Close(fd)
	}, nil
}

const maxSecondaryRegistrySize = 16 << 20

func load() (*registry, error) {
	if err := os.MkdirAll(filepath.Dir(path()), 0o700); err != nil {
		return nil, fmt.Errorf("create secondary registry directory: %w", err)
	}
	p := path()
	fd, err := unix.Open(p, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		if err == unix.ENOENT {
			return &registry{ConfigVer: 1}, nil
		}
		return nil, fmt.Errorf("open secondary registry: %w", err)
	}
	f := os.NewFile(uintptr(fd), p)
	if f == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open secondary registry: invalid file descriptor")
	}
	defer f.Close()

	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return nil, fmt.Errorf("inspect secondary registry: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, fmt.Errorf("secondary registry is not a regular file")
	}
	if st.Uid != uint32(os.Geteuid()) {
		return nil, fmt.Errorf("secondary registry is not owned by the current user")
	}
	if st.Size < 0 || st.Size > maxSecondaryRegistrySize {
		return nil, fmt.Errorf("secondary registry size exceeds %d bytes", maxSecondaryRegistrySize)
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSecondaryRegistrySize+1))
	if err != nil {
		return nil, fmt.Errorf("read secondary registry: %w", err)
	}
	if len(b) > maxSecondaryRegistrySize {
		return nil, fmt.Errorf("secondary registry size exceeds %d bytes", maxSecondaryRegistrySize)
	}
	var r registry
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func save(r *registry) error {
	if r != nil && r.Devices == nil {
		r.Devices = []Device{}
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path()), 0o700); err != nil {
		return fmt.Errorf("create secondary registry directory: %w", err)
	}
	dir := filepath.Dir(path())
	tmpFile, err := os.CreateTemp(dir, ".devices.json.tmp-*")
	if err != nil {
		return fmt.Errorf("create secondary registry temporary file: %w", err)
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)

	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("set secondary registry temporary file permissions: %w", err)
	}
	if _, err := tmpFile.Write(append(raw, '\n')); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write secondary registry temporary file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync secondary registry temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close secondary registry temporary file: %w", err)
	}
	if err := os.Rename(tmp, path()); err != nil {
		return fmt.Errorf("replace secondary registry: %w", err)
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open secondary registry directory for sync: %w", err)
	}
	defer dirFile.Close()
	if err := dirFile.Sync(); err != nil {
		return fmt.Errorf("sync secondary registry directory: %w", err)
	}
	return nil
}

func BumpConfigVer() (int, error) {
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return 0, lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return 0, err
	}
	r.ConfigVer++
	if err := save(r); err != nil {
		return 0, fmt.Errorf("persist secondary config version: %w", err)
	}
	return r.ConfigVer, nil
}

func ConfigVerE() (int, error) {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return 0, fmt.Errorf("load secondary registry config version: %w", err)
	}
	return r.ConfigVer, nil
}

func ConfigVer() int {
	v, _ := ConfigVerE()
	if v < 1 {
		return 1
	}
	return v
}

// IssueToken creates a new agent token. Replaces never-joined (ghost) devices with the
// same name so repeated prepare-pack / deploy secondary does not accumulate offline alerts.
func IssueToken(name string) (id, token string, err error) {
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return "", "", lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return "", "", err
	}
	var tb, ib [16]byte
	if _, err := rand.Read(tb[:]); err != nil {
		return "", "", fmt.Errorf("generate secondary token: %w", err)
	}
	if _, err := rand.Read(ib[:]); err != nil {
		return "", "", fmt.Errorf("generate secondary ID: %w", err)
	}
	token = hex.EncodeToString(tb[:])
	id = "secondary-" + hex.EncodeToString(ib[:8])
	if name == "" {
		name = id
	}
	// Replace only an unjoined preparation for this same logical device name.
	// Other names may represent independently prepared packs and must remain valid.
	var kept []Device
	for _, d := range r.Devices {
		ghost := d.LastSeen.IsZero() && d.Name == name
		if ghost {
			continue
		}
		kept = append(kept, d)
	}
	r.Devices = kept
	r.Devices = append(r.Devices, Device{
		ID: id, Token: token, Name: name, CreatedAt: time.Now().UTC(), SNI: "api.vk.me",
	})
	if r.ConfigVer < 1 {
		r.ConfigVer = 1
	}
	return id, token, save(r)
}

func LookupByToken(token string) (*Device, error) {
	if token == "" || len(token) < 16 {
		return nil, nil
	}
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil, fmt.Errorf("load secondary registry for token lookup: %w", err)
	}
	for i := range r.Devices {
		t := r.Devices[i].Token
		if tokenSHAEq(t, token) {
			d := r.Devices[i]
			return &d, nil
		}
	}
	return nil, nil
}

// FindByToken preserves the legacy fail-closed lookup contract.
// New request handlers should use LookupByToken to distinguish storage errors.
func FindByToken(token string) *Device {
	d, _ := LookupByToken(token)
	return d
}

type HeartbeatIn struct {
	PublicIP      string         `json:"public_ip"`
	PBK           string         `json:"pbk"`
	SID           string         `json:"sid"`
	SNI           string         `json:"sni"`
	Version       string         `json:"version"`
	SingBoxOK     bool           `json:"singbox_ok"`
	UplinkOK      bool           `json:"uplink_ok"`
	UplinkRTTMs   float64        `json:"uplink_rtt_ms,omitempty"`
	EgressRxBps   float64        `json:"egress_rx_bps,omitempty"`
	EgressTxBps   float64        `json:"egress_tx_bps,omitempty"`
	UserTraffic   []UserTraffic  `json:"user_traffic,omitempty"`
	ConfigVer     int            `json:"config_ver"`
	CPUPercent    float64        `json:"cpu_percent"`
	MemUsedMB     int64          `json:"mem_used_mb"`
	MemTotalMB    int64          `json:"mem_total_mb"`
	Load1         float64        `json:"load1"`
	CmdDone       string         `json:"cmd_done,omitempty"`
	CmdOK         bool           `json:"cmd_ok,omitempty"`
	CmdLog        string         `json:"cmd_log,omitempty"`
	MismatchTotal int            `json:"mismatch_total,omitempty"`
	MismatchByIP  map[string]int `json:"mismatch_by_ip,omitempty"`
	RecoveryFP    string         `json:"recovery_fp,omitempty"`
}

func Heartbeat(token string, in HeartbeatIn) (*Device, int, error) {
	return HeartbeatWithTLSIdentity(token, in, "")
}

// HeartbeatWithTLSIdentity persists the TLS identity observed by the server.
// tlsSerial must be derived from r.TLS.PeerCertificates, never from request JSON.
func HeartbeatWithTLSIdentity(token string, in HeartbeatIn, tlsSerial string) (*Device, int, error) {
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return nil, 0, lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return nil, 0, err
	}
	if token == "" || len(token) < 16 {
		return nil, 0, ErrUnauthorized
	}
	for i := range r.Devices {
		t := r.Devices[i].Token
		if t == "" || !tokenSHAEq(t, token) {
			continue
		}
		// A transient public-IP probe failure must not erase the last known endpoint.
		// Non-empty reports still replace it so real endpoint changes remain visible.
		if ip := strings.TrimSpace(in.PublicIP); ip != "" {
			r.Devices[i].PublicIP = ip
		}
		r.Devices[i].PBK = in.PBK
		r.Devices[i].SID = in.SID
		if in.SNI != "" {
			r.Devices[i].SNI = in.SNI
		}
		r.Devices[i].Version = in.Version
		r.Devices[i].SingBoxOK = in.SingBoxOK
		r.Devices[i].UplinkOK = in.UplinkOK
		r.Devices[i].UplinkRTTMs = in.UplinkRTTMs
		r.Devices[i].EgressRxBps = in.EgressRxBps
		r.Devices[i].EgressTxBps = in.EgressTxBps
		r.Devices[i].UserTraffic = in.UserTraffic
		r.Devices[i].ConfigVer = in.ConfigVer
		r.Devices[i].CPUPercent = in.CPUPercent
		r.Devices[i].MemUsedMB = in.MemUsedMB
		r.Devices[i].MemTotalMB = in.MemTotalMB
		r.Devices[i].Load1 = in.Load1
		r.Devices[i].MismatchTotal = in.MismatchTotal
		r.Devices[i].MismatchByIP = in.MismatchByIP
		if in.RecoveryFP != "" {
			r.Devices[i].RecoveryFP = in.RecoveryFP
		}
		if in.CmdDone != "" {
			if in.CmdDone == "ssh_keys" && in.CmdOK {
				if err := integrity.CacheSecondaryKeys(r.Devices[i].ID, in.CmdLog); err != nil {
					return nil, 0, fmt.Errorf("cache secondary SSH keys: %w", err)
				}
			}
			r.Devices[i].LastCmd = in.CmdDone
			r.Devices[i].LastCmdOK = in.CmdOK
			r.Devices[i].LastCmdLog = in.CmdLog
			r.Devices[i].LastCmdAt = time.Now().UTC()
			// Clear sticky offsite-fail mark so doctor/TG stop nagging after late ack.
			if in.CmdOK && strings.HasPrefix(in.CmdDone, "backup_pull") && strings.Contains(in.CmdLog, "key=ok") {
				_ = os.Remove(filepath.Join(paths.StateDir(), "backup_offsite_fail"))
			}
		}
		now := time.Now().UTC()
		r.Devices[i].LastSeen = now
		if tlsSerial != "" {
			r.Devices[i].MTLSSerial = strings.ToLower(tlsSerial)
			r.Devices[i].MTLSSeenAt = now
		}
		// After first/real heartbeat: drop only superseded never-joined tokens with the same name.
		// Other empty-IP records may belong to independently prepared Secondary packs.
		name := r.Devices[i].Name
		curID := r.Devices[i].ID
		var kept []Device
		for j := range r.Devices {
			if r.Devices[j].ID == curID {
				kept = append(kept, r.Devices[j])
				continue
			}
			if r.Devices[j].LastSeen.IsZero() && r.Devices[j].Name == name {
				continue
			}
			kept = append(kept, r.Devices[j])
		}
		r.Devices = kept
		if err := save(r); err != nil {
			return nil, 0, fmt.Errorf("persist secondary heartbeat: %w", err)
		}
		// re-find current after prune
		for j := range r.Devices {
			if r.Devices[j].ID == curID {
				d := r.Devices[j]
				return &d, r.ConfigVer, nil
			}
		}
		d := Device{ID: curID}
		return &d, r.ConfigVer, nil
	}
	return nil, 0, os.ErrNotExist
}

func ListE() ([]Device, error) {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil, fmt.Errorf("load secondary registry: %w", err)
	}
	out := make([]Device, len(r.Devices))
	copy(out, r.Devices)
	return out, nil
}

func List() []Device {
	out, _ := ListE()
	return out
}

func Online(d Device, within time.Duration) bool {
	if d.LastSeen.IsZero() {
		return false
	}
	return time.Since(d.LastSeen) < within
}

func ExitEnabledE() (bool, error) {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return false, fmt.Errorf("load secondary registry exit setting: %w", err)
	}
	return r.ExitEnabled, nil
}

func ExitEnabled() bool {
	on, _ := ExitEnabledE()
	return on
}

func SetExitEnabled(on bool) error {
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return err
	}
	r.ExitEnabled = on
	r.ConfigVer++
	return save(r)
}

// PruneDuplicates keeps the newest online device per PublicIP (or per token family).
func PruneDuplicates() (int, error) {
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return 0, lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return 0, fmt.Errorf("load secondary registry for duplicate pruning: %w", err)
	}
	if r == nil {
		return 0, errors.New("secondary registry is nil")
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
	if err := save(r); err != nil {
		return 0, fmt.Errorf("persist duplicate pruning: %w", err)
	}
	return before - len(out), nil
}

// PruneStale removes devices not seen within maxAge (and empty-IP ghosts).
func PruneStale(maxAge time.Duration) (int, error) {
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return 0, lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return 0, fmt.Errorf("load secondary registry for stale pruning: %w", err)
	}
	if r == nil {
		return 0, errors.New("secondary registry is nil")
	}
	var keep []Device
	removed := 0
	now := time.Now().UTC()
	for _, d := range r.Devices {
		// Never-joined: keep 2h after create (heartbeat race with metrics collect)
		if d.LastSeen.IsZero() {
			if d.CreatedAt.IsZero() || now.Sub(d.CreatedAt) < 2*time.Hour {
				keep = append(keep, d)
				continue
			}
			removed++
			continue
		}
		stale := now.Sub(d.LastSeen) > maxAge
		if d.PublicIP == "" && stale {
			removed++
			continue
		}
		if stale && maxAge > 0 && now.Sub(d.LastSeen) > maxAge*2 {
			removed++
			continue
		}
		keep = append(keep, d)
	}
	if removed > 0 {
		if keep == nil {
			keep = []Device{}
		}
		r.Devices = keep
		if err := save(r); err != nil {
			return 0, fmt.Errorf("persist stale secondary pruning: %w", err)
		}
	}
	return removed, nil
}

// RemoveByPublicIP drops all devices for ip except optional keepID (empty = drop all).
func RemoveByPublicIP(ip, keepID string) (int, error) {
	if ip == "" {
		return 0, nil
	}
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return 0, lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return 0, fmt.Errorf("load secondary registry for IP removal: %w", err)
	}
	if r == nil {
		return 0, errors.New("secondary registry is nil")
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
		if err := save(r); err != nil {
			return 0, fmt.Errorf("persist secondary IP removal: %w", err)
		}
	}
	return n, nil
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
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return lockErr
	}
	defer unlock()
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
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return lockErr
	}
	defer unlock()
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

// allowedSecondaryCmd mirrors runAgentCmd in agent.go — queue only known ops.
func allowedSecondaryCmd(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "reboot", "upgrade", "mtls_refresh", "backup_pull", "metrics", "journal", "ssh_keys", "backup_local", "aptguard_scan":
		return true
	}
	if strings.HasPrefix(cmd, "upgrade:") || strings.HasPrefix(cmd, "backup_pull:") || strings.HasPrefix(cmd, "ssh_allowlist:") {
		return true
	}
	if strings.HasPrefix(cmd, "restart:") {
		unit := strings.TrimPrefix(cmd, "restart:")
		switch unit {
		case "sing-box", "netductor-secondary-agent", "netductor-api", "netductor-telegram-bot", "blocky":
			return true
		}
	}
	return false
}

func EnqueueCmd(id, cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || !allowedSecondaryCmd(cmd) {
		return fmt.Errorf("%w: %q", ErrInvalidCommand, cmd)
	}
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return lockErr
	}
	defer unlock()
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

func TakeCmds(id string) ([]string, error) {
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return nil, lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return nil, err
	}
	for i := range r.Devices {
		if r.Devices[i].ID != id {
			continue
		}
		cmds := append([]string{}, r.Devices[i].PendingCmds...)
		r.Devices[i].PendingCmds = nil
		if err := save(r); err != nil {
			return nil, fmt.Errorf("persist secondary command dequeue: %w", err)
		}
		return cmds, nil
	}
	return nil, os.ErrNotExist
}

func tokenSHAEq(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	x := sha256.Sum256([]byte(a))
	y := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}

// ListPublicIPsE returns unique non-empty public IPs and propagates registry errors.
func ListPublicIPsE() ([]string, error) {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil, fmt.Errorf("load secondary registry public IPs: %w", err)
	}
	if r == nil {
		return nil, errors.New("secondary registry is nil")
	}
	seen := map[string]bool{}
	var out []string
	for _, d := range r.Devices {
		ip := strings.TrimSpace(d.PublicIP)
		if ip == "" || seen[ip] {
			continue
		}
		seen[ip] = true
		out = append(out, ip)
	}
	return out, nil
}

// ListPublicIPs preserves the legacy fail-closed lookup contract.
func ListPublicIPs() []string {
	out, _ := ListPublicIPsE()
	return out
}

// StoreRecoveryPeerFingerprint saves a per-device recovery TLS fingerprint.
// A shared fingerprint is deliberately not written: multiple secondaries must
// never overwrite one another's trust anchor.
func StoreRecoveryPeerFingerprint(deviceID, fp string) error {
	deviceID = strings.TrimSpace(deviceID)
	fp = strings.TrimSpace(fp)
	if deviceID == "" || fp == "" {
		return fmt.Errorf("recovery fingerprint requires device ID and fingerprint")
	}
	fp = strings.TrimPrefix(strings.ToLower(fp), "sha256:")
	raw, err := hex.DecodeString(fp)
	if err != nil || len(raw) != sha256.Size {
		return fmt.Errorf("invalid recovery SHA-256 fingerprint")
	}
	sum := sha256.Sum256([]byte(deviceID))
	name := "recovery_peer_" + hex.EncodeToString(sum[:]) + ".fp"
	dir := filepath.Join(paths.StateDir(), "secondary")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create secondary recovery pin directory: %w", err)
	}
	line := []byte("sha256:" + fp + "\n")
	tmpFile, err := os.CreateTemp(dir, ".recovery-pin.tmp-*")
	if err != nil {
		return fmt.Errorf("create secondary recovery fingerprint temporary file: %w", err)
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)
	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("set secondary recovery fingerprint permissions: %w", err)
	}
	if _, err := tmpFile.Write(line); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write secondary recovery fingerprint: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync secondary recovery fingerprint: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close secondary recovery fingerprint: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("replace secondary recovery fingerprint: %w", err)
	}
	dirFile, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open secondary recovery pin directory for sync: %w", err)
	}
	defer dirFile.Close()
	if err := dirFile.Sync(); err != nil {
		return fmt.Errorf("sync secondary recovery pin directory: %w", err)
	}
	return nil
}

func sanitizeID(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// SetRegion sets CF-14 region label on a secondary device (eu, ru, …).
func SetRegion(id, region string) error {
	id = strings.TrimSpace(id)
	region = strings.TrimSpace(strings.ToLower(region))
	mu.Lock()
	defer mu.Unlock()
	unlock, lockErr := acquireRegistryLock()
	if lockErr != nil {
		return lockErr
	}
	defer unlock()
	r, err := load()
	if err != nil {
		return err
	}
	for i := range r.Devices {
		if r.Devices[i].ID == id {
			r.Devices[i].Region = region
			return save(r)
		}
	}
	return fmt.Errorf("secondary device not found: %s", id)
}
