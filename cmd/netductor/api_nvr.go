package main

import (
	"net/http"
	"strings"
)

func registerNVRAPI(mux *http.ServeMux) {
	registerNVRCameras(mux)
	registerNVRConfig(mux)
	registerNVREvents(mux)
	registerNVRLive(mux)
}

func nvrStr(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func nvrTruthy(v any, def bool) bool {
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
