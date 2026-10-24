package svcpaths

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var peerIDRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// PeerAddOpts provisions a new service-plane peer on primary (does not promote).
type PeerAddOpts struct {
	PeerID    string
	PublicIP  string
	PrimaryIP string
}

func validatePeerID(id string) error {
	if id == "" {
		return fmt.Errorf("empty peer id")
	}
	if !peerIDRe.MatchString(id) {
		return fmt.Errorf("invalid peer id %q (use [a-zA-Z0-9_-], max 64)", id)
	}
	return nil
}

func validatePublicIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid public IP %q", ip)
	}
	return nil
}

// PeerAdd allocates a free slot, generates material, applies Primary conf for that peer only.
func PeerAdd(o PeerAddOpts) (*Material, *ServicePeer, error) {
	peerMu.Lock()
	defer peerMu.Unlock()
	if err := MigrateLegacyPeerIfNeededUnlocked(); err != nil {
		return nil, nil, fmt.Errorf("legacy migrate: %w", err)
	}

	idx, err := LoadPeersIndex()
	if err != nil {
		return nil, nil, err
	}
	if o.PeerID == "" {
		o.PeerID = "peer-" + strings.ReplaceAll(o.PublicIP, ".", "-")
	}
	if err := validatePeerID(o.PeerID); err != nil {
		return nil, nil, err
	}
	if err := validatePublicIP(o.PublicIP); err != nil {
		return nil, nil, err
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
	_ = writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "allocating"})

	m, err := GenerateMaterial(o.PrimaryIP, o.PublicIP)
	if err != nil {
		return nil, nil, err
	}
	m.PeerID = o.PeerID
	applySlotToMaterial(m, slot)
	m.Primary = o.PrimaryIP
	m.Secondary = o.PublicIP

	if err := os.MkdirAll(filepath.Join(peersDirPath(), o.PeerID), 0o700); err != nil {
		return nil, nil, err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(peerMaterialPath(o.PeerID), b, 0o600); err != nil {
		return nil, nil, err
	}
	_ = writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "material"})

	writeLegacy := len(idx.Peers) == 0
	if err := bootstrapPrimaryFn(m, BootstrapPrimaryOpts{
		WriteLegacyMaterial: writeLegacy,
		ResetPolicy:         writeLegacy,
		InstallHealthTimer:  writeLegacy,
	}); err != nil {
		_ = writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "bootstrap_failed"})
		return nil, nil, fmt.Errorf("bootstrap primary peer %s: %w (retry: peer recover %s)", o.PeerID, err, o.PeerID)
	}
	_ = writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "bootstrap"})

	now := time.Now().UTC()
	sp := ServicePeer{
		ID: o.PeerID, PublicIP: o.PublicIP, State: "prepared", Slot: slot,
		CreatedAt: now, UpdatedAt: now,
	}
	idx.Peers = append(idx.Peers, sp)
	if err := SavePeersIndex(idx); err != nil {
		_ = writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "index_failed"})
		return nil, nil, fmt.Errorf("save peers index: %w (retry: peer recover %s)", err, o.PeerID)
	}
	clearPeerOp(o.PeerID)
	_ = appendPeerJournal("add", o.PeerID, "prepared", "")
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
		if err := validatePeerID(id); err != nil {
			id = "legacy-slot0"
		}
	}
	slot := SlotTable()[0]
	applySlotToMaterial(m, slot)
	if err := os.MkdirAll(filepath.Join(peersDirPath(), id), 0o700); err != nil {
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
	if err := os.WriteFile(activePeerPathFile(), []byte(id+"\n"), 0o644); err != nil {
		return err
	}
	return SavePeersIndex(idx)
}

