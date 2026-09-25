package operator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/sites"
)

// SiteSpec — MikroTik (+ optional RPi edge id) from operator machine.
type SiteSpec struct {
	SiteID   string
	Name     string
	RPiID    string
	MikroTikID string
	RPiLAN   string
	MTHost   string
	MTUser   string
	MTPass   string
	MTPort   int
	DoPush   bool
	// OperatorKeyPath private key; .pub used for harden
	OperatorKeyPath string
}

// DeploySite saves site registry + optional RSC push + SSH harden.
func DeploySite(s SiteSpec) (string, error) {
	var b strings.Builder
	id := strings.TrimSpace(s.SiteID)
	if id == "" {
		return "", fmt.Errorf("site id required")
	}
	if !ValidDeviceID(id) && !ValidHost(id) {
		// site ids are similar to device ids
		if strings.ContainsAny(id, " \t/\\") {
			return "", fmt.Errorf("invalid site id")
		}
	}
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = id
	}
	st, err := sites.Upsert(sites.Site{
		ID: id, Name: name, RPiID: strings.TrimSpace(s.RPiID), MikroTikID: strings.TrimSpace(s.MikroTikID),
	})
	if err != nil {
		return "", err
	}
	b.WriteString(fmt.Sprintf("site saved: %s rpi=%s mt=%s\n", st.ID, st.RPiID, st.MikroTikID))

	gw := strings.TrimSpace(s.RPiLAN)
	rsc, err := sites.RSCForSiteWithGateway(id, gw)
	if err != nil {
		return b.String(), err
	}
	b.WriteString("--- RSC ---\n")
	b.WriteString(rsc)
	b.WriteString("\n")

	if !s.DoPush {
		b.WriteString("push skipped\n")
		return b.String(), nil
	}
	host := strings.TrimSpace(s.MTHost)
	if !ValidHost(host) {
		return b.String(), fmt.Errorf("invalid MikroTik host")
	}
	user := s.MTUser
	if user == "" {
		user = "admin"
	}
	port := s.MTPort
	if port <= 0 {
		port = 22
	}
	if err := mikrotik.PushRSC(host, user, s.MTPass, nil, rsc, port); err != nil {
		return b.String(), fmt.Errorf("push: %w", err)
	}
	b.WriteString("RSC pushed to " + host + "\n")

	c := mikrotik.ConnOpts{Host: host, User: user, Password: s.MTPass, Port: port}
	if pub := loadOperatorPub(s.OperatorKeyPath); pub != "" {
		if err := mikrotik.HardenSSH(c, pub); err != nil {
			b.WriteString("SSH harden: " + err.Error() + "\n")
		} else {
			b.WriteString("operator pubkey installed on MikroTik\n")
		}
	}
	return b.String(), nil
}

func loadOperatorPub(keyPath string) string {
	keyPath = strings.TrimSpace(keyPath)
	if keyPath == "" {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, ".ssh", "netductor_primary")
	}
	if strings.HasPrefix(keyPath, "~/") {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[2:])
	}
	b, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// MikroTikAction runs one-shot manage (identity/resource/routes/ping).
func MikroTikAction(host, user, pass string, port int, action string) (string, error) {
	if !ValidHost(host) {
		return "", fmt.Errorf("invalid host")
	}
	if user == "" {
		user = "admin"
	}
	if port <= 0 {
		port = 22
	}
	c := mikrotik.ConnOpts{Host: host, User: user, Password: pass, Port: port}
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "identity", "":
		return mikrotik.Identity(c)
	case "resource":
		return mikrotik.Resource(c)
	case "routes":
		return mikrotik.Routes(c)
	case "ping":
		return mikrotik.PingFromMT(c, "1.1.1.1")
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
}

// ParsePort soft parse.
func ParsePort(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || n > 65535 {
		return def
	}
	return n
}
