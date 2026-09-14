package vpn

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// ImportSNIPresetsFromURL fetches a JSON array of {name,sni,note} or string list and merges into presets file.
func ImportSNIPresetsFromURL(url string) (int, error) {
	if url == "" {
		url = "https://raw.githubusercontent.com/openlibrecommunity/twl/main/data/sni_good.json"
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return 0, err
	}
	var presets []SNIPreset
	if json.Unmarshal(body, &presets) != nil {
		// try string array
		var ss []string
		if json.Unmarshal(body, &ss) != nil {
			return 0, fmt.Errorf("unsupported preset JSON (HTTP %d)", resp.StatusCode)
		}
		for _, s := range ss {
			if s == "" {
				continue
			}
			presets = append(presets, SNIPreset{Name: s, SNI: s, Note: "imported"})
		}
	}
	if len(presets) == 0 {
		return 0, fmt.Errorf("empty import")
	}
	// merge with defaults
	seen := map[string]bool{}
	var merged []SNIPreset
	for _, p := range append(defaultSNIPresets(), presets...) {
		if p.SNI == "" || seen[p.SNI] {
			continue
		}
		seen[p.SNI] = true
		merged = append(merged, p)
	}
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	raw, _ := json.MarshalIndent(merged, "", "  ")
	if err := os.WriteFile(sniPresetsPath(), append(raw, 10), 0o644); err != nil {
		return 0, err
	}
	// cache copy
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "sni_import_cache.json"), append(raw, 10), 0o600)
	return len(merged), nil
}
