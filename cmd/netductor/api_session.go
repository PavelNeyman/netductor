package main

import (
	"net/http"
	"strings"
)

var allowedSystemdRestart = map[string]bool{
	"netductor-api":             true,
	"netductor-telegram-bot":    true,
	"netductor-secondary-agent": true,
	"netductor-agent":           true,
	"netductor":                 true,
}

func allowSystemdUnit(unit string) bool {
	unit = strings.TrimSpace(unit)
	if unit == "" || strings.ContainsAny(unit, " \t\r\n;/|&") {
		return false
	}
	if allowedSystemdRestart[unit] {
		return true
	}
	// allow netductor-*.service style
	if strings.HasPrefix(unit, "netductor-") && !strings.Contains(unit, "..") {
		return true
	}
	return false
}

func registerSessionAPI(mux *http.ServeMux) {
	registerSessionNodes(mux)
	registerSessionSNI(mux)
	registerSessionSites(mux)
	registerSessionStatus(mux)
	registerSessionMTLS(mux)
}
