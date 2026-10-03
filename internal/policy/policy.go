// Package policy implements the service catalog and per-subject access policies
// (VPN users and edge routers). See docs/PLAN-SERVICE-ACCESS-POLICY.md.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// AccessPolicy is attached to a VPN user or edge device.
type AccessPolicy struct {
	AllowInternet bool     `json:"allow_internet"`
	Services      []string `json:"services"`
	ServicesMode  string   `json:"services_mode,omitempty"` // list | all
	UpdatedAt     string   `json:"updated_at,omitempty"`
	UpdatedBy     string   `json:"updated_by,omitempty"`
}

// DefaultUserPolicy is applied on migration when a user has no policy yet.
func DefaultUserPolicy() AccessPolicy {
	return AccessPolicy{
		AllowInternet: true,
		Services:      []string{},
		ServicesMode:  "list",
	}
}

// DefaultEdgePolicy is applied when an edge device has no policy yet.
func DefaultEdgePolicy() AccessPolicy {
	return AccessPolicy{
		AllowInternet: true,
		Services:      []string{},
		ServicesMode:  "list",
	}
}

// Normalize fills defaults and validates service ids against the catalog when strict.
func (p *AccessPolicy) Normalize() {
	if p.ServicesMode != "all" {
		p.ServicesMode = "list"
	}
	if p.Services == nil {
		p.Services = []string{}
	}
	// dedupe + sort
	seen := map[string]struct{}{}
	out := make([]string, 0, len(p.Services))
	for _, s := range p.Services {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" || s == "internet" {
			continue // internet is AllowInternet flag
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	p.Services = out
}

// Validate checks mode and that service ids exist in catalog (when cat non-nil).
func (p AccessPolicy) Validate(cat *Catalog) error {
	if p.ServicesMode != "" && p.ServicesMode != "list" && p.ServicesMode != "all" {
		return fmt.Errorf("services_mode must be list or all")
	}
	if cat == nil {
		return nil
	}
	if p.ServicesMode == "all" {
		return nil
	}
	known := map[string]struct{}{}
	for _, s := range cat.Services {
		if s.Disabled {
			continue
		}
		known[s.ID] = struct{}{}
	}
	for _, id := range p.Services {
		if id == "internet" {
			continue
		}
		if _, ok := known[id]; !ok {
			return fmt.Errorf("unknown service id %q", id)
		}
	}
	return nil
}

// EffectiveServices returns internal service ids allowed by this policy.
func (p AccessPolicy) EffectiveServices(cat *Catalog) []string {
	p.Normalize()
	if cat == nil {
		cat = DefaultCatalog()
	}
	if p.ServicesMode == "all" {
		var ids []string
		for _, s := range cat.Services {
			if s.Disabled || s.Kind == KindEgress {
				continue
			}
			ids = append(ids, s.ID)
		}
		sort.Strings(ids)
		return ids
	}
	return append([]string{}, p.Services...)
}

// Allows reports whether service id is permitted (internet via AllowInternet).
func (p AccessPolicy) Allows(serviceID string, cat *Catalog) bool {
	serviceID = strings.TrimSpace(strings.ToLower(serviceID))
	if serviceID == "" || serviceID == "internet" {
		return p.AllowInternet
	}
	for _, id := range p.EffectiveServices(cat) {
		if id == serviceID {
			return true
		}
	}
	return false
}

// Touch sets UpdatedAt/By.
func (p *AccessPolicy) Touch(by string) {
	p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	p.UpdatedBy = by
}

// CatalogPath returns on-disk catalog path.
func CatalogPath() string {
	return filepath.Join(paths.StateDir(), "service-catalog.json")
}

// EnsureCatalog loads catalog or writes the default seed.
func EnsureCatalog() (*Catalog, error) {
	if _, err := os.Stat(CatalogPath()); err != nil {
		c := DefaultCatalog()
		if err := SaveCatalog(c); err != nil {
			return c, err
		}
		return c, nil
	}
	c, err := LoadCatalog()
	if err != nil {
		return nil, err
	}
	if c == nil || len(c.Services) == 0 {
		c = DefaultCatalog()
		if err := SaveCatalog(c); err != nil {
			return c, err
		}
	}
	return c, nil
}

// LoadCatalog reads catalog from disk (empty error → default not written).
func LoadCatalog() (*Catalog, error) {
	b, err := os.ReadFile(CatalogPath())
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultCatalog(), nil
		}
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Version == 0 {
		c.Version = 1
	}
	if c.Services == nil {
		c.Services = []Service{}
	}
	return &c, nil
}

// SaveCatalog writes catalog atomically.
func SaveCatalog(c *Catalog) error {
	if c == nil {
		return fmt.Errorf("nil catalog")
	}
	if err := c.Validate(); err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(CatalogPath()), 0o755)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := CatalogPath() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, CatalogPath())
}

