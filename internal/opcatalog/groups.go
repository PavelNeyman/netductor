package opcatalog

import "strings"

// Group is a Day-2 navigation bucket shared by Web Control, TG Tools, TUI.
// Multiple Action.Section values map into one group (fewer tabs, same catalog).
type Group struct {
	ID       string   `json:"id"`
	LabelEN  string   `json:"label_en"`
	LabelRU  string   `json:"label_ru"`
	Sections []string `json:"sections"`
	Surfaces []string `json:"surfaces,omitempty"`
}

// Groups is the single Day-2 IA. UI must not invent primary tabs outside this list.
func Groups() []Group {
	all := allSurfaces()
	return []Group{
		{ID: "home", LabelEN: "Home", LabelRU: "Главное", Sections: []string{"overview", "updates"}, Surfaces: all},
		{ID: "users", LabelEN: "Users", LabelRU: "Users", Sections: []string{"vpn"}, Surfaces: all},
		{ID: "fleet", LabelEN: "Fleet", LabelRU: "Флот", Sections: []string{"nodes"}, Surfaces: all},
		{ID: "edge", LabelEN: "Edge", LabelRU: "Edge", Sections: []string{"edge"}, Surfaces: all},
		{ID: "media", LabelEN: "Media", LabelRU: "Медиа", Sections: []string{"nvr"}, Surfaces: all},
		{ID: "data", LabelEN: "Data", LabelRU: "Данные", Sections: []string{"backup", "logs", "dns", "git"}, Surfaces: all},
		{ID: "adv", LabelEN: "Advanced", LabelRU: "Ещё", Sections: []string{"probes"}, Surfaces: all},
	}
}

// GroupLabel returns EN/RU label for group id.
func GroupLabel(id, lang string) string {
	for _, g := range Groups() {
		if g.ID == id {
			if lang == "ru" && g.LabelRU != "" {
				return g.LabelRU
			}
			return g.LabelEN
		}
	}
	return id
}

// ActionsInGroup returns actions for a group filtered by surface.
func ActionsInGroup(groupID, surface string) []Action {
	var secs map[string]bool
	for _, g := range Groups() {
		if g.ID != groupID {
			continue
		}
		secs = make(map[string]bool)
		for _, s := range g.Sections {
			secs[s] = true
		}
		break
	}
	if secs == nil {
		return nil
	}
	var out []Action
	for _, a := range ForSurface(surface) {
		if secs[a.Section] {
			out = append(out, a)
		}
	}
	return out
}

// ByGroup maps group id → actions for surface.
func ByGroup(surface string) map[string][]Action {
	if surface == "" {
		surface = SurfWeb
	}
	m := make(map[string][]Action)
	for _, g := range Groups() {
		m[g.ID] = ActionsInGroup(g.ID, surface)
	}
	return m
}

// GroupSectionSet helper for docs.
func GroupSectionsCSV(g Group) string {
	return strings.Join(g.Sections, ", ")
}
