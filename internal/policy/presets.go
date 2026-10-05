package policy

import "strings"

// Named presets for operator UIs (media = internet + lampac/nvr; full = all internal).
const (
	PresetMedia = "media"
	PresetFull  = "full"
	PresetNone  = "none" // internet only, no internal
)

// ApplyPreset returns a policy derived from a named preset.
func ApplyPreset(name string, base AccessPolicy) AccessPolicy {
	p := base
	p.Normalize()
	switch strings.ToLower(strings.TrimSpace(name)) {
	case PresetFull, "all":
		p.AllowInternet = true
		p.ServicesMode = "all"
		p.Services = []string{}
	case PresetMedia:
		p.AllowInternet = true
		p.ServicesMode = "list"
		// Only services that are actually enabled (e.g. nvr when recording on)
		svcs := []string{}
		for _, id := range []string{"lampac", "nvr"} {
			if ServiceEnabled != nil && !ServiceEnabled(id) {
				continue
			}
			svcs = append(svcs, id)
		}
		p.Services = svcs
	case PresetNone, "internet":
		p.AllowInternet = true
		p.ServicesMode = "list"
		p.Services = []string{}
	default:
		// unknown: leave base
	}
	return p
}
