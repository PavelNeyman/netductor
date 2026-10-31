package aptguard

import (
	"fmt"
	"os"
	"time"
)

// RunLocalScan loads policy, scans root filesystem, saves report, returns it.
func RunLocalScan(stateRoot, serverID, fsRoot string) (Report, Report, error) {
	var prev Report
	if p, err := LoadLatest(stateRoot, serverID); err == nil {
		prev = p
	}
	policy, err := EnsurePolicy(stateRoot)
	if err != nil {
		return Report{}, prev, err
	}
	if serverID == "" {
		serverID, _ = os.Hostname()
		if serverID == "" {
			serverID = "local"
		}
	}
	rep, err := ScanHost(serverID, policy, fsRoot)
	if err != nil {
		return Report{}, prev, err
	}
	if err := SaveReport(stateRoot, rep); err != nil {
		return rep, prev, fmt.Errorf("save report: %w", err)
	}
	return rep, prev, nil
}

// DoctorSummary formats a one-line status for doctor.
func DoctorSummary(stateRoot string, maxAge time.Duration) (status string, detail string, severity Severity) {
	host, _ := os.Hostname()
	if host == "" {
		host = "local"
	}
	rep, err := LoadLatest(stateRoot, host)
	if err != nil {
		// try "local"
		rep, err = LoadLatest(stateRoot, "local")
		if err != nil {
			return "missing", "no aptguard report yet (run: netductor aptguard scan)", SevInfo
		}
	}
	if Stale(rep, maxAge) {
		return "stale", fmt.Sprintf("last check %s (>%s)", rep.CheckedAt.UTC().Format(time.RFC3339), maxAge), SevWarn
	}
	nFail, nWarn := 0, 0
	for _, f := range rep.Findings {
		if f.Severity == SevFail {
			nFail++
		} else if f.Severity == SevWarn {
			nWarn++
		}
	}
	detail = fmt.Sprintf("status=%s policy_v=%d fail=%d warn=%d checked=%s",
		rep.Status, rep.PolicyVersion, nFail, nWarn, rep.CheckedAt.UTC().Format(time.RFC3339))
	return string(rep.Status), detail, rep.Status
}