// Service kinds.
const (
	KindEgress   = "egress"
	KindInternal = "internal"
)

// Endpoint is a reachable address for an internal service.
type Endpoint struct {
	Network string `json:"network,omitempty"` // service | loopback | lan
	Addr    string `json:"addr,omitempty"`
	Port    int    `json:"port,omitempty"`
	Proto   string `json:"proto,omitempty"` // tcp | udp
}

// Service is one catalog entry.
type Service struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Kind        string     `json:"kind"` // egress | internal
	Description string     `json:"description,omitempty"`
	Endpoints   []Endpoint `json:"endpoints,omitempty"`
	Disabled    bool       `json:"disabled,omitempty"`
}

// Catalog is the global service registry on primary.
type Catalog struct {
	Version  int       `json:"version"`
	Services []Service `json:"services"`
}

// DefaultCatalog returns the product seed.
func DefaultCatalog() *Catalog {
	return &Catalog{
		Version: 1,
		Services: []Service{
			{
				ID:          "internet",
				Title:       "Internet egress",
				Kind:        KindEgress,
				Description: "General non-RU / default proxy path",
			},
			{
				ID:          "lampac",
				Title:       "Lampac",
				Kind:        KindInternal,
				Description: "Media / Lampac on primary",
				Endpoints:   []Endpoint{{Network: "loopback", Addr: "127.0.0.1", Port: 9118, Proto: "tcp"}},
			},
			{
				ID:          "git",
				Title:       "Git",
				Kind:        KindInternal,
				Description: "Bare git / thin git on primary",
				Endpoints:   []Endpoint{{Network: "loopback", Addr: "127.0.0.1", Port: 2222, Proto: "tcp"}},
			},
			{
				ID:          "registry",
				Title:       "OCI registry",
				Kind:        KindInternal,
				Description: "Local container registry",
				Endpoints:   []Endpoint{{Network: "loopback", Addr: "127.0.0.1", Port: 5000, Proto: "tcp"}},
			},
			{
				ID:          "nvr",
				Title:       "NVR / go2rtc",
				Kind:        KindInternal,
				Description: "Camera NVR UI and streams (when enabled)",
				Endpoints:   []Endpoint{{Network: "loopback", Addr: "127.0.0.1", Port: 1984, Proto: "tcp"}},
			},
		},
	}
}

// Validate catalog structure.
func (c *Catalog) Validate() error {
	if c.Version < 1 {
		c.Version = 1
	}
	seen := map[string]struct{}{}
	for i := range c.Services {
		s := &c.Services[i]
		s.ID = strings.TrimSpace(strings.ToLower(s.ID))
		if !idRe.MatchString(s.ID) {
			return fmt.Errorf("invalid service id %q", s.ID)
		}
		if _, ok := seen[s.ID]; ok {
			return fmt.Errorf("duplicate service id %q", s.ID)
		}
		seen[s.ID] = struct{}{}
		if s.Kind != KindEgress && s.Kind != KindInternal {
			return fmt.Errorf("service %s: kind must be egress or internal", s.ID)
		}
		if strings.TrimSpace(s.Title) == "" {
			s.Title = s.ID
		}
	}
	return nil
}

// Get returns a service by id.
func (c *Catalog) Get(id string) (Service, bool) {
	id = strings.TrimSpace(strings.ToLower(id))
	for _, s := range c.Services {
		if s.ID == id {
			return s, true
		}
	}
	return Service{}, false
}

// Upsert adds or replaces a service.
func (c *Catalog) Upsert(s Service) error {
	s.ID = strings.TrimSpace(strings.ToLower(s.ID))
	if !idRe.MatchString(s.ID) {
		return fmt.Errorf("invalid service id")
	}
	if s.Kind != KindEgress && s.Kind != KindInternal {
		return fmt.Errorf("kind must be egress or internal")
	}
	for i := range c.Services {
		if c.Services[i].ID == s.ID {
			c.Services[i] = s
			return c.Validate()
		}
	}
	c.Services = append(c.Services, s)
	return c.Validate()
}

// SoftDisable marks a service disabled (keeps id for UI history).
func (c *Catalog) SoftDisable(id string) error {
	id = strings.TrimSpace(strings.ToLower(id))
	for i := range c.Services {
		if c.Services[i].ID == id {
			c.Services[i].Disabled = true
			return nil
		}
	}
	return fmt.Errorf("service not found: %s", id)
}

// PolicyFile is optional side store; primary storage is embedded on user/device.
// Kept for future bulk export.
func PolicySnapshotPath() string {
	return filepath.Join(paths.StateDir(), "access-policies.json")
}
