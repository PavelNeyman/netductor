package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// keysRoot is the svc-paths data dir (overridable in tests).
var keysRoot = KeysDir

func peersDirPath() string       { return filepath.Join(keysRoot, "peers") }
func peersIndexPath() string     { return filepath.Join(keysRoot, "peers.json") }
func activePeerPathFile() string { return filepath.Join(keysRoot, "active_peer_id") }

// SetKeysRootForTest redirects peer registry paths; empty restores KeysDir.
func SetKeysRootForTest(dir string) {
	if dir == "" {
		keysRoot = KeysDir
		return
	}
	keysRoot = dir
}

// PeerSlot is a reserved address/port assignment (no overlap with build 10.87.12/13).
type PeerSlot struct {
	Index    int    `json:"index"`
	AddrP_SP string `json:"addr_p_sp"`
	AddrS_SP string `json:"addr_s_sp"`
	AddrP_PS string `json:"addr_p_ps"`
	AddrS_PS string `json:"addr_s_ps"`
	PortSP   int    `json:"port_sp"`
	PortPS   int    `json:"port_ps"`
	WSS_SP   int    `json:"wss_sp"`
	WSS_PS   int    `json:"wss_ps"`
	IfaceSP  string `json:"iface_sp"`
	IfacePS  string `json:"iface_ps"`
}

// ServicePeer is one HA secondary on the service plane.
type ServicePeer struct {
	ID        string    `json:"id"`
	PublicIP  string    `json:"public_ip"`
	AgentID   string    `json:"agent_id,omitempty"`
	State     string    `json:"state"` // prepared|ready|active|failed|removed
	Slot      PeerSlot  `json:"slot"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PeersIndex is the on-disk registry on primary.
type PeersIndex struct {
	Version   int           `json:"version"`
	ActiveID  string        `json:"active_id,omitempty"`
	Peers     []ServicePeer `json:"peers"`
	UpdatedAt time.Time     `json:"updated_at"`
}

var peerMu sync.Mutex

// acquirePeerRegistryLock serializes peer-index read/modify/write operations
// across CLI, API, recovery and agent processes. The lock lives beside peers.json.
func acquirePeerRegistryLock() (func(), error) {
	if err := os.MkdirAll(keysRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create peer registry directory: %w", err)
	}
	dirInfo, err := os.Lstat(keysRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect peer registry directory: %w", err)
	}
	dirStat, ok := dirInfo.Sys().(*unix.Stat_t)
	if !ok || !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 ||
		dirStat.Uid != uint32(os.Geteuid()) || dirInfo.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf("peer registry directory must be a real directory owned by euid %d and not writable by group/others", os.Geteuid())
	}
	path := filepath.Join(keysRoot, "peers.lock")
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open peer registry lock: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("inspect peer registry lock: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o077 != 0 {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("peer registry lock must be a private regular file owned by euid %d", os.Geteuid())
	}
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("lock peer registry: %w", err)
	}
	return func() {
		_ = unix.Flock(fd, unix.LOCK_UN)
		_ = unix.Close(fd)
	}, nil
}

// acquireFirewallMutationLock shares the host-wide lock with firewall.ApplyRole.
func acquireFirewallMutationLock() (func(), error) {
	const path = "/run/lock/netductor-firewall.lock"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create firewall lock directory: %w", err)
	}
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil { return nil, fmt.Errorf("open firewall mutation lock: %w", err) }
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("inspect firewall mutation lock: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0o022 != 0 {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("firewall mutation lock must be owned by euid %d and not writable by group/others", os.Geteuid())
	}
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("acquire firewall mutation lock: %w", err)
	}
	return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = unix.Close(fd) }, nil
}

// SlotTable fixed non-overlapping /30 pairs (slot0 = legacy constants).
func SlotTable() []PeerSlot {
	// WSS_SP unique per slot on primary (nd-wss-sp-server[-iface]); WSS_PS unique on secondary.
	return []PeerSlot{
		{0, "10.87.10.1", "10.87.10.2", "10.87.11.1", "10.87.11.2", 51830, 51832, 8444, 8445, "nd-svc-sp", "nd-svc-ps"},
		{1, "10.87.20.1", "10.87.20.2", "10.87.21.1", "10.87.21.2", 51840, 51842, 8454, 8455, "nd-svc-sp1", "nd-svc-ps1"},
		{2, "10.87.22.1", "10.87.22.2", "10.87.23.1", "10.87.23.2", 51850, 51852, 8464, 8465, "nd-svc-sp2", "nd-svc-ps2"},
		{3, "10.87.24.1", "10.87.24.2", "10.87.25.1", "10.87.25.2", 51860, 51862, 8474, 8475, "nd-svc-sp3", "nd-svc-ps3"},
	}
}

func peerMaterialPath(id string) string {
	return filepath.Join(peersDirPath(), id, "material.json")
}

func LoadPeersIndex() (*PeersIndex, error) {
	b, err := os.ReadFile(peersIndexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &PeersIndex{Version: 1}, nil
		}
		return nil, err
	}
	var idx PeersIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, fmt.Errorf("peers.json corrupt: %w", err)
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	return &idx, nil
}

func SavePeersIndex(idx *PeersIndex) error {
	if err := os.MkdirAll(keysRoot, 0o700); err != nil {
		return err
	}
	idx.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := peersIndexPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, peersIndexPath())
}

func (idx *PeersIndex) findAny(id string) *ServicePeer {
	for i := range idx.Peers {
		if idx.Peers[i].ID == id {
			return &idx.Peers[i]
		}
	}
	return nil
}

func (idx *PeersIndex) Find(id string) *ServicePeer {
	for i := range idx.Peers {
		if idx.Peers[i].ID == id && idx.Peers[i].State != "removed" {
			return &idx.Peers[i]
		}
	}
	return nil
}

func (idx *PeersIndex) FindByIP(ip string) *ServicePeer {
	for i := range idx.Peers {
		if idx.Peers[i].PublicIP == ip && idx.Peers[i].State != "removed" {
			return &idx.Peers[i]
		}
	}
	return nil
}

func (idx *PeersIndex) usedSlots() map[int]bool {
	u := map[int]bool{}
	for _, p := range idx.Peers {
		if p.State != "removed" {
			u[p.Slot.Index] = true
		}
	}
	return u
}

func allocateSlot(idx *PeersIndex) (PeerSlot, error) {
	used := idx.usedSlots()
	for _, s := range SlotTable() {
		if !used[s.Index] {
			return s, nil
		}
	}
	return PeerSlot{}, fmt.Errorf("no free service-plane slots (max %d peers)", len(SlotTable()))
}

func applySlotToMaterial(m *Material, s PeerSlot) {
	m.Slot = s.Index
	m.AddrP_SP, m.AddrS_SP = s.AddrP_SP, s.AddrS_SP
	m.AddrP_PS, m.AddrS_PS = s.AddrP_PS, s.AddrS_PS
	m.PortSP, m.PortPS = s.PortSP, s.PortPS
	m.WSS_SP, m.WSS_PS = s.WSS_SP, s.WSS_PS
	m.IfaceSP, m.IfacePS = s.IfaceSP, s.IfacePS
}

// MigrateLegacyPeerIfNeeded imports single material.json as slot0 peer.
func MigrateLegacyPeerIfNeeded() error {
	unlock, err := acquirePeerRegistryLock()
	if err != nil { return err }
	defer unlock()
	peerMu.Lock()
	defer peerMu.Unlock()
	return MigrateLegacyPeerIfNeededUnlocked()
}
