package svcpaths

import (
	"encoding/json"
	"errors"
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
	return atomicWritePeerFile(peerOpPath(op.PeerID), b, 0o600)
}

func writePeerOp(op peerOp) error {
	return writePeerOpFn(op)
}

func clearPeerOp(id string) error {
	path := peerOpPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove peer op journal: %w", err)
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("open peer op directory for sync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync peer op directory after removal: %w", err)
	}
	return nil
}

func loadPeerOp(id string) (*peerOp, error) {
	b, err := readPeerFile(peerOpPath(id), maxPeerJSONBytes)
	if err != nil {
		return nil, err
	}
	var op peerOp
	if err := json.Unmarshal(b, &op); err != nil {
		return nil, fmt.Errorf("op.json corrupt for %s: %w", id, err)
	}
	return &op, nil
}

func slotFromTable(index int) (PeerSlot, bool) {
	for _, s := range SlotTable() {
		if s.Index == index {
			return s, true
		}
	}
	return PeerSlot{}, false
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
	if _, ok := slotFromTable(op.SlotIndex); !ok {
		return fmt.Errorf("op slot_index %d not in SlotTable", op.SlotIndex)
	}
	if op.PublicIP == "" {
		return fmt.Errorf("op public_ip required")
	}
	if err := validatePublicIP(op.PublicIP); err != nil {
		return err
	}
	if m != nil {
		if m.PeerID != peerID {
			return fmt.Errorf("material peer_id %q != %q", m.PeerID, peerID)
		}
		if m.Secondary == "" {
			return fmt.Errorf("material secondary_ip required")
		}
		if m.Secondary != op.PublicIP {
			return fmt.Errorf("material secondary IP %s != op %s", m.Secondary, op.PublicIP)
		}
		if op.PrimaryIP != "" && m.Primary != "" && m.Primary != op.PrimaryIP {
			return fmt.Errorf("material primary IP %s != op %s", m.Primary, op.PrimaryIP)
		}
		if m.Slot != op.SlotIndex {
			return fmt.Errorf("material slot %d != op slot %d", m.Slot, op.SlotIndex)
		}
		if _, ok := slotFromTable(m.Slot); !ok {
			return fmt.Errorf("material slot %d not in SlotTable", m.Slot)
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
	unlock, lockErr := acquirePeerRegistryLock()
	if lockErr != nil { return "", lockErr }
	defer unlock()
	peerMu.Lock()
	defer peerMu.Unlock()
	op, err := loadPeerOp(peerID)
	if err != nil {
		if os.IsNotExist(err) {
			return "no incomplete op", nil
		}
		return "", err
	}
	// Validate the durable operation identity before considering material recovery.
	if err := validatePeerOp(op, peerID, nil); err != nil {
		return "", fmt.Errorf("refuse recover: %w", err)
	}
	idx, err := LoadPeersIndex()
	if err != nil {
		return "", err
	}
	// A stale add journal must never resurrect an identity that has since been
	// removed, or append a duplicate identity/IP/slot into a changed registry.
	for i := range idx.Peers {
		existing := &idx.Peers[i]
		if existing.ID == peerID && existing.State == "removed" {
			return "", fmt.Errorf("peer %s is already removed; refusing stale add recovery or identity resurrection", peerID)
		}
		if existing.State == "removed" {
			continue
		}
		if existing.ID != peerID && existing.PublicIP == op.PublicIP {
			return "", fmt.Errorf("public IP %s is now assigned to peer %s; refusing stale add recovery", op.PublicIP, existing.ID)
		}
		if existing.ID != peerID && existing.Slot.Index == op.SlotIndex {
			return "", fmt.Errorf("slot %d is now assigned to peer %s; refusing stale add recovery", op.SlotIndex, existing.ID)
		}
	}
	m, err := loadPeerMaterial(peerID)
	if err != nil {
		materialPath := peerMaterialPath(peerID)
		_, statErr := os.Lstat(materialPath)
		if statErr == nil {
			// Existing but unreadable/corrupt/incomplete material may contain keys
			// already relied upon by an interrupted operation. Never replace it.
			return "", fmt.Errorf("incomplete add %s phase=%s: existing material is not valid (%v); refusing key regeneration", peerID, op.Phase, err)
		}
		if !os.IsNotExist(statErr) {
			return "", fmt.Errorf("inspect incomplete add material %s: %w", peerID, statErr)
		}
		if op.Phase != "allocating" {
			return "", fmt.Errorf("incomplete add %s phase=%s: material missing; refusing key regeneration outside allocating phase", peerID, op.Phase)
		}
		if existing := idx.findAny(peerID); existing != nil {
			return "", fmt.Errorf("peer %s already has registry state %s; refusing to regenerate missing material", peerID, existing.State)
		}
		if byIP := idx.FindByIP(op.PublicIP); byIP != nil {
			return "", fmt.Errorf("public IP %s is already assigned to peer %s; refusing recovery", op.PublicIP, byIP.ID)
		}
		for _, existing := range idx.Peers {
			if existing.State != "removed" && existing.Slot.Index == op.SlotIndex {
				return "", fmt.Errorf("slot %d is already assigned to peer %s; refusing recovery", op.SlotIndex, existing.ID)
			}
		}
		generated, genErr := generateMaterialFn(op.PrimaryIP, op.PublicIP)
		if genErr != nil {
			return "", fmt.Errorf("regenerate material for interrupted allocating peer %s: %w", peerID, genErr)
		}
		if generated == nil {
			return "", fmt.Errorf("regenerate material for interrupted allocating peer %s: generator returned nil", peerID)
		}
		generated.PeerID = peerID
		generated.Primary = op.PrimaryIP
		generated.Secondary = op.PublicIP
		slot, ok := slotFromTable(op.SlotIndex)
		if !ok {
			return "", fmt.Errorf("operation slot %d not in SlotTable", op.SlotIndex)
		}
		applySlotToMaterial(generated, slot)
		if generated.SPPrivP == "" || generated.SPPrivS == "" {
			return "", fmt.Errorf("regenerated material for peer %s is incomplete; refusing to persist", peerID)
		}
		if err := assertPeerMaterialIdentity(generated, peerID, op.PublicIP, op.SlotIndex); err != nil {
			return "", fmt.Errorf("regenerated material identity invalid: %w", err)
		}
		b, marshalErr := json.MarshalIndent(generated, "", "  ")
		if marshalErr != nil {
			return "", marshalErr
		}
		if err := atomicWritePeerFile(materialPath, b, 0o600); err != nil {
			return "", fmt.Errorf("persist regenerated peer material: %w", err)
		}
		if err := writePeerOp(peerOp{Op: "add", PeerID: peerID, PublicIP: op.PublicIP, PrimaryIP: op.PrimaryIP, SlotIndex: op.SlotIndex, Phase: "material"}); err != nil {
			return "", fmt.Errorf("material persisted but operation checkpoint failed: %w (retry peer recover %s)", err, peerID)
		}
		m = generated
	}
	if err := validatePeerOp(op, peerID, m); err != nil {
		return "", fmt.Errorf("refuse recover: %w", err)
	}
	if existing := idx.Find(peerID); existing != nil {
		// The registry entry alone does not prove that bootstrap completed.
		// A crash may leave the index and op journal out of sync; retain the
		// journal and fail closed unless the durable phase confirms bootstrap.
		if existing.PublicIP != op.PublicIP || existing.Slot.Index != op.SlotIndex {
			return "", fmt.Errorf("peer %s is indexed with a different IP or slot; retain op journal for manual reconciliation", peerID)
		}
		if op.Phase != "bootstrap" && op.Phase != "index_failed" {
			return "", fmt.Errorf("peer %s is indexed but bootstrap completion is not durably confirmed (phase=%s); retain op journal and reconcile service-plane state before retry", peerID, op.Phase)
		}
		if err := clearPeerOp(peerID); err != nil {
			return "", fmt.Errorf("peer already indexed and bootstrap-confirmed but op journal cleanup is not durable: %w", err)
		}
		return "already indexed; cleared op", nil
	}
	if needsBootstrap(op.Phase) {
		if err := bootstrapPrimaryFn(m, BootstrapPrimaryOpts{
			WriteLegacyMaterial: false,
			ResetPolicy:         false,
			InstallHealthTimer:  false,
		}); err != nil {
			jerr := writePeerOp(peerOp{Op: "add", PeerID: peerID, PublicIP: op.PublicIP, PrimaryIP: op.PrimaryIP, SlotIndex: op.SlotIndex, Phase: "bootstrap_failed"})
			if jerr != nil {
				return "", errors.Join(fmt.Errorf("resume bootstrap: %w", err), fmt.Errorf("checkpoint bootstrap_failed: %w", jerr))
			}
			return "", fmt.Errorf("resume bootstrap: %w", err)
		}
		if err := writePeerOp(peerOp{Op: "add", PeerID: peerID, PublicIP: op.PublicIP, PrimaryIP: op.PrimaryIP, SlotIndex: op.SlotIndex, Phase: "bootstrap"}); err != nil {
			return "", fmt.Errorf("bootstrap ok but checkpoint failed: %w (retry recover; bootstrap should be idempotent)", err)
		}
	}
	now := time.Now().UTC()
	slot, ok := slotFromTable(m.Slot)
	if !ok {
		return "", fmt.Errorf("material slot %d not in SlotTable", m.Slot)
	}
	idx.Peers = append(idx.Peers, ServicePeer{
		ID: peerID, PublicIP: op.PublicIP, State: "prepared", Slot: slot,
		CreatedAt: now, UpdatedAt: now,
	})
	if err := savePeersIndexFn(idx); err != nil {
		jerr := writePeerOp(peerOp{Op: "add", PeerID: peerID, PublicIP: op.PublicIP, PrimaryIP: op.PrimaryIP, SlotIndex: op.SlotIndex, Phase: "index_failed"})
		if jerr != nil {
			return "", errors.Join(
				fmt.Errorf("save peers index: %w (retry: peer recover %s)", err, peerID),
				fmt.Errorf("checkpoint index_failed also failed: %w — safe resume without re-bootstrap NOT guaranteed", jerr),
			)
		}
		return "", fmt.Errorf("save peers index: %w (retry: peer recover %s)", err, peerID)
	}
	if err := clearPeerOp(peerID); err != nil {
		return "", fmt.Errorf("peer recovered and indexed, but op journal cleanup is not durable: %w (retry peer recover %s)", err, peerID)
	}
	logPeerJournal("recover", peerID, "prepared", op.Phase)
	return "recovered to prepared", nil
}

// mSlot resolves slot or returns slot 0 only when index is 0 and present in table.
// Prefer slotFromTable for validation paths.
func mSlot(m *Material) PeerSlot {
	if s, ok := slotFromTable(m.Slot); ok {
		return s
	}
	return PeerSlot{}
}
