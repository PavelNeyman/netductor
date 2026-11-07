package stack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/upgrade"
)

// chainJournal tracks in-progress multi-hop apply (CF-15 phase C resume hint).
type chainJournal struct {
	Target    string   `json:"target"`
	Path      []string `json:"path"`
	Done      []string `json:"done"`
	Current   string   `json:"current,omitempty"`
	StartedAt string   `json:"started_at"`
	UpdatedAt string   `json:"updated_at"`
	Error     string   `json:"error,omitempty"`
}

func chainJournalPath() string {
	return filepath.Join(paths.StateDir(), "stack", "chain-apply.json")
}

func writeChainJournal(j chainJournal) {
	j.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = os.MkdirAll(filepath.Dir(chainJournalPath()), 0o755)
	b, _ := json.MarshalIndent(j, "", "  ")
	_ = os.WriteFile(chainJournalPath(), b, 0o644)
}

func clearChainJournal() {
	_ = os.Remove(chainJournalPath())
}

func loadChainJournal() (chainJournal, bool) {
	var j chainJournal
	b, err := os.ReadFile(chainJournalPath())
	if err != nil {
		return j, false
	}
	if json.Unmarshal(b, &j) != nil || len(j.Path) == 0 {
		return j, false
	}
	return j, true
}

// ResolveApplyPath builds CF-15 hop list for tag using local release store + target meta.
func ResolveApplyPath(tag string) ([]upgrade.PathStep, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		var err error
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			return nil, err
		}
	}
	tag = strings.TrimPrefix(tag, "v")
	local := upgrade.LocalVersion()
	var hist []upgrade.Meta
	if tags, err := ndupdate.ListLocalTags(); err == nil {
		for _, tg := range tags {
			if m, ok, _ := upgrade.FetchMeta(tg); ok {
				hist = append(hist, m)
			}
		}
	}
	if m, ok, err := upgrade.FetchMeta(tag); err != nil {
		return nil, err
	} else if ok {
		hist = append(hist, m)
	}
	return upgrade.ResolveUpgradePath(local, tag, hist)
}

// FormatApplyPath is a CLI one-liner for hop chain.
func FormatApplyPath(steps []upgrade.PathStep) string {
	return upgrade.FormatPathCLI(steps)
}

// ApplyChain resolves the upgrade path and applies each hop in order (CF-15 phase C, control node).
// Writes stack/chain-apply.json during the run for operator resume visibility; clears on success.
func ApplyChain(tag string) error {
	steps, err := ResolveApplyPath(tag)
	if err != nil {
		return err
	}
	if len(steps) == 0 {
		return fmt.Errorf("stack apply --chain: empty path")
	}
	pathTags := make([]string, len(steps))
	for i, s := range steps {
		pathTags[i] = "v" + strings.TrimPrefix(s.Tag, "v")
	}
	fmt.Fprintf(os.Stderr, "stack apply --chain: %s\n", FormatApplyPath(steps))
	j := chainJournal{
		Target:    pathTags[len(pathTags)-1],
		Path:      pathTags,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	// Resume: reuse Done from prior failed journal when same target.
	if prev, ok := loadChainJournal(); ok && strings.TrimPrefix(prev.Target, "v") == strings.TrimPrefix(j.Target, "v") {
		j.Done = append([]string{}, prev.Done...)
		j.StartedAt = prev.StartedAt
		if len(prev.Done) > 0 {
			fmt.Fprintf(os.Stderr, "stack apply --chain: resume after %v\n", prev.Done)
		}
	}
	j.Error = ""
	writeChainJournal(j)
	doneSet := map[string]bool{}
	for _, d := range j.Done {
		doneSet[d] = true
		doneSet[strings.TrimPrefix(d, "v")] = true
	}
	for i, s := range steps {
		hop := "v" + strings.TrimPrefix(s.Tag, "v")
		if doneSet[hop] || doneSet[strings.TrimPrefix(hop, "v")] {
			fmt.Fprintf(os.Stderr, "stack apply --chain: skip done hop %d/%d %s\n", i+1, len(steps), hop)
			continue
		}
		j.Current = hop
		writeChainJournal(j)
		fmt.Fprintf(os.Stderr, "stack apply --chain: hop %d/%d %s\n", i+1, len(steps), hop)
		if err := Apply(hop); err != nil {
			j.Error = err.Error()
			writeChainJournal(j)
			return fmt.Errorf("stack apply --chain hop %s: %w (journal %s)", hop, err, chainJournalPath())
		}
		j.Done = append(j.Done, hop)
		j.Current = ""
		writeChainJournal(j)
	}
	clearChainJournal()
	return nil
}
