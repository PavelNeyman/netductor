package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PeerAddOpts provisions a new service-plane peer on primary (does not promote).
type PeerAddOpts struct {
	PeerID    string
	PublicIP  string
	PrimaryIP string
}

// PeerAdd allocates a free slot, generates material, applies Primary conf for that peer.
// Does not change preferred_secondary_ip / active peer.
func PeerAdd(o PeerAddOpts) (*Material, *ServicePeer, error) {
	peerMu.Lock()
	defer peerMu.Unlock()
	_ = MigrateLegacyPeerIfNeededUnlocked()

	idx, err := LoadPeersIndex()
	if err != nil {
		return nil, nil, err
	}
	if o.PeerID == "" {
		o.PeerID = "peer-" + strings.ReplaceAll(o.PublicIP, ".", "-")
	}
	if o.PublicIP == "" {
		return nil, nil, fmt.Errorf("peer add: need public IP")
	}
	if existing := idx.Find(o.PeerID); existing != nil {
		m, err := loadPeerMaterial(o.PeerID)
		if err != nil {
			return nil, nil, err
		}
		return m.ForSecondary(), existing, nil
	}
	if byIP := idx.FindByIP(o.PublicIP); byIP != nil {
		m, err := loadPeerMaterial(byIP.ID)
		if err != nil {
			return nil, nil, err
		}
		return m.ForSecondary(), byIP, nil
	}
	slot, err := allocateSlot(idx)
	if err != nil {
		return nil, nil, err
	}
	m, err := GenerateMaterial(o.PrimaryIP, o.PublicIP)
	if err != nil {
		return nil, nil, err
	}
	m.PeerID = o.PeerID
	applySlotToMaterial(m, slot)
	m.Primary = o.PrimaryIP
	m.Secondary = o.PublicIP

	if err := os.MkdirAll(filepath.Join(peersDir, o.PeerID), 0o700); err != nil {
		return nil, nil, err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(peerMaterialPath(o.PeerID), b, 0o600); err != nil {
		return nil, nil, err
	}
	if len(idx.Peers) == 0 {
		_ = os.WriteFile(filepath.Join(KeysDir, "material.json"), b, 0o600)
	}

	if err := BootstrapPrimary(m); err != nil {
		return nil, nil, fmt.Errorf("bootstrap primary peer %s: %w", o.PeerID, err)
	}

	now := time.Now().UTC()
	sp := ServicePeer{
		ID: o.PeerID, PublicIP: o.PublicIP, State: "prepared", Slot: slot,
		CreatedAt: now, UpdatedAt: now,
	}
	idx.Peers = append(idx.Peers, sp)
	if err := SavePeersIndex(idx); err != nil {
		return nil, nil, err
	}
	return m.ForSecondary(), &sp, nil
}

func loadPeerMaterial(id string) (*Material, error) {
	b, err := os.ReadFile(peerMaterialPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("peer %s material missing", id)
		}
		return nil, fmt.Errorf("peer %s material: %w", id, err)
	}
	var m Material
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("peer %s material corrupt: %w", id, err)
	}
	if m.SPPrivP == "" || m.SPPrivS == "" {
		return nil, fmt.Errorf("peer %s material incomplete", id)
	}
	return &m, nil
}

