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

// savePeersIndexFn allows tests to inject registry save failures.
var savePeersIndexFn = SavePeersIndex

// writePeerOpFn allows tests to inject journal write failures.
var writePeerOpFn = writePeerOpDefault

// peerOp tracks in-progress peer add for crash recovery.
type peerOp struct {
	Op        string    `json:"op"`
	PeerID    string    `json:"peer_id"`
	PublicIP  string    `json:"public_ip"`
	PrimaryIP string    `json:"primary_ip"`
	SlotIndex int       `json:"slot_index"`
	Phase     string    `json:"phase"` // allocating|material|bootstrap|bootstrap_failed|index_failed
	UpdatedAt time.Time `json:"updated_at"`
}

func peerOpPath(id string) string {
	return filepath.Join(peersDirPath(), id, "op.json")
}

func writePeerOpDefault(op peerOp) error {
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

func writePeerOp(op peerOp) error {
	return writePeerOpFn(op)
}

func clearPeerOp(id string) {
	_ = os.Remove(peerOpPath(id))
}

func loadPeerOp(id string) (*peerOp, error) {
	b, err := os.ReadFile(peerOpPath(id))
	if err != nil {
		return nil, err
	}
	var op peerOp
	if err := json.Unmarshal(b, &op); err != nil {
		return nil, fmt.Errorf("op.json corrupt for %s: %w", id, err)
	}
	return &op, nil
}

func validatePeerOp(op *peerOp, peerID string, m *Material) error {
	if op.Op != "add" {
		return fmt.Errorf("op %q not supported for recover", op.Op)
	}
	if op.PeerID != peerID {
		return fmt.Errorf("op peer_id %q != requested %q", op.PeerID, peerID)
	}
	if err := validatePeerID(peerID); err != nil {
		return err
	}
	switch op.Phase {
	case "allocating", "material", "bootstrap", "bootstrap_failed", "index_failed":
	default:
		return fmt.Errorf("unknown op phase %q", op.Phase)
	}
	if m != nil {
		if m.PeerID != "" && m.PeerID != peerID {
			return fmt.Errorf("material peer_id %q != %q", m.PeerID, peerID)
		}
		if op.PublicIP != "" && m.Secondary != "" && op.PublicIP != m.Secondary {
			return fmt.Errorf("material secondary IP %s != op %s", m.Secondary, op.PublicIP)
		}
		if op.SlotIndex != 0 || m.Slot != 0 {
			if m.Slot != op.SlotIndex && !(m.Slot == 0 && op.SlotIndex == 0) {
				// allow material slot 0 default only if op also 0
				if m.Slot != op.SlotIndex {
					return fmt.Errorf("material slot %d != op slot %d", m.Slot, op.SlotIndex)
				}
			}
		}
	}
	return nil
}

// needsBootstrap returns true if phase means bootstrap not yet successfully completed.
func needsBootstrap(phase string) bool {
	switch phase {
	case "allocating", "material", "bootstrap_failed":
		return true
	case "bootstrap", "index_failed":
		return false // bootstrap already succeeded
	default:
		return true
	}
}

// RecoverIncompletePeerAdd resumes or reports stuck add without regenerating keys if material exists.
func RecoverIncompletePeerAdd(peerID string) (string, error) {
	peerMu.Lock()
	defer peerMu.Unlock()
	op, err := loadPeerOp(peerID)
	if err != nil {
		if os.IsNotExist(err) {
			return "no incomplete op", nil
		}
		return "", err
	}
	m, err := loadPeerMaterial(peerID)
	if err != nil {
		return "", fmt.Errorf("incomplete add %s phase=%s: material missing (%v) — fix manually", peerID, op.Phase, err)
	}
	if err := validatePeerOp(op, peerID, m); err != nil {
		return "", fmt.Errorf("refuse recover: %w", err)
	}
	idx, err := LoadPeersIndex()
	if err != nil {
		return "", err
	}
	if idx.Find(peerID) != nil {
		clearPeerOp(peerID)
		return "already indexed; cleared op", nil
	}
	if needsBootstrap(op.Phase) {
		if err := bootstrapPrimaryFn(m, BootstrapPrimaryOpts{
			WriteLegacyMaterial: false,
			ResetPolicy:         false,
			InstallHealthTimer:  false,
		}); err != nil {
			_ = writePeerOp(peerOp{Op: "add", PeerID: peerID, PublicIP: op.PublicIP, PrimaryIP: op.PrimaryIP, SlotIndex: op.SlotIndex, Phase: "bootstrap_failed"})
			return "", fmt.Errorf("resume bootstrap: %w", err)
		}
		if err := writePeerOp(peerOp{Op: "add", PeerID: peerID, PublicIP: op.PublicIP, PrimaryIP: op.PrimaryIP, SlotIndex: op.SlotIndex, Phase: "bootstrap"}); err != nil {
			return "", fmt.Errorf("bootstrap ok but checkpoint failed: %w (retry recover; bootstrap should be idempotent)", err)
		}
	}
	now := time.Now().UTC()
	idx.Peers = append(idx.Peers, ServicePeer{
		ID: peerID, PublicIP: op.PublicIP, State: "prepared", Slot: mSlot(m),
		CreatedAt: now, UpdatedAt: now,
	})
	if err := savePeersIndexFn(idx); err != nil {
		_ = writePeerOp(peerOp{Op: "add", PeerID: peerID, PublicIP: op.PublicIP, PrimaryIP: op.PrimaryIP, SlotIndex: op.SlotIndex, Phase: "index_failed"})
		return "", fmt.Errorf("save peers index: %w (retry: peer recover %s)", err, peerID)
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
