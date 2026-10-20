package main

import (
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nodes"
)

func registerEdgeAgent(mux *http.ServeMux) {
	mux.HandleFunc("/api/edge/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		tokenDID := edge.DeviceIDFromAuth(auth)
		if tokenDID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		payload := readJSON(r)
		did, _ := payload["device_id"].(string)
		if did == "" {
			did = tokenDID
			payload["device_id"] = did
		}
		if did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		edge.Heartbeat(payload)
		if raw, ok := payload["facts"]; ok {
			if fm, ok := raw.(map[string]any); ok {
				if sm, ok := fm["storage"].(map[string]any); ok {
					var st edge.StorageFacts
					if b, ok := sm["has_usb_disk"].(bool); ok {
						st.HasUSBDisk = b
					}
					if b, ok := sm["extroot_active"].(bool); ok {
						st.ExtrootActive = b
					}
					if b, ok := sm["expect_usb"].(bool); ok {
						st.ExpectUSB = b
					}
					if disks, ok := sm["usb_disks"].([]any); ok && len(disks) > 0 {
						st.HasUSBDisk = true
					}
					edge.EvaluateStorageAlerts(did, st)
				}
			}
		}
		if ser, _ := payload["cert_serial"].(string); ser != "" {
			_ = mtls.ConfirmRotate(did, ser)
		}
		hn, _ := payload["hostname"].(string)
		ip, _ := payload["wan_ip"].(string)
		if ip == "" {
			ip, _ = payload["public_ip"].(string)
		}
		kind, _ := payload["kind"].(string)
		if kind == "" {
			kind = "openwrt"
		}
		role, _ := payload["role"].(string)
		if role == "" {
			role = "edge"
		}
		resp := map[string]any{"ok": true}
		if did != "" {
			n, err := nodes.UpsertFromDevice(nodes.Node{
				ID: did, Hostname: hn, Role: role, Kind: kind, PublicIP: ip, Status: "online",
			})
			if err == nil && n.DesiredHN != "" {
				resp["desired_hostname"] = n.DesiredHN
			}
		}
		writeJSON(w, 200, resp)
	})
	mux.HandleFunc("/api/edge/commands", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		auth := r.Header.Get("Authorization")
		did := edge.DeviceIDFromAuth(auth)
		if did == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		q := r.URL.Query().Get("device_id")
		if q != "" && q != did {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		writeJSON(w, 200, map[string]any{"commands": edge.PollCommands(did)})
	})

	mux.HandleFunc("/api/edge/mtls/material", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		did := edge.DeviceIDFromAuth(r.Header.Get("Authorization"))
		if did == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		ca, cert, key, err := mtls.ReadMaterial(did)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		info, _ := mtls.ReadClientInfo(did)
		writeJSON(w, 200, map[string]any{
			"ca_pem": string(ca), "cert_pem": string(cert), "key_pem": string(key),
			"serial": info.Serial, "node_id": did,
		})
	})

	mux.HandleFunc("/api/edge/rsc", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if edge.DeviceIDFromAuth(auth) == "" {
			if !requireSession(w, r) {
				return
			}
		}
		name := r.URL.Query().Get("name")
		if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
			writeJSON(w, 400, map[string]string{"error": "name"})
			return
		}
		b, err := edge.ReadRSC(name)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/edge/apply_rsc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did, _ := body["device_id"].(string)
		script, _ := body["script"].(string)
		if did == "" || script == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id and script required"})
			return
		}
		name, err := edge.SaveRSC(did, script)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		// RSC is MikroTik RouterOS — OpenWrt agent cannot apply it. File saved for operator download / MT push.
		cid := edge.EnqueueCmd(did, "apply_rsc", name)
		if cid == "" {
			writeJSON(w, 200, map[string]any{
				"ok": true, "cmd_id": "", "rsc": name,
				"note": "RSC saved; not queued to edge agent (use MikroTik sites push / PushRSC). OpenWrt agents deny apply_rsc.",
			})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": cid, "rsc": name})
	})
	mux.HandleFunc("/api/edge/results", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		writeJSON(w, 200, map[string]any{"results": edge.ListResults()})
	})
	mux.HandleFunc("/api/edge/cmd_result", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
			return
		}
		tokenDID := edge.DeviceIDFromAuth(r.Header.Get("Authorization"))
		if tokenDID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		body := readJSON(r)
		if did, _ := body["device_id"].(string); did != "" && did != tokenDID {
			writeJSON(w, 403, map[string]string{"error": "device_mismatch"})
			return
		}
		body["device_id"] = tokenDID
		edge.CmdResult(body)
		writeJSON(w, 200, map[string]bool{"ok": true})
	})
}
