// Package cluster builds a Phase 4 fleet status view from local control + secondary registry.
package cluster

import (
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/controlplane"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/version"
)

// Member is one row in the cluster view (control self or egress secondary).
type Member struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"` // control | egress
	Hostname   string `json:"hostname,omitempty"`
	PublicIP   string `json:"public_ip,omitempty"`
	Online     bool   `json:"online"`
	Version    string `json:"version,omitempty"`
	ControlRole string `json:"control_role,omitempty"`
	WriteAllowed bool `json:"write_allowed,omitempty"`
	IsLeader   bool   `json:"is_leader,omitempty"`
	SingBoxOK  bool   `json:"singbox_ok,omitempty"`
	UplinkOK   bool   `json:"uplink_ok,omitempty"`
	LastSeen   string `json:"last_seen,omitempty"`
	Note       string `json:"note,omitempty"`
}

// View is the aggregated snapshot for Op/Web/TG.
type View struct {
	OK           bool     `json:"ok"`
	LeaderID     string   `json:"leader_id"`
	WriteAllowed bool     `json:"write_allowed"`
	Members      []Member `json:"members"`
	TS           string   `json:"ts"`
}

// Build returns self + secondaries (+ optional nodes registry rows not already listed).
func Build() View {
	hn, _ := os.Hostname()
	role := controlplane.Get()
	leader := controlplane.WriteAllowed()
	selfID := "control-self"
	if hn != "" {
		selfID = "control:" + hn
	}
	v := View{
		OK:           true,
		LeaderID:     "",
		WriteAllowed: leader,
		TS:           time.Now().UTC().Format(time.RFC3339),
	}
	self := Member{
		ID:           selfID,
		Kind:         "control",
		Hostname:     hn,
		Online:       true,
		Version:      version.Release,
		ControlRole:  string(role),
		WriteAllowed: leader,
		IsLeader:     leader,
		Note:         "local control node",
	}
	if leader {
		v.LeaderID = selfID
	} else {
		self.Note = "standby — SoT writes fenced"
	}
	v.Members = append(v.Members, self)

	seenIP := map[string]bool{}
	for _, d := range secondary.List() {
		online := !d.LastSeen.IsZero() && time.Since(d.LastSeen) < 3*time.Minute
		m := Member{
			ID:        d.ID,
			Kind:      "egress",
			Hostname:  d.Name,
			PublicIP:  d.PublicIP,
			Online:    online,
			Version:   d.Version,
			SingBoxOK: d.SingBoxOK,
			UplinkOK:  d.UplinkOK,
			Note:      "secondary agent heartbeat",
		}
		if !d.LastSeen.IsZero() {
			m.LastSeen = d.LastSeen.UTC().Format(time.RFC3339)
		}
		if d.PublicIP != "" {
			seenIP[d.PublicIP] = true
		}
		v.Members = append(v.Members, m)
	}

	// Optional fleet nodes registry (edges etc.) — status only, no SSH probe.
	nodeList, _ := nodes.List()
	for _, n := range nodeList {
		if n.Role == "secondary" || n.Role == "relay" {
			if n.PublicIP != "" && seenIP[n.PublicIP] {
				continue
			}
		}
		if n.Role == "primary" || n.Role == "core" {
			continue // covered by self
		}
		online := strings.EqualFold(n.Status, "online")
		v.Members = append(v.Members, Member{
			ID:       n.ID,
			Kind:     kindFromNode(n),
			Hostname: n.Hostname,
			PublicIP: n.PublicIP,
			Online:   online,
			Note:     "nodes registry",
		})
	}
	return v
}

func kindFromNode(n nodes.Node) string {
	switch strings.ToLower(n.Role) {
	case "secondary", "relay", "egress":
		return "egress"
	case "edge":
		return "edge"
	case "build":
		return "build"
	default:
		if n.Kind != "" {
			return n.Kind
		}
		return "member"
	}
}
