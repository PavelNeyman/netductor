package main

import (
	"net/http"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func registerNVRCameras(mux *http.ServeMux) {
	mux.HandleFunc("/api/nvr/cameras", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, map[string]any{"cameras": nvr.ListCameras()})
		case http.MethodPost:
			body := readJSON(r)
			c := nvr.Camera{
				ID:        nvrStr(body["id"]),
				SiteID:    nvrStr(body["site_id"]),
				Name:      nvrStr(body["name"]),
				MAC:       nvrStr(body["mac"]),
				LANIP:     nvrStr(body["lan_ip"]),
				RTSPUser:  nvrStr(body["rtsp_user"]),
				RTSPPath:  nvrStr(body["rtsp_path"]),
				SecretRef: nvrStr(body["secret_ref"]),
				Enabled:   nvrTruthy(body["enabled"], true),
				Record:    nvrTruthy(body["record"], true),
				StreamSub: nvrTruthy(body["stream_sub"], false),
			}
			if v, ok := body["rtsp_port"].(float64); ok {
				c.RTSPPort = int(v)
			}
			if feat, ok := body["features"].(map[string]any); ok {
				c.Features = map[string]bool{}
				for k, v := range feat {
					c.Features[k] = nvrTruthy(v, false)
				}
			}
			pass := nvrStr(body["rtsp_password"])
			out, err := nvr.UpsertCamera(c)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			if pass != "" {
				ref := out.SecretRef
				if ref == "" {
					ref = out.ID
					out.SecretRef = ref
					out, _ = nvr.UpsertCamera(out)
				}
				_ = nvr.SetSecret(ref, pass)
			}
			// TP-Link cloud password for TPAP control + go2rtc talk (tapo://), not RTSP.
			if cloud := nvrStr(body["cloud_password"]); cloud != "" {
				ref := out.SecretRef
				if ref == "" {
					ref = out.ID
				}
				_ = nvr.SetSecret(ref+"_cloud", cloud)
			}
			writeJSON(w, 200, map[string]any{"ok": true, "camera": out})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/nvr/cameras/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		id := nvrStr(body["id"])
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "id required"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": nvr.DeleteCamera(id)})
	})

	mux.HandleFunc("/api/nvr/site/leases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := nvrStr(body["device_id"])
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id required"})
			return
		}
		id := edge.EnqueueCmd(did, "dhcp_leases", "")
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "enqueue failed (device not approved?)"})
			return
		}
		// Default: wait for agent result and parse leases for UI autofill.
		// Body wait=false → only cmd_id (async).
		wait := true
		if v, ok := body["wait"]; ok {
			wait = nvrTruthy(v, true)
		}
		if !wait {
			writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": id, "hint": "poll /api/edge/results"})
			return
		}
		timeout := 90 * time.Second
		if v, ok := body["timeout_sec"].(float64); ok && v > 0 {
			timeout = time.Duration(v) * time.Second
		}
		res, err := edge.WaitCmdResult(id, timeout)
		if err != nil {
			writeJSON(w, 200, map[string]any{"ok": false, "cmd_id": id, "error": err.Error(), "hint": "poll /api/edge/results"})
			return
		}
		raw := ""
		if s, ok := res["result"].(string); ok {
			raw = s
		} else if s, ok := res["output"].(string); ok {
			raw = s
		}
		leases := nvr.ParseLeasesResult(raw)
		writeJSON(w, 200, map[string]any{
			"ok": true, "cmd_id": id, "leases": leases, "raw": raw, "result": res,
		})
	})

	mux.HandleFunc("/api/nvr/site/wifi_clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := nvrStr(body["device_id"])
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id required"})
			return
		}
		id := edge.EnqueueCmd(did, "wifi_clients", "")
		writeJSON(w, 200, map[string]any{"ok": id != "", "cmd_id": id})
	})

	mux.HandleFunc("/api/nvr/site/dhcp_static", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := nvrStr(body["device_id"])
		mac := nvrStr(body["mac"])
		ip := nvrStr(body["ip"])
		name := nvrStr(body["name"])
		if did == "" || mac == "" || ip == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id, mac, ip required"})
			return
		}
		arg := "mac=" + mac + "|ip=" + ip
		if name != "" {
			arg += "|name=" + name
		}
		id := edge.EnqueueCmd(did, "dhcp_static", arg)
		writeJSON(w, 200, map[string]any{"ok": id != "", "cmd_id": id})
	})

}
