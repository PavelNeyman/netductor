package svcpaths

import (
	"encoding/json"
	"errors"
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

var generateMaterialFn = GenerateMaterial

// Preferred file paths (overridable in tests; default production).
var (
	preferredIPPath        = "/etc/netductor/preferred_secondary_ip"
	preferredIPSecretsPath = "/etc/netductor/secrets/preferred_secondary_ip"
	writeFileFn            = os.WriteFile // tests may inject failures
)

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

// assertPeerMaterialIdentity ensures on-disk material matches the requested peer, public IP,
// and (when regSlot >= 0) the registry slot index. Empty secondary/iface/addrs fail closed.
func assertPeerMaterialIdentity(m *Material, peerID, publicIP string, regSlot int) error {
	if m == nil {
		return fmt.Errorf("nil material")
	}
	if m.PeerID != peerID {
		return fmt.Errorf("material peer_id %q != requested %q", m.PeerID, peerID)
	}
	if publicIP == "" {
		return fmt.Errorf("requested public IP empty")
	}
	if m.Secondary == "" {
		return fmt.Errorf("material secondary_ip empty")
	}
	if m.Secondary != publicIP {
		return fmt.Errorf("material secondary_ip %s != requested %s", m.Secondary, publicIP)
	}
	s, ok := slotFromTable(m.Slot)
	if !ok {
		return fmt.Errorf("material slot %d not in SlotTable", m.Slot)
	}
	if regSlot >= 0 && m.Slot != regSlot {
		return fmt.Errorf("material slot %d != registry slot %d", m.Slot, regSlot)
	}
	if m.IfaceSP == "" || m.IfacePS == "" {
		return fmt.Errorf("material iface_sp/ps required")
	}
	if m.IfaceSP != s.IfaceSP || m.IfacePS != s.IfacePS {
		return fmt.Errorf("material ifaces %s/%s != slot %s/%s", m.IfaceSP, m.IfacePS, s.IfaceSP, s.IfacePS)
	}
	if m.AddrS_SP == "" || m.AddrP_SP == "" || m.AddrS_PS == "" || m.AddrP_PS == "" {
		return fmt.Errorf("material SP/PS addresses required")
	}
	if m.AddrS_SP != s.AddrS_SP || m.AddrP_SP != s.AddrP_SP || m.AddrS_PS != s.AddrS_PS || m.AddrP_PS != s.AddrP_PS {
		return fmt.Errorf("material addresses do not match slot %d table", m.Slot)
	}
	if m.PortSP == 0 || m.PortPS == 0 || m.WSS_SP == 0 || m.WSS_PS == 0 {
		return fmt.Errorf("material ports/wss required (got port_sp=%d port_ps=%d wss_sp=%d wss_ps=%d)", m.PortSP, m.PortPS, m.WSS_SP, m.WSS_PS)
	}
	if m.PortSP != s.PortSP || m.PortPS != s.PortPS || m.WSS_SP != s.WSS_SP || m.WSS_PS != s.WSS_PS {
		return fmt.Errorf("material ports/wss do not match slot %d table", m.Slot)
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
	if removed := idx.findAny(o.PeerID); removed != nil && removed.State == "removed" {
		return nil, nil, fmt.Errorf("peer %s is removed (units and configs were torn down); use a new id or peer recover — add does not rebuild a removed peer", o.PeerID)
	}
	if existing := idx.Find(o.PeerID); existing != nil {
		m, err := loadPeerMaterial(o.PeerID)
		if err != nil {
			return nil, nil, err
		}
		if err := assertPeerMaterialIdentity(m, o.PeerID, o.PublicIP, existing.Slot.Index); err != nil {
			return nil, nil, fmt.Errorf("existing peer %s: %w", o.PeerID, err)
		}
		return m.ForSecondary(), existing, nil
	}
	if byIP := idx.FindByIP(o.PublicIP); byIP != nil {
		if byIP.ID != o.PeerID {
			return nil, nil, fmt.Errorf("public IP %s already registered as peer %q (requested %q); refuse silent rebind — remove or use matching --id", o.PublicIP, byIP.ID, o.PeerID)
		}
		m, err := loadPeerMaterial(byIP.ID)
		if err != nil {
			return nil, nil, err
		}
		if err := assertPeerMaterialIdentity(m, byIP.ID, o.PublicIP, byIP.Slot.Index); err != nil {
			return nil, nil, fmt.Errorf("existing peer by IP %s: %w", o.PublicIP, err)
		}
		return m.ForSecondary(), byIP, nil
	}
	slot, err := allocateSlot(idx)
	if err != nil {
		return nil, nil, err
	}
	if err := writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "allocating"}); err != nil {
		return nil, nil, fmt.Errorf("peer op checkpoint allocating: %w", err)
	}

	m, err := generateMaterialFn(o.PrimaryIP, o.PublicIP)
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
	if err := writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "material"}); err != nil {
		return nil, nil, fmt.Errorf("peer op checkpoint material: %w", err)
	}

	writeLegacy := len(idx.Peers) == 0
	if err := bootstrapPrimaryFn(m, BootstrapPrimaryOpts{
		WriteLegacyMaterial: writeLegacy,
		ResetPolicy:         writeLegacy,
		InstallHealthTimer:  writeLegacy,
	}); err != nil {
		jerr := writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "bootstrap_failed"})
		if jerr != nil {
			return nil, nil, errors.Join(
				fmt.Errorf("bootstrap primary peer %s: %w (retry: peer recover %s)", o.PeerID, err, o.PeerID),
				fmt.Errorf("checkpoint bootstrap_failed: %w", jerr),
			)
		}
		return nil, nil, fmt.Errorf("bootstrap primary peer %s: %w (retry: peer recover %s)", o.PeerID, err, o.PeerID)
	}
	if err := writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "bootstrap"}); err != nil {
		return nil, nil, fmt.Errorf("bootstrap ok but checkpoint failed: %w (retry: peer recover %s)", err, o.PeerID)
	}

	now := time.Now().UTC()
	sp := ServicePeer{
		ID: o.PeerID, PublicIP: o.PublicIP, State: "prepared", Slot: slot,
		CreatedAt: now, UpdatedAt: now,
	}
	idx.Peers = append(idx.Peers, sp)
	if err := savePeersIndexFn(idx); err != nil {
		jerr := writePeerOp(peerOp{Op: "add", PeerID: o.PeerID, PublicIP: o.PublicIP, PrimaryIP: o.PrimaryIP, SlotIndex: slot.Index, Phase: "index_failed"})
		if jerr != nil {
			return nil, nil, fmt.Errorf("save peers index: %w; also checkpoint index_failed: %v (retry: peer recover %s)", err, jerr, o.PeerID)
		}
		return nil, nil, fmt.Errorf("save peers index: %w (retry: peer recover %s)", err, o.PeerID)
	}
	clearPeerOp(o.PeerID)
	logPeerJournal("add", o.PeerID, "prepared", "")
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
	m.PeerID = id
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
	if err := writeFileFn(activePeerPathFile(), []byte(id+"\n"), 0o644); err != nil {
		return err
	}
	return SavePeersIndex(idx)
}

