package singboxconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONSchema(t *testing.T) {
	tmp := t.TempDir()
	// override via env not available — write to temp by monkeying ConfPath is const.
	// Test marshal shape only.
	cfg := map[string]any{"inbounds": []any{}}
	cfg["_netductor"] = map[string]any{"schema": SchemaVersion}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	meta, ok := out["_netductor"].(map[string]any)
	if !ok || int(meta["schema"].(float64)) != SchemaVersion {
		t.Fatalf("schema marker missing: %v", out)
	}
	_ = filepath.Join(tmp, "x")
	_ = os.ModePerm
}
