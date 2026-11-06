package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

// FetchMeta loads upgrade.json for tag from local release store, then GitHub release asset.
// Missing file is not an error: returns ok=false.
func FetchMeta(tag string) (meta Meta, ok bool, err error) {
	tag = strings.TrimSpace(tag)
	if tag != "" && !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	if tag == "" || tag == "v" {
		return Meta{}, false, fmt.Errorf("empty tag")
	}
	// Local mirror first
	local := filepath.Join(ndupdate.LocalReleasesDir(), tag, "upgrade.json")
	if b, err := os.ReadFile(local); err == nil {
		if err := json.Unmarshal(b, &meta); err != nil {
			return Meta{}, false, fmt.Errorf("local upgrade.json: %w", err)
		}
		return meta, true, nil
	}
	url := fmt.Sprintf("https://github.com/PavelNeyman/netductor/releases/download/%s/upgrade.json", tag)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return Meta{}, false, nil // treat network as "no meta"
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Meta{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Meta{}, false, nil
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Meta{}, false, err
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return Meta{}, false, fmt.Errorf("upgrade.json: %w", err)
	}
	return meta, true, nil
}

// LocalVersion reads /etc/netductor/VERSION (best-effort).
func LocalVersion() string {
	for _, p := range []string{"/etc/netductor/VERSION"} {
		if b, err := os.ReadFile(p); err == nil {
			return normalizeVer(string(b))
		}
	}
	return ""
}