// MigrateLegacyPeerIfNeededUnlocked assumes peerMu held.
func MigrateLegacyPeerIfNeededUnlocked() error {
	idx, err := LoadPeersIndex()
	if err != nil {
		return err
	}
	if len(idx.Peers) > 0 {
		return nil
	}
	m, err := LoadMaterialStrict()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	id := "legacy-slot0"
	if m.Secondary != "" {
		id = "legacy-" + strings.ReplaceAll(m.Secondary, ".", "-")
	}
	slot := SlotTable()[0]
	applySlotToMaterial(m, slot)
	if err := os.MkdirAll(filepath.Join(peersDir, id), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(peerMaterialPath(id), b, 0o600); err != nil {
		return err
	}
	now := time.Now().UTC()
	idx.Peers = append(idx.Peers, ServicePeer{
		ID: id, PublicIP: m.Secondary, State: "active", Slot: slot,
		CreatedAt: now, UpdatedAt: now,
	})
	idx.ActiveID = id
	_ = os.WriteFile(activePeerPath, []byte(id+"\n"), 0o644)
	return SavePeersIndex(idx)
}

// PeerPromote sets active peer and preferred_secondary_ip (clientcfg).
func PeerPromote(peerID string) error {
	peerMu.Lock()
	defer peerMu.Unlock()
	idx, err := LoadPeersIndex()
	if err != nil {
		return err
	}
	p := idx.Find(peerID)
	if p == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}
	for i := range idx.Peers {
		if idx.Peers[i].State == "active" {
			idx.Peers[i].State = "ready"
		}
		if idx.Peers[i].ID == peerID {
			idx.Peers[i].State = "active"
			idx.Peers[i].UpdatedAt = time.Now().UTC()
		}
	}
	idx.ActiveID = peerID
	if err := SavePeersIndex(idx); err != nil {
		return err
	}
	_ = os.WriteFile(activePeerPath, []byte(peerID+"\n"), 0o644)
	if p.PublicIP != "" {
		_ = os.MkdirAll("/etc/netductor/secrets", 0o700)
		_ = os.WriteFile("/etc/netductor/preferred_secondary_ip", []byte(p.PublicIP+"\n"), 0o644)
		_ = os.WriteFile("/etc/netductor/secrets/preferred_secondary_ip", []byte(p.PublicIP+"\n"), 0o600)
	}
	return nil
}

// PeerRollback promotes explicit id or first ready peer.
func PeerRollback(toPeerID string) error {
	if toPeerID != "" {
		return PeerPromote(toPeerID)
	}
	peerMu.Lock()
	idx, err := LoadPeersIndex()
	peerMu.Unlock()
	if err != nil {
		return err
	}
	for _, p := range idx.Peers {
		if p.State == "ready" {
			return PeerPromote(p.ID)
		}
	}
	return fmt.Errorf("no ready peer to rollback to")
}

// PeerRemove marks peer removed and stops its units on primary.
func PeerRemove(peerID string) error {
	peerMu.Lock()
	defer peerMu.Unlock()
	idx, err := LoadPeersIndex()
	if err != nil {
		return err
	}
	if idx.ActiveID == peerID {
		return fmt.Errorf("refuse remove active peer %s — promote another first", peerID)
	}
	p := idx.Find(peerID)
	if p == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}
	ifaceSP, ifacePS := p.Slot.IfaceSP, p.Slot.IfacePS
	_ = exec.Command("systemctl", "stop", "wg-quick@"+ifaceSP).Run()
	_ = exec.Command("systemctl", "stop", "wg-quick@"+ifacePS).Run()
	if ifacePS != "nd-svc-ps" {
		_ = exec.Command("systemctl", "stop", "nd-wss-ps-client-"+ifacePS).Run()
	}
	for i := range idx.Peers {
		if idx.Peers[i].ID == peerID {
			idx.Peers[i].State = "removed"
			idx.Peers[i].UpdatedAt = time.Now().UTC()
		}
	}
	return SavePeersIndex(idx)
}

// PeerList returns registry snapshot.
func PeerList() (*PeersIndex, error) {
	_ = MigrateLegacyPeerIfNeeded()
	return LoadPeersIndex()
}

// PeerMarkReady sets state ready after secondary bootstrap + validation.
func PeerMarkReady(peerID string) error {
	peerMu.Lock()
	defer peerMu.Unlock()
	idx, err := LoadPeersIndex()
	if err != nil {
		return err
	}
	for i := range idx.Peers {
		if idx.Peers[i].ID == peerID {
			idx.Peers[i].State = "ready"
			idx.Peers[i].UpdatedAt = time.Now().UTC()
			return SavePeersIndex(idx)
		}
	}
	return fmt.Errorf("peer %s not found", peerID)
}
