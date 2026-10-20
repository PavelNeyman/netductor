package main

import (
	"encoding/json"
	"net/http"

	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerSessionSNI(mux *http.ServeMux) {
	mux.HandleFunc("/api/sni-presets", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		_ = vpn.EnsureSNIPresetsFile()
		writeJSON(w, 200, map[string]any{"presets": vpn.ListSNIPresets(), "active": vpn.ActiveSNI()})
	})
	mux.HandleFunc("/api/sni", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			writeJSON(w, 200, map[string]any{"active": vpn.ActiveSNI(), "presets": vpn.ListSNIPresets()})
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			SNI  string `json:"sni"`
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sniName := body.SNI
		if sniName == "" {
			sniName = body.Name
		}
		for _, p := range vpn.ListSNIPresets() {
			if p.Name == sniName {
				sniName = p.SNI
				break
			}
		}
		if err := vpn.SetActiveSNI(sniName); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		ver := secondary.BumpConfigVer()
		writeJSON(w, 200, map[string]any{"ok": true, "active": vpn.ActiveSNI(), "secondary_config_ver": ver})
	})

}
