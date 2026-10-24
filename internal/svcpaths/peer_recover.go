package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// bootstrapPrimaryFn is BootstrapPrimaryWith; tests may stub to avoid systemctl/wg.
var bootstrapPrimaryFn = BootstrapPrimaryWith

// peerOp tracks in-progress peer add for crash recovery.
type peerOp struct {
	Op        string    `json:"op"`
	PeerID    string    `json:"peer_id"`
	PublicIP  string    `json:"public_ip"`
	PrimaryIP string    `json:"primary_ip"`
	SlotIndex int       `json:"slot_index"`
	Phase     string    `json:"phase"` // allocating|material|bootstrap|indexed
	UpdatedAt time.Time `json:"updated_at"`
}

func peerOpPath(id string) string {
	return filepath.Join(peersDirPath(), id, "op.json")
}

func writePeerOp(op peerOp) error {
	op.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(filepath.Join(peersDirPath(), op.PeerID), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(peerOpPath(op.PeerID), b, 0o600)
}

func clearPeerOp(id string) {
	_ = os.Remove(peerOpPath(id))
}

// RecoverIncompletePeerAdd resumes or reports stuck add without regenerating keys if material exists.
func RecoverIncompletePeerAdd(peerID string) (string, error) {
	peerMu.Lock()
	defer peerMu.Unlock()
	b, err := os.ReadFile(peerOpPath(peerID))
	if err != nil {
		if os.IsNotExist(err) {
			return "no incomplete op", nil
		}
		return "", err
	}
	var op peerOp
	if err := json.Unmarshal(b, &op); err != nil {
		return "", fmt.Errorf("op.json corrupt for %s: %w", peerID, err)
	}
	m, err := loadPeerMaterial(peerID)
	if err != nil {
		return "", fmt.Errorf("incomplete add %s phase=%s: material missing (%v) — fix manually", peerID, op.Phase, err)
	}
	idx, err := LoadPeersIndex()
	if err != nil {
		return "", err
	}
	if idx.Find(peerID) != nil {
		clearPeerOp(peerID)
		return "already indexed; cleared op", nil
	}
	if err := bootstrapPrimaryFn(m, BootstrapPrimaryOpts{
		WriteLegacyMaterial: false,
		ResetPolicy:         false,
		InstallHealthTimer:  false,
	}); err != nil {
		return "", fmt.Errorf("resume bootstrap: %w", err)
	}
	now := time.Now().UTC()
	idx.Peers = append(idx.Peers, ServicePeer{
		ID: peerID, PublicIP: op.PublicIP, State: "prepared", Slot: mSlot(m),
		CreatedAt: now, UpdatedAt: now,
	})
	if err := SavePeersIndex(idx); err != nil {
		return "", err
	}
	clearPeerOp(peerID)
	_ = appendPeerJournal("recover", peerID, "prepared", op.Phase)
	return "recovered to prepared", nil
}

func mSlot(m *Material) PeerSlot {
	for _, s := range SlotTable() {
		if s.Index == m.Slot {
			return s
		}
	}
	return SlotTable()[0]
}
