package aptguard

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Policy is the centralized allowlist for APT sources (versioned).
type Policy struct {
	Version int              `yaml:"version" json:"version"`
	Debian  DebianPolicy     `yaml:"debian" json:"debian"`
	Repos   []RepositorySpec `yaml:"repositories" json:"repositories"`
	Scope   PolicyScope      `yaml:"scope" json:"scope"`
}

type DebianPolicy struct {
	AllowedReleases []string `yaml:"allowed_releases" json:"allowed_releases"`
}

type PolicyScope struct {
	// All, primary, secondary, or explicit server IDs (empty = all).
	ApplyTo string   `yaml:"apply_to" json:"apply_to"` // all|primary|secondary|list
	Servers []string `yaml:"servers,omitempty" json:"servers,omitempty"`
}

type RepositorySpec struct {
	ID         string   `yaml:"id" json:"id"`
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	Type       string   `yaml:"type" json:"type"` // debian|debian-security|third-party
	URIs       []string `yaml:"uris" json:"uris"`
	Suites     []string `yaml:"suites" json:"suites"`
	Components []string `yaml:"components" json:"components"`
	SignedBy   []string `yaml:"signed_by,omitempty" json:"signed_by,omitempty"`
	Required   bool     `yaml:"required,omitempty" json:"required,omitempty"`
}

// DefaultPolicy is a conservative Debian bookworm/trixie baseline.
func DefaultPolicy() Policy {
	return Policy{
		Version: 1,
		Debian:  DebianPolicy{AllowedReleases: []string{"bookworm", "trixie"}},
		Scope:   PolicyScope{ApplyTo: "all"},
		Repos: []RepositorySpec{
			{
				ID: "debian-main", Enabled: true, Type: "debian", Required: true,
				URIs:       []string{"https://deb.debian.org/debian", "http://deb.debian.org/debian"},
				Suites:     []string{"bookworm", "bookworm-updates", "trixie", "trixie-updates"},
				Components: []string{"main", "contrib", "non-free-firmware"},
			},
			{
				ID: "debian-security", Enabled: true, Type: "debian-security", Required: true,
				URIs:       []string{"https://security.debian.org/debian-security", "http://security.debian.org/debian-security"},
				Suites:     []string{"bookworm-security", "trixie-security"},
				Components: []string{"main", "contrib", "non-free-firmware"},
			},
		},
	}
}

func (p Policy) Validate() error {
	if p.Version < 1 {
		return fmt.Errorf("aptguard: policy version must be >= 1")
	}
	if len(p.Debian.AllowedReleases) == 0 {
		return fmt.Errorf("aptguard: allowed_releases empty")
	}
	seen := map[string]bool{}
	for _, r := range p.Repos {
		id := strings.TrimSpace(r.ID)
		if id == "" {
			return fmt.Errorf("aptguard: repository id empty")
		}
		if seen[id] {
			return fmt.Errorf("aptguard: duplicate repository id %q", id)
		}
		seen[id] = true
		if r.Enabled {
			if len(r.URIs) == 0 {
				return fmt.Errorf("aptguard: repository %q has no uris", id)
			}
			for _, u := range r.URIs {
				if strings.TrimSpace(u) == "" {
					return fmt.Errorf("aptguard: repository %q empty uri", id)
				}
			}
			if len(r.Suites) == 0 {
				return fmt.Errorf("aptguard: repository %q has no suites", id)
			}
		}
	}
	return nil
}

func LoadPolicyFile(path string) (Policy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	if err := yaml.Unmarshal(b, &p); err != nil {
		return Policy{}, fmt.Errorf("aptguard: parse policy: %w", err)
	}
	if err := p.Validate(); err != nil {
		return Policy{}, err
	}
	return p, nil
}

func SavePolicyFile(path string, p Policy) error {
	if err := p.Validate(); err != nil {
		return err
	}
	b, err := yaml.Marshal(&p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
