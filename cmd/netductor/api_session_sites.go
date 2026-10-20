package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/nvr"
	"github.com/PavelNeyman/netductor/internal/sites"
)

func registerSessionSites(mux *http.ServeMux) {
	mux.HandleFunc("/api/sites", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			list, err := sites.List()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, 200, map[string]any{"sites": list})
			return
		}
		if r.Method == http.MethodPost {
			var s sites.Site
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			out, err := sites.Upsert(s)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, 200, out)
			return
		}
		http.Error(w, "method", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/api/sites/rsc", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := r.URL.Query().Get("id")
		rsc, err := sites.RSCForSite(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, 200, map[string]any{"id": id, "rsc": rsc})
	})

	mux.HandleFunc("/api/sites/push-rsc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			if r.Method != http.MethodPost {
				http.Error(w, "method", http.StatusMethodNotAllowed)
			}
			return
		}
		var body struct {
			SiteID   string `json:"site_id"`
			Host     string `json:"host"`
			User     string `json:"user"`
			Password string `json:"password"`
			Port     int    `json:"port"`
			RpiLAN   string `json:"rpi_lan"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		rsc, err := sites.RSCForSiteWithGateway(body.SiteID, body.RpiLAN)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := mikrotik.PushRSC(body.Host, body.User, body.Password, nil, rsc, body.Port); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/sites/rooms", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			siteID := strings.TrimSpace(r.URL.Query().Get("site"))
			if siteID == "" {
				writeJSON(w, 400, map[string]string{"error": "site query required"})
				return
			}
			list, err := sites.ListRooms(siteID)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"site": siteID, "rooms": list})
		case http.MethodPost:
			var body struct {
				ID          string   `json:"id"`
				SiteID      string   `json:"site_id"`
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Tags        []string `json:"tags"`
				CameraIDs   []string `json:"camera_ids"`
				EdgeIDs     []string `json:"edge_ids"`
				Notes       string   `json:"notes"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			out, err := sites.UpsertRoom(sites.Room{
				ID: body.ID, SiteID: body.SiteID, Name: body.Name, Description: body.Description,
				Tags: body.Tags, CameraIDs: body.CameraIDs, EdgeIDs: body.EdgeIDs, Notes: body.Notes,
			})
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			var unknown []string
			if len(body.CameraIDs) > 0 {
				known := map[string]bool{}
				for _, c := range nvr.ListCameras() {
					known[c.ID] = true
					if c.Name != "" {
						known[c.Name] = true
					}
				}
				for _, id := range body.CameraIDs {
					if id != "" && !known[id] {
						unknown = append(unknown, id)
					}
				}
			}
			writeJSON(w, 200, map[string]any{"ok": true, "room": out, "unknown_cameras": unknown})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/sites/rooms/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !requireSession(w, r) {
			return
		}
		var body struct {
			SiteID string `json:"site_id"`
			ID     string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if err := sites.DeleteRoom(body.SiteID, body.ID); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sites/rooms/photo", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			siteID := strings.TrimSpace(r.URL.Query().Get("site"))
			roomID := strings.TrimSpace(r.URL.Query().Get("id"))
			idx := 0
			fmt.Sscanf(r.URL.Query().Get("idx"), "%d", &idx)
			b, err := sites.GetPhoto(siteID, roomID, idx)
			if err != nil {
				http.Error(w, err.Error(), 404)
				return
			}
			ct := "image/jpeg"
			if len(b) >= 2 && b[0] == 0x89 && b[1] == 'P' {
				ct = "image/png"
			}
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Cache-Control", "private, max-age=60")
			_, _ = w.Write(b)
		case http.MethodPost:
			// JSON: {site_id,id,data_base64} or multipart file
			ct := r.Header.Get("Content-Type")
			if strings.HasPrefix(ct, "multipart/") {
				if err := r.ParseMultipartForm(3 << 20); err != nil {
					writeJSON(w, 400, map[string]string{"error": err.Error()})
					return
				}
				siteID := r.FormValue("site_id")
				roomID := r.FormValue("id")
				f, hdr, err := r.FormFile("file")
				if err != nil {
					writeJSON(w, 400, map[string]string{"error": "file required"})
					return
				}
				defer f.Close()
				data, err := io.ReadAll(io.LimitReader(f, 2<<20+1))
				if err != nil {
					writeJSON(w, 400, map[string]string{"error": err.Error()})
					return
				}
				idx, err := sites.AddPhoto(siteID, roomID, data, hdr.Header.Get("Content-Type"))
				if err != nil {
					writeJSON(w, 400, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, 200, map[string]any{"ok": true, "index": idx})
				return
			}
			var body struct {
				SiteID      string `json:"site_id"`
				ID          string `json:"id"`
				DataBase64  string `json:"data_base64"`
				ContentType string `json:"content_type"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			data, err := base64.StdEncoding.DecodeString(body.DataBase64)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": "bad base64"})
				return
			}
			idx, err := sites.AddPhoto(body.SiteID, body.ID, data, body.ContentType)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "index": idx})
		case http.MethodDelete:
			siteID := strings.TrimSpace(r.URL.Query().Get("site"))
			roomID := strings.TrimSpace(r.URL.Query().Get("id"))
			idx := 0
			fmt.Sscanf(r.URL.Query().Get("idx"), "%d", &idx)
			if err := sites.DeletePhoto(siteID, roomID, idx); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	})

}
