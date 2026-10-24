package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	peersDir       = KeysDir + "/peers"
	peersIndex     = KeysDir + "/peers.json"
	activePeerPath = KeysDir + "/active_peer_id"
)

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
	return filepath.Join(peersDir, id, "material.json")
}

func LoadPeersIndex() (*PeersIndex, error) {
	b, err := os.ReadFile(peersIndex)
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
	if err := os.MkdirAll(KeysDir, 0o700); err != nil {
		return err
	}
	idx.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := peersIndex + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, peersIndex)
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
	peerMu.Lock()
	defer peerMu.Unlock()
	return MigrateLegacyPeerIfNeededUnlocked()
}