// PeerValidateResult is the outcome of real readiness checks (run where secondary is reachable).
type PeerValidateResult struct {
	PeerID    string   `json:"peer_id"`
	OK        bool     `json:"ok"`
	Checks    []string `json:"checks"`
	Errors    []string `json:"errors,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// PeerValidateLocal runs checks on this host (secondary after bootstrap, or primary smoke).
// On secondary: units, peer pings, :443, core URL. Does not set ready — caller uses PeerMarkReady after OK.
func PeerValidateLocal(peerID string) (*PeerValidateResult, error) {
	res := &PeerValidateResult{PeerID: peerID, CheckedAt: time.Now().UTC()}
	add := func(ok bool, msg string) {
		if ok {
			res.Checks = append(res.Checks, "OK "+msg)
		} else {
			res.Errors = append(res.Errors, msg)
		}
	}

	m, err := LoadMaterialStrict()
	if err != nil {
		// try peer material on primary
		if peerID != "" {
			m, err = loadPeerMaterial(peerID)
		}
	}
	if err != nil {
		add(false, "material: "+err.Error())
		res.OK = false
		return res, nil
	}
	_, sSP, _, sPS, _, _, _, wssPS, ifaceSP, ifacePS := m.resolvedAddrs()
	pSP, _, pPS, _, _, _, _, _, _, _ := m.resolvedAddrs()
	_ = wssPS

	role := hostRole()
	if role == "secondary" {
		for _, u := range []string{"nd-wss-sp-client", "nd-wss-ps-server", "wg-quick@" + ifaceSP, "wg-quick@" + ifacePS} {
			st, err := exec.Command("systemctl", "is-active", u).CombinedOutput()
			add(err == nil && strings.TrimSpace(string(st)) == "active", "unit "+u+"="+strings.TrimSpace(string(st)))
		}
		add(pingOK(pSP), "ping SP peer "+pSP)
		add(pingOK(pPS), "ping PS peer "+pPS)
		add(probeTCP("127.0.0.1", "443", 2*time.Second), "local :443 listener")
		if b, err := os.ReadFile(coreURLPath); err != nil {
			add(false, "secondary_core_url missing")
		} else {
			add(strings.Contains(string(b), pSP), "core_url targets SP "+pSP)
		}
	} else {
		// primary: this peer's units
		spUnit := "nd-wss-sp-server"
		if ifaceSP != "nd-svc-sp" {
			spUnit = "nd-wss-sp-server-" + ifaceSP
		}
		psUnit := "nd-wss-ps-client"
		if ifacePS != "nd-svc-ps" {
			psUnit = "nd-wss-ps-client-" + ifacePS
		}
		for _, u := range []string{spUnit, psUnit, "wg-quick@" + ifaceSP, "wg-quick@" + ifacePS} {
			st, err := exec.Command("systemctl", "is-active", u).CombinedOutput()
			add(err == nil && strings.TrimSpace(string(st)) == "active", "unit "+u+"="+strings.TrimSpace(string(st)))
		}
		add(pingOK(sSP), "ping secondary SP "+sSP)
		add(pingOK(sPS), "ping secondary PS "+sPS)
	}
	res.OK = len(res.Errors) == 0
	return res, nil
}

// PeerMarkReady sets ready only after successful PeerValidateLocal when requireValidate is true.
func PeerMarkReady(peerID string, requireValidate bool) error {
	peerMu.Lock()
	defer peerMu.Unlock()
	if err := validatePeerID(peerID); err != nil {
		return err
	}
	idx, err := LoadPeersIndex()
	if err != nil {
		return err
	}
	p := idx.Find(peerID)
	if p == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}
	if requireValidate {
		// unlock for validate (may take time); re-check after
		peerMu.Unlock()
		vr, err := PeerValidateLocal(peerID)
		peerMu.Lock()
		if err != nil {
			return err
		}
		if !vr.OK {
			_ = appendPeerJournal("validate", peerID, "failed", strings.Join(vr.Errors, "; "))
			return fmt.Errorf("peer %s validate failed: %s", peerID, strings.Join(vr.Errors, "; "))
		}
		// reload index
		idx, err = LoadPeersIndex()
		if err != nil {
			return err
		}
	}
	for i := range idx.Peers {
		if idx.Peers[i].ID == peerID {
			idx.Peers[i].State = "ready"
			idx.Peers[i].UpdatedAt = time.Now().UTC()
			if err := SavePeersIndex(idx); err != nil {
				return err
			}
			_ = appendPeerJournal("ready", peerID, "ready", "")
			return nil
		}
	}
	return fmt.Errorf("peer %s not found", peerID)
}

// PeerPromote requires state ready (or already active). Writes registry + active + preferred atomically-ish.
func PeerPromote(peerID string) error {
	peerMu.Lock()
	defer peerMu.Unlock()
	if err := validatePeerID(peerID); err != nil {
		return err
	}
	idx, err := LoadPeersIndex()
	if err != nil {
		return err
	}
	p := idx.Find(peerID)
	if p == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}
	if p.State != "ready" && p.State != "active" {
		return fmt.Errorf("peer %s state %s — promote only from ready (run peer ready/validate first)", peerID, p.State)
	}
	prevActive := idx.ActiveID
	prevPreferred := readPreferredIP()

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
	if err := os.WriteFile(activePeerPathFile(), []byte(peerID+"\n"), 0o644); err != nil {
		idx.ActiveID = prevActive
		_ = SavePeersIndex(idx)
		return fmt.Errorf("write active_peer_id: %w", err)
	}
	if p.PublicIP != "" {
		_ = os.MkdirAll("/etc/netductor/secrets", 0o700)
		if err := os.WriteFile("/etc/netductor/preferred_secondary_ip", []byte(p.PublicIP+"\n"), 0o644); err != nil {
			_ = rollbackPromote(prevActive, prevPreferred)
			return fmt.Errorf("preferred_secondary_ip: %w", err)
		}
		if err := os.WriteFile("/etc/netductor/secrets/preferred_secondary_ip", []byte(p.PublicIP+"\n"), 0o600); err != nil {
			_ = rollbackPromote(prevActive, prevPreferred)
			return fmt.Errorf("secrets/preferred_secondary_ip: %w", err)
		}
	}
	_ = appendPeerJournal("promote", peerID, "active", "prev="+prevActive)
	return nil
}

func readPreferredIP() string {
	b, err := os.ReadFile("/etc/netductor/preferred_secondary_ip")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func rollbackPromote(prevActive, prevPreferred string) error {
	idx, err := LoadPeersIndex()
	if err != nil {
		return err
	}
	for i := range idx.Peers {
		if idx.Peers[i].ID == prevActive {
			idx.Peers[i].State = "active"
		} else if idx.Peers[i].State == "active" {
			idx.Peers[i].State = "ready"
		}
	}
	idx.ActiveID = prevActive
	_ = SavePeersIndex(idx)
	if prevActive != "" {
		_ = os.WriteFile(activePeerPathFile(), []byte(prevActive+"\n"), 0o644)
	}
	if prevPreferred != "" {
		_ = os.WriteFile("/etc/netductor/preferred_secondary_ip", []byte(prevPreferred+"\n"), 0o644)
		_ = os.WriteFile("/etc/netductor/secrets/preferred_secondary_ip", []byte(prevPreferred+"\n"), 0o600)
	}
	return nil
}

// PeerRollback promotes explicit id or first ready peer.
func PeerRollback(toPeerID string) error {
	if toPeerID != "" {
		// allow rollback to ready peer; if prepared, still require ready
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

// PeerRemove disables, stops, removes confs/units for peer, then marks removed (frees slot).
func PeerRemove(peerID string) error {
	peerMu.Lock()
	defer peerMu.Unlock()
	if err := validatePeerID(peerID); err != nil {
		return err
	}
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
	spUnit := "nd-wss-sp-server"
	if ifaceSP != "nd-svc-sp" {
		spUnit = "nd-wss-sp-server-" + ifaceSP
	}
	psUnit := "nd-wss-ps-client"
	if ifacePS != "nd-svc-ps" {
		psUnit = "nd-wss-ps-client-" + ifacePS
	}
	units := []string{"wg-quick@" + ifaceSP, "wg-quick@" + ifacePS, spUnit, psUnit}
	for _, u := range units {
		_ = exec.Command("systemctl", "disable", "--now", u).Run()
	}
	_ = os.Remove("/etc/wireguard/" + ifaceSP + ".conf")
	_ = os.Remove("/etc/wireguard/" + ifacePS + ".conf")
	_ = os.Remove("/etc/systemd/system/" + spUnit + ".service")
	_ = os.Remove("/etc/systemd/system/" + psUnit + ".service")
	_ = exec.Command("systemctl", "daemon-reload").Run()

	for i := range idx.Peers {
		if idx.Peers[i].ID == peerID {
			idx.Peers[i].State = "removed"
			idx.Peers[i].UpdatedAt = time.Now().UTC()
		}
	}
	if err := SavePeersIndex(idx); err != nil {
		return err
	}
	_ = appendPeerJournal("remove", peerID, "removed", "")
	return nil
}

// PeerList returns registry snapshot.
func PeerList() (*PeersIndex, error) {
	if err := MigrateLegacyPeerIfNeeded(); err != nil {
		return nil, err
	}
	return LoadPeersIndex()
}

func appendPeerJournal(op, peerID, state, detail string) error {
	_ = os.MkdirAll(keysRoot, 0o700)
	path := filepath.Join(keysRoot, "peer-ops.log")
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", time.Now().UTC().Format(time.RFC3339), op, peerID, state, detail)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}
