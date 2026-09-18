package main

import (
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func registerNVRAPI(mux *http.ServeMux) {
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
				ID:        str(body["id"]),
				SiteID:    str(body["site_id"]),
				Name:      str(body["name"]),
				MAC:       str(body["mac"]),
				LANIP:     str(body["lan_ip"]),
				RTSPUser:  str(body["rtsp_user"]),
				RTSPPath:  str(body["rtsp_path"]),
				SecretRef: str(body["secret_ref"]),
				Enabled:   truthy(body["enabled"], true),
				Record:    truthy(body["record"], true),
				StreamSub: truthy(body["stream_sub"], false),
			}
			if v, ok := body["rtsp_port"].(float64); ok {
				c.RTSPPort = int(v)
			}
			if feat, ok := body["features"].(map[string]any); ok {
				c.Features = map[string]bool{}
				for k, v := range feat {
					c.Features[k] = truthy(v, false)
				}
			}
			pass := str(body["rtsp_password"])
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
			writeJSON(w, 200, map[string]any{"ok": true, "camera": out})
		default:
			writeJSON(w, 405, map[string]string{"error": "method"})
		}
	})

	mux.HandleFunc("/api/nvr/cameras/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		id := str(body["id"])
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "id required"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": nvr.DeleteCamera(id)})
	})

	// Enqueue dhcp_leases / wifi_clients / dhcp_static on edge device
	mux.HandleFunc("/api/nvr/site/leases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := str(body["device_id"])
		if did == "" {
			writeJSON(w, 400, map[string]string{"error": "device_id required"})
			return
		}
		id := edge.EnqueueCmd(did, "dhcp_leases", "")
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "enqueue failed (device not approved?)"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "cmd_id": id, "hint": "poll /api/edge/results"})
	})

	mux.HandleFunc("/api/nvr/site/wifi_clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		body := readJSON(r)
		did := str(body["device_id"])
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
		did := str(body["device_id"])
		mac := str(body["mac"])
		ip := str(body["ip"])
		name := str(body["name"])
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

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func truthy(v any, def bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true") || t == "yes"
	case float64:
		return t != 0
	case nil:
		return def
	default:
		return def
	}
}
