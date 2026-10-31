package aptguard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Dir returns state directory for aptguard (default under NETDUCTOR_STATE).
func Dir(stateRoot string) string {
	if stateRoot == "" {
		stateRoot = "/var/lib/netductor"
	}
	return filepath.Join(stateRoot, "aptguard")
}

func PolicyPath(stateRoot string) string {
	return filepath.Join(Dir(stateRoot), "policy.yaml")
}

func LatestPath(stateRoot, serverID string) string {
	safe := filepath.Base(serverID)
	if safe == "" || safe == "." {
		safe = "local"
	}
	return filepath.Join(Dir(stateRoot), "reports", safe+".json")
}

func HistoryDir(stateRoot, serverID string) string {
	safe := filepath.Base(serverID)
	if safe == "" || safe == "." {
		safe = "local"
	}
	return filepath.Join(Dir(stateRoot), "history", safe)
}

// EnsurePolicy loads policy from disk or writes DefaultPolicy once.
func EnsurePolicy(stateRoot string) (Policy, error) {
	path := PolicyPath(stateRoot)
	if _, err := os.Stat(path); err == nil {
		return LoadPolicyFile(path)
	}
	p := DefaultPolicy()
	if err := os.MkdirAll(Dir(stateRoot), 0o755); err != nil {
		return Policy{}, err
	}
	if err := SavePolicyFile(path, p); err != nil {
		return Policy{}, err
	}
	return p, nil
}

// SaveReport writes latest + history snapshot.
func SaveReport(stateRoot string, rep Report) error {
	if err := os.MkdirAll(filepath.Dir(LatestPath(stateRoot, rep.ServerID)), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(LatestPath(stateRoot, rep.ServerID), b, 0o644); err != nil {
		return err
	}
	hd := HistoryDir(stateRoot, rep.ServerID)
	if err := os.MkdirAll(hd, 0o755); err != nil {
		return err
	}
	name := rep.CheckedAt.UTC().Format("20060102T150405Z") + ".json"
	return os.WriteFile(filepath.Join(hd, name), b, 0o644)
}

// LoadLatest returns last saved report or error if missing.
func LoadLatest(stateRoot, serverID string) (Report, error) {
	b, err := os.ReadFile(LatestPath(stateRoot, serverID))
	if err != nil {
		return Report{}, err
	}
	var r Report
	if err := json.Unmarshal(b, &r); err != nil {
		return Report{}, err
	}
	return r, nil
}

// Stale if no report or older than maxAge.
func Stale(rep Report, maxAge time.Duration) bool {
	if rep.CheckedAt.IsZero() {
		return true
	}
	return time.Since(rep.CheckedAt) > maxAge
}

// DiffFindings returns codes that are new relative to previous report.
func DiffFindings(prev, cur Report) (added, removed []Finding) {
	prevKeys := map[string]Finding{}
	for _, f := range prev.Findings {
		prevKeys[findingKey(f)] = f
	}
	curKeys := map[string]Finding{}
	for _, f := range cur.Findings {
		curKeys[findingKey(f)] = f
	}
	for k, f := range curKeys {
		if _, ok := prevKeys[k]; !ok {
			added = append(added, f)
		}
	}
	for k, f := range prevKeys {
		if _, ok := curKeys[k]; !ok {
			removed = append(removed, f)
		}
	}
	return added, removed
}

func findingKey(f Finding) string {
	return fmt.Sprintf("%s|%s|%s", f.Code, f.SourceFile, f.Message)
}