// PeerValidateResult is the outcome of real readiness checks (run where secondary is reachable).
type PeerValidateResult struct {
	PeerID    string    `json:"peer_id"`
	OK        bool      `json:"ok"`
	Checks    []string  `json:"checks"`
	Errors    []string  `json:"errors,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// PeerValidateLocal runs checks on this host (secondary after bootstrap, or primary smoke).
// On secondary: units, peer pings, :443, core URL. Does not set ready — caller uses PeerMarkReady after OK.
var peerValidateRole = hostRole
var peerValidateForReady = PeerValidateLocal

func PeerValidateLocal(peerID string) (*PeerValidateResult, error) {
	res := &PeerValidateResult{PeerID: peerID, CheckedAt: time.Now().UTC()}
	add := func(ok bool, msg string) {
		if ok {
			res.Checks = append(res.Checks, "OK "+msg)
		} else {
			res.Errors = append(res.Errors, msg)
		}
	}

	// Primary with an id validates that peer's file. Secondary has only local material.json.
	var m *Material
	var err error
	role := peerValidateRole()
	if role == "secondary" {
		m, err = LoadMaterialForHost()
		if err != nil {
			add(false, "material: "+err.Error())
			res.OK = false
			return res, nil
		}
		if peerID != "" && m.PeerID != peerID {
			add(false, fmt.Sprintf("local material peer_id %q != requested %q", m.PeerID, peerID))
			res.OK = false
			return res, nil
		}
	} else if peerID != "" {
		m, err = loadPeerMaterial(peerID)
		if err != nil {
			add(false, "peer material "+peerID+": "+err.Error())
			res.OK = false
			return res, nil
		}
		if m.PeerID != peerID {
			add(false, fmt.Sprintf("material peer_id %q != requested %q", m.PeerID, peerID))
			res.OK = false
			return res, nil
		}
	} else {
		m, err = LoadMaterialForHost()
		if err != nil {
			add(false, "material: "+err.Error())
			res.OK = false
			return res, nil
		}
	}
	pSP, sSP, pPS, sPS, _, _, _, wssPS, ifaceSP, ifacePS, errR := resolveBootstrapParams(m)
	_ = wssPS
	if errR != nil {
		add(false, "material slot params: "+errR.Error())
		res.OK = false
		return res, nil
	}

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
	p := idx.findAny(peerID)
	if p == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}
	if p.State == "removed" || p.State == "removing" {
		return fmt.Errorf("peer %s is %s — refuse ready", peerID, p.State)
	}
	before := p.State
	if requireValidate {
		// unlock for validate (may take time); re-check after
		peerMu.Unlock()
		vr, err := peerValidateForReady(peerID)
		peerMu.Lock()
		if err != nil {
			return err
		}
		if !vr.OK {
			logPeerJournal("validate", peerID, "failed", strings.Join(vr.Errors, "; "))
			return fmt.Errorf("peer %s validate failed: %s", peerID, strings.Join(vr.Errors, "; "))
		}
		idx, err = LoadPeersIndex()
		if err != nil {
			return err
		}
		again := idx.findAny(peerID)
		if again == nil || again.State != before || again.State == "active" || again.State == "removed" || again.State == "removing" {
			return fmt.Errorf("peer %s changed during validate (was %s, now %s) — refuse ready", peerID, before, peerState(again))
		}
	}
	for i := range idx.Peers {
		if idx.Peers[i].ID == peerID {
			idx.Peers[i].State = "ready"
			idx.Peers[i].UpdatedAt = time.Now().UTC()
			if err := SavePeersIndex(idx); err != nil {
				return err
			}
			logPeerJournal("ready", peerID, "ready", "infra-only; traffic not proven")
			return nil
		}
	}
	return fmt.Errorf("peer %s not found", peerID)
}

// promoteSnap is pre-promote disk/registry state for full rollback.
type promoteSnap struct {
	Index            *PeersIndex
	ActiveIDFile     string // content including newline, or "" if missing
	ActiveIDExisted  bool
	ActiveIDMode     os.FileMode
	Preferred        string
	PreferredExisted bool
	PreferredMode    os.FileMode
	PreferredSecrets string
	SecretsExisted   bool
	SecretsMode      os.FileMode
}

func readSnapFile(path string) (content string, existed bool, mode os.FileMode, err error) {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, 0, nil
		}
		return "", false, 0, fmt.Errorf("snapshot %s: %w", path, err)
	}
	if st.IsDir() {
		return "", false, 0, fmt.Errorf("snapshot %s: is a directory", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false, 0, fmt.Errorf("snapshot %s: %w", path, err)
	}
	return string(b), true, st.Mode().Perm(), nil
}

func snapshotPromote() (*promoteSnap, error) {
	idx, err := LoadPeersIndex()
	if err != nil {
		return nil, err
	}
	cp := *idx
	cp.Peers = append([]ServicePeer(nil), idx.Peers...)
	s := &promoteSnap{Index: &cp}

	if c, ok, mode, err := readSnapFile(activePeerPathFile()); err != nil {
		return nil, err
	} else if ok {
		s.ActiveIDFile, s.ActiveIDExisted, s.ActiveIDMode = c, true, mode
	}
	if c, ok, mode, err := readSnapFile(preferredIPPath); err != nil {
		return nil, err
	} else if ok {
		s.Preferred, s.PreferredExisted, s.PreferredMode = c, true, mode
	}
	if c, ok, mode, err := readSnapFile(preferredIPSecretsPath); err != nil {
		return nil, err
	} else if ok {
		s.PreferredSecrets, s.SecretsExisted, s.SecretsMode = c, true, mode
	}
	return s, nil
}

func restoreFile(path, content string, existed bool, mode os.FileMode) error {
	if !existed {
		if err := peerRemoveRemoveFile(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if mode == 0 {
		mode = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeFileFn(path, []byte(content), mode)
}

func rollbackPromoteSnap(snap *promoteSnap) error {
	if snap == nil || snap.Index == nil {
		return fmt.Errorf("nil promote snapshot")
	}
	var errs []error
	if err := SavePeersIndex(snap.Index); err != nil {
		errs = append(errs, fmt.Errorf("restore peers index: %w", err))
	}
	if err := restoreFile(activePeerPathFile(), snap.ActiveIDFile, snap.ActiveIDExisted, snap.ActiveIDMode); err != nil {
		errs = append(errs, fmt.Errorf("restore active_peer_id: %w", err))
	}
	if err := restoreFile(preferredIPPath, snap.Preferred, snap.PreferredExisted, snap.PreferredMode); err != nil {
		errs = append(errs, fmt.Errorf("restore preferred: %w", err))
	}
	sm := snap.SecretsMode
	if sm == 0 {
		sm = 0o600
	}
	if err := restoreFile(preferredIPSecretsPath, snap.PreferredSecrets, snap.SecretsExisted, sm); err != nil {
		errs = append(errs, fmt.Errorf("restore preferred secrets: %w", err))
	}
	return errors.Join(errs...)
}

// PeerPromote requires state ready (or already active). Writes registry + active + preferred; rolls back full snap on failure.
func PeerPromote(peerID string) error {
	peerMu.Lock()
	defer peerMu.Unlock()
	if err := validatePeerID(peerID); err != nil {
		return err
	}
	snap, err := snapshotPromote()
	if err != nil {
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
	if err := writeFileFn(activePeerPathFile(), []byte(peerID+"\n"), 0o644); err != nil {
		rerr := rollbackPromoteSnap(snap)
		if rerr != nil {
			return errors.Join(fmt.Errorf("write active_peer_id: %w", err), fmt.Errorf("rollback: %w", rerr))
		}
		return fmt.Errorf("write active_peer_id: %w", err)
	}
	if p.PublicIP != "" {
		_ = os.MkdirAll(filepath.Dir(preferredIPSecretsPath), 0o700)
		if err := writeFileFn(preferredIPPath, []byte(p.PublicIP+"\n"), 0o644); err != nil {
			rerr := rollbackPromoteSnap(snap)
			if rerr != nil {
				return errors.Join(fmt.Errorf("preferred_secondary_ip: %w", err), fmt.Errorf("rollback: %w", rerr))
			}
			return fmt.Errorf("preferred_secondary_ip: %w", err)
		}
		if err := writeFileFn(preferredIPSecretsPath, []byte(p.PublicIP+"\n"), 0o600); err != nil {
			rerr := rollbackPromoteSnap(snap)
			if rerr != nil {
				return errors.Join(fmt.Errorf("secrets/preferred_secondary_ip: %w", err), fmt.Errorf("rollback: %w", rerr))
			}
			return fmt.Errorf("secrets/preferred_secondary_ip: %w", err)
		}
	}
	logPeerJournal("promote", peerID, "active", "prev="+snap.Index.ActiveID)
	return nil
}

func readPreferredIP() string {
	b, err := os.ReadFile(preferredIPPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
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
	if !safeUnitName(ifaceSP) || !safeUnitName(ifacePS) || !safeUnitName(spUnit) || !safeUnitName(psUnit) {
		return fmt.Errorf("peer %s slot unit name invalid", peerID)
	}
	units := []string{"wg-quick@" + ifaceSP, "wg-quick@" + ifacePS, spUnit, psUnit}
	// Stop first. Do not delete configs if a unit is still running.
	for _, u := range units {
		if err := peerRemoveStop(u); err != nil {
			return fmt.Errorf("peer %s stop failed (configs kept, not marked removed): %w", peerID, err)
		}
	}
	if err := markPeerState(idx, peerID, "removing"); err != nil {
		return fmt.Errorf("peer %s mark removing: %w", peerID, err)
	}
	for _, path := range []string{
		"/etc/wireguard/" + ifaceSP + ".conf",
		"/etc/wireguard/" + ifacePS + ".conf",
		"/etc/systemd/system/" + spUnit + ".service",
		"/etc/systemd/system/" + psUnit + ".service",
	} {
		if err := peerRemoveRemoveFile(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("peer %s remove file %s (not marked removed): %w", peerID, path, err)
		}
	}
	if err := peerRemoveUFW(p.PublicIP, p.Slot.AddrS_SP, p.Slot.AddrS_PS, p.Slot.WSS_SP); err != nil {
		return fmt.Errorf("peer %s ufw cleanup (not marked removed): %w", peerID, err)
	}
	if err := os.RemoveAll(filepath.Join(peersDirPath(), peerID)); err != nil {
		return fmt.Errorf("peer %s material dir (not marked removed): %w", peerID, err)
	}
	if err := peerRemoveReload(); err != nil {
		return fmt.Errorf("peer %s daemon-reload (not marked removed): %w", peerID, err)
	}

	if err := markPeerState(idx, peerID, "removed"); err != nil {
		return fmt.Errorf("peer %s mark removed after cleanup: %w", peerID, err)
	}
	logPeerJournal("remove", peerID, "removed", "")
	return nil
}

// PeerList returns registry snapshot.
func PeerList() (*PeersIndex, error) {
	if err := MigrateLegacyPeerIfNeeded(); err != nil {
		return nil, err
	}
	return LoadPeersIndex()
}

// appendPeerJournal appends a diagnostic audit line to peer-ops.log.
// Contract: NOT used for crash recovery (that is op.json + peers.json + material).
// Plane success is not rolled back solely because the audit line failed.
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

// logPeerJournal records a journal line after a committed plane mutation.
// On error prints to stderr; does not fail the caller.
func logPeerJournal(op, peerID, state, detail string) {
	if err := appendPeerJournal(op, peerID, state, detail); err != nil {
		fmt.Fprintf(os.Stderr, "svc-paths: peer-ops journal %s %s: %v (journal is diagnostic only)\n", op, peerID, err)
	}
}

// peerRemoveStop / peerRemoveReload / peerRemoveSystemctl are overridable in tests.
var peerRemoveStop = stopUnitIdempotent
var peerRemoveReload = reloadSystemd
var peerRemoveUFW = removePeerUFW
var peerRemoveRemoveFile = os.Remove
var peerRemoveSystemctl = func(args ...string) ([]byte, error) {
	return exec.Command("systemctl", args...).CombinedOutput()
}

func stopUnitIdempotent(unit string) error {
	out, err := peerRemoveSystemctl("disable", "--now", unit)
	if err == nil {
		return nil
	}
	// systemctl never started (missing binary, permission on exec). Fail closed.
	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return fmt.Errorf("systemctl not started: %w", err)
	}
	if _, ok := err.(*exec.ExitError); !ok {
		return fmt.Errorf("systemctl not started: %w", err)
	}
	// systemctl ran and reported this unit absent — idempotent.
	if unitMissing(string(out)) {
		return nil
	}
	return fmt.Errorf("disable %s: %v: %s", unit, err, strings.TrimSpace(string(out)))
}

func reloadSystemd() error {
	out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func safeUnitName(s string) bool {
	if s == "" || strings.ContainsAny(s, "/\\") || strings.Contains(s, "..") {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '@' {
			continue
		}
		return false
	}
	return true
}

func unitMissing(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	// Only a unit-specific systemd message. A D-Bus "No such file or directory" is not this.
	// Unit-specific only. Bare "not found" / "not loaded" / "not-found" is not enough.
	hasUnit := strings.Contains(m, "unit ") || strings.Contains(m, ".service")
	if !hasUnit {
		return false
	}
	return strings.Contains(m, "not found") || strings.Contains(m, "not-found") ||
		strings.Contains(m, "not loaded") || strings.Contains(m, "does not exist")
}

var peerUFWExec = func(args ...string) ([]byte, error) {
	if _, err := exec.LookPath("ufw"); err != nil {
		return nil, errUFWMissing
	}
	return exec.Command("ufw", args...).CombinedOutput()
}

var errUFWMissing = fmt.Errorf("ufw not installed")

func removePeerUFW(publicIP, addrSSP, addrSPS string, wssPort int) error {
	out, err := peerUFWExec("status")
	if err != nil && errors.Is(err, errUFWMissing) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ufw status: %v: %s", err, strings.TrimSpace(string(out)))
	}
	rules := [][]string{
		{"delete", "allow", "from", addrSSP, "to", "any", "port", "8789", "proto", "tcp"},
		{"delete", "allow", "from", addrSPS, "to", "any", "port", "8789", "proto", "tcp"},
		{"delete", "allow", "from", publicIP, "to", "any", "port", fmt.Sprintf("%d", wssPort), "proto", "tcp"},
	}
	for _, rule := range rules {
		if rule[3] == "" || rule[7] == "0" {
			continue
		}
		bout, berr := peerUFWExec(rule...)
		msg := strings.ToLower(string(bout) + " " + errString(berr))
		if berr != nil && !strings.Contains(msg, "could not delete") && !strings.Contains(msg, "not found") && !strings.Contains(msg, "skipping") {
			return fmt.Errorf("ufw delete %s: %s", rule[3], strings.TrimSpace(string(bout)+" "+errString(berr)))
		}
	}
	after, err := peerUFWExec("status")
	if err != nil {
		return fmt.Errorf("ufw status after delete: %v: %s", err, strings.TrimSpace(string(after)))
	}
	checks := []struct{ src, port string }{
		{addrSSP, "8789"},
		{addrSPS, "8789"},
		{publicIP, fmt.Sprintf("%d", wssPort)},
	}
	for _, c := range checks {
		if c.src == "" || c.port == "0" {
			continue
		}
		if ufwRulePresent(string(after), c.src, c.port) {
			return fmt.Errorf("ufw still has %s/tcp from %s", c.port, c.src)
		}
	}
	return nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func markPeerState(idx *PeersIndex, peerID, state string) error {
	for i := range idx.Peers {
		if idx.Peers[i].ID == peerID {
			idx.Peers[i].State = state
			idx.Peers[i].UpdatedAt = time.Now().UTC()
		}
	}
	return SavePeersIndex(idx)
}

func peerState(p *ServicePeer) string {
	if p == nil {
		return "missing"
	}
	return p.State
}

func ufwRulePresent(status, src, port string) bool {
	wantPort := port + "/tcp"
	for _, line := range strings.Split(status, "\n") {
		fields := strings.Fields(line)
		hasSrc, hasPort := false, false
		for _, f := range fields {
			if f == src {
				hasSrc = true
			}
			if f == wantPort {
				hasPort = true
			}
		}
		if hasSrc && hasPort {
			return true
		}
	}
	return false
}
