package main

import "testing"

func TestAPIPathExemptAgentPlanes(t *testing.T) {
	for _, p := range []string{
		"/health", "/api/bot-status", "/api/session/issue",
		"/api/edge/heartbeat", "/api/edge/storage/backup-push", "/api/edge/backups",
		"/api/build-host/heartbeat", "/api/build-host/jobs/claim",
		"/api/build-host/jobs/x/work.tgz", "/api/secondary/agent/heartbeat",
		"/api/nvr/clip", "/api/nvr/ingest",
	} {
		if !apiPathExemptFromSession(p) {
			t.Fatalf("want exempt %s", p)
		}
	}
	for _, p := range []string{
		"/api/build-host/status", "/api/build-host/recovery-arm",
		"/api/vpn/users", "/api/metrics", "/api/edge/list",
	} {
		if apiPathExemptFromSession(p) {
			t.Fatalf("must NOT exempt %s", p)
		}
	}
}
