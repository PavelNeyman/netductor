package main

import (
	"encoding/json"
	"net/http"

	"github.com/PavelNeyman/netductor/internal/integrity"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func registerSecurityAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/security/ssh-keys", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		keys := integrity.LiveKeysPrimary()
		secs := []map[string]any{}
		for _, d := range secondary.List() {
			sk := integrity.LiveKeysSecondaryCached(d.ID)
			secs = append(secs, map[string]any{
				"id": d.ID, "name": d.Name, "keys": sk, "cached": len(sk) > 0,
			})
		}
		st := integrity.ReadAllowlistState()
		writeJSON(w, 200, map[string]any{
			"primary": keys,
			"secondaries": secs,
			"allowlist_lines": len(integrity.ReadAllowlistLines()),
			"enforce": st.Enforce,
			"updated_at": st.UpdatedAt,
		})
	})
	mux.HandleFunc("/api/security/ssh-keys/probe-secondary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var in struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		n := 0
		for _, d := range secondary.List() {
			if in.ID != "" && d.ID != in.ID {
				continue
			}
			_ = secondary.EnqueueCmd(d.ID, "ssh_keys")
			n++
		}
		writeJSON(w, 200, map[string]any{"ok": true, "queued": n})
	})
	mux.HandleFunc("/api/security/ssh-allowlist", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			st := integrity.ReadAllowlistState()
			writeJSON(w, 200, map[string]any{
				"lines": integrity.ReadAllowlistLines(),
				"enforce": st.Enforce,
				"updated_at": st.UpdatedAt,
				"updated_by": st.UpdatedBy,
			})
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		var in struct {
			Fingerprints []string `json:"fingerprints"`
			Enforce      bool     `json:"enforce"`
			Merge        bool     `json:"merge"`
			PushSecondary bool    `json:"push_secondary"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		cands := integrity.LiveKeysPrimary()
		for _, d := range secondary.List() {
			cands = append(cands, integrity.LiveKeysSecondaryCached(d.ID)...)
		}
		var err error
		if in.Merge {
			err = integrity.MergeAllowlist(in.Fingerprints, cands, "api")
		} else {
			err = integrity.SetAllowlistByFingerprints(in.Fingerprints, cands, in.Enforce, "api")
		}
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		pushed := 0
		if in.PushSecondary || in.Enforce {
			b64, err := integrity.AllowlistB64()
			if err == nil {
				for _, d := range secondary.List() {
					_ = secondary.EnqueueCmd(d.ID, "ssh_allowlist:"+b64)
					pushed++
				}
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true, "enforce": in.Enforce, "secondary_queued": pushed})
	})
}
