package fleet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Control-plane model (netductor fleet):
//
//   Primary (abroad): source of truth — users, policies, TG bot, admin API,
//   edge enroll, backups, optional Lampac. Secrets stay here.
//
//   Secondary (RU): VPN data-plane entry only (VLESS/Reality + thin agent).
//   Not a service mirror. No Lampac/bot/DNS fleet sync to secondary.
//
// VPN traffic is NOT load-balanced. Clients prefer secondary entry under WL.

const (
	LabelControlPlane = "control_plane" // primary | secondary
	LabelRegion       = "region"        // abroad | ru
	LabelLampac       = "svc.lampac"    // prefer | active | standby | off
	LabelBot          = "svc.bot"       // active | standby | off
)

func fleetFile() string {
	return filepath.Join(paths.StateDir(), "fleet", "policy.json")
}

// Policy is local operator preference (also mirrored into node labels when possible).
type Policy struct {
	PrimaryNodeID   string `json:"primary_node_id"`
	PrimarySSH      string `json:"primary_ssh,omitempty"` // root@ip for checks from secondary
	SecondaryNodeID string `json:"secondary_node_id,omitempty"`
	SecondarySSH    string `json:"secondary_ssh,omitempty"`
	LampacNodeID    string `json:"lampac_node_id,omitempty"` // preferred runtime host
	BotNodeID       string `json:"bot_node_id,omitempty"`    // preferred TG bot host
	SyncEnabled     bool              `json:"sync_enabled"`
	Notes           string            `json:"notes,omitempty"`
}

func defaultPolicy() Policy {
	return Policy{
		SyncEnabled: false,
		Notes:       "primary=abroad control-plane; secondary=RU VPN entry only",
	}
}

// LoadPolicy reads fleet policy or defaults.
func LoadPolicy() Policy {
	b, err := os.ReadFile(fleetFile())
	if err != nil {
		return defaultPolicy()
	}
	var p Policy
	if jsonUnmarshal(b, &p) != nil {
		return defaultPolicy()
	}
	if !p.SyncEnabled && p.PrimaryNodeID == "" {
		p.SyncEnabled = true
	}
	return p
}

func SavePolicy(p Policy) error {
	_ = os.MkdirAll(filepath.Dir(fleetFile()), 0o700)
	return os.WriteFile(fleetFile(), jsonMarshal(p), 0o600)
}

// SetPrimary marks node as control-plane primary; clears primary on others.
func SetPrimary(nodeID string) error {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("node id required")
	}
	list, err := nodes.List()
	if err != nil {
		return err
	}
	found := false
	for _, n := range list {
		if n.ID == nodeID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown node %s", nodeID)
	}
	for _, n := range list {
		labs := n.Labels
		if labs == nil {
			labs = map[string]string{}
		}
		if n.ID == nodeID {
			labs[LabelControlPlane] = "primary"
			if labs[LabelRegion] == "" {
				// heuristic: core role + non-relay → abroad
				if n.Role == "secondary" {
					labs[LabelRegion] = "ru"
				} else {
					labs[LabelRegion] = "abroad"
				}
			}
		} else if labs[LabelControlPlane] == "primary" {
			labs[LabelControlPlane] = "secondary"
		}
		n.Labels = labs
		_, _ = nodes.UpsertFromDevice(n)
	}
	p := LoadPolicy()
	p.PrimaryNodeID = nodeID
	if n, ok, _ := nodes.Get(nodeID); ok && n.PublicIP != "" {
		p.PrimarySSH = "root@" + n.PublicIP
	}
	if p.BotNodeID == "" {
		p.BotNodeID = nodeID // TG bot stays on control-plane primary by default
	}
	return SavePolicy(p)
}

// SetSecondary records secondary and region=ru when role is relay.
func SetSecondary(nodeID string) error {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("node id required")
	}
	n, ok, err := nodes.Get(nodeID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unknown node %s", nodeID)
	}
	labs := n.Labels
	if labs == nil {
		labs = map[string]string{}
	}
	labs[LabelControlPlane] = "secondary"
	if labs[LabelRegion] == "" {
		labs[LabelRegion] = "ru"
	}
	n.Labels = labs
	if _, err := nodes.UpsertFromDevice(n); err != nil {
		return err
	}
	p := LoadPolicy()
	p.SecondaryNodeID = nodeID
	if n.PublicIP != "" {
		p.SecondarySSH = "root@" + n.PublicIP
	}
	if p.LampacNodeID == "" {
		p.LampacNodeID = nodeID // Lampac prefer RU by default
	}
	return SavePolicy(p)
}

// SetServiceNode sets preferred host for lampac|bot.
func SetServiceNode(service, nodeID string) error {
	service = strings.ToLower(strings.TrimSpace(service))
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("node id required")
	}
	if _, ok, err := nodes.Get(nodeID); err != nil || !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("unknown node %s", nodeID)
	}
	p := LoadPolicy()
	switch service {
	case "lampac":
		p.LampacNodeID = nodeID
	case "bot", "telegram", "tg":
		p.BotNodeID = nodeID
	default:
		return fmt.Errorf("service must be lampac|bot")
	}
	return SavePolicy(p)
}

// StatusSummary human-readable fleet policy + nodes.
func StatusSummary() string {
	p := LoadPolicy()
	var b strings.Builder
	b.WriteString("Fleet policy (primary / secondary)" + "\n")
	b.WriteString("  model: primary=abroad control-plane; secondary=RU VPN entry only" + "\n")
	b.WriteString("  note:  VPN agent may still use role=relay internally; no service mirror" + "\n")
	b.WriteString(fmt.Sprintf("  primary:   %s  ssh=%s\n", empty(p.PrimaryNodeID, "(unset)"), empty(p.PrimarySSH, "-")))
	b.WriteString(fmt.Sprintf("  secondary: %s  ssh=%s\n", empty(p.SecondaryNodeID, "(unset)"), empty(p.SecondarySSH, "-")))
	b.WriteString(fmt.Sprintf("  lampac ->  %s\n", empty(p.LampacNodeID, "(unset)")))
	b.WriteString(fmt.Sprintf("  bot ->     %s\n", empty(p.BotNodeID, "(unset)")))
	b.WriteString(fmt.Sprintf("  sync:      %v\n", p.SyncEnabled))
	list, _ := nodes.List()
	b.WriteString("Nodes" + "\n")
	for _, n := range list {
		reg := "-"
		if n.Labels != nil && n.Labels[LabelRegion] != "" {
			reg = n.Labels[LabelRegion]
		}
		disp := n.Role
		if disp == "relay" {
			disp = "secondary"
		}
		if n.Labels != nil && n.Labels[LabelControlPlane] == "primary" {
			disp = "primary"
		}
		if n.Labels != nil && n.Labels[LabelControlPlane] == "secondary" {
			disp = "secondary"
		}
		b.WriteString(fmt.Sprintf("  * %s  host=%s fleet=%s ip=%s status=%s region=%s\n",
			n.ID, n.Hostname, disp, n.PublicIP, n.Status, reg))
	}
	return b.String()
}


func empty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
