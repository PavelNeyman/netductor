package stack

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

// PreflightCheck is one gate before stack apply.
type PreflightCheck struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Critical bool  `json:"critical"`
	Detail  string `json:"detail"`
}

// PreflightReport aggregates apply readiness for tag.
type PreflightReport struct {
	Tag    string           `json:"tag"`
	OK     bool             `json:"ok"`
	Checks []PreflightCheck `json:"checks"`
}

// Preflight validates release exists with checksums and basic fleet health.
// Critical failures block apply unless NETDUCTOR_STACK_SKIP_PREFLIGHT=1.
func Preflight(tag string) PreflightReport {
	tag = strings.TrimSpace(tag)
	if tag != "" && !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	rep := PreflightReport{Tag: tag, OK: true}
	add := func(c PreflightCheck) {
		rep.Checks = append(rep.Checks, c)
		if c.Critical && !c.OK {
			rep.OK = false
		}
	}

	if tag == "" {
		add(PreflightCheck{ID: "tag", OK: false, Critical: true, Detail: "empty tag"})
		return rep
	}

	// Release + SHA256SUMS (node + tg required)
	sums, err := ndupdate.FetchSHA256SUMS(tag)
	if err != nil {
		add(PreflightCheck{ID: "release_sums", OK: false, Critical: true, Detail: fmt.Sprintf("SHA256SUMS: %v", err)})
	} else {
		need := []string{"netductor-linux-amd64", "netductor-tg-linux-amd64"}
		missing := []string{}
		for _, n := range need {
			if sums[n] == "" {
				missing = append(missing, n)
			}
		}
		if len(missing) > 0 {
			add(PreflightCheck{ID: "release_sums", OK: false, Critical: true, Detail: "missing checksums: " + strings.Join(missing, ",")})
		} else {
			add(PreflightCheck{ID: "release_sums", OK: true, Critical: true, Detail: fmt.Sprintf("%s has SHA256SUMS (%d entries)", tag, len(sums))})
		}
	}

	if ok, why := IsPinned(); ok {
		add(PreflightCheck{ID: "pin", OK: false, Critical: true, Detail: "pinned: " + why})
	} else {
		add(PreflightCheck{ID: "pin", OK: true, Critical: true, Detail: "not pinned"})
	}

	secs := secondary.List()
	if len(secs) > 0 {
		off := 0
		for _, s := range secs {
			if !secondary.Online(s, 2*time.Minute) {
				off++
			}
		}
		if off == len(secs) {
			add(PreflightCheck{ID: "secondary", OK: false, Critical: false, Detail: "all secondaries offline (apply still possible)"})
		} else if off > 0 {
			add(PreflightCheck{ID: "secondary", OK: true, Critical: false, Detail: fmt.Sprintf("%d/%d secondary offline", off, len(secs))})
		} else {
			add(PreflightCheck{ID: "secondary", OK: true, Critical: false, Detail: "secondaries online"})
		}
	} else {
		add(PreflightCheck{ID: "secondary", OK: true, Critical: false, Detail: "no secondaries enrolled"})
	}

	d := fleet.Build()
	if d.BackupAgeHours != nil && *d.BackupAgeHours > 72 {
		add(PreflightCheck{ID: "backup_age", OK: false, Critical: false, Detail: fmt.Sprintf("backup age %.0fh", *d.BackupAgeHours)})
	} else if d.BackupAgeHours != nil {
		add(PreflightCheck{ID: "backup_age", OK: true, Critical: false, Detail: fmt.Sprintf("backup age %.1fh", *d.BackupAgeHours)})
	} else {
		add(PreflightCheck{ID: "backup_age", OK: false, Critical: false, Detail: "no backup found under " + paths.StateDir() + "/backups"})
	}

	return rep
}

// FormatPreflight CLI/TG text.
func FormatPreflight(r PreflightReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("preflight %s ok=%v\n", r.Tag, r.OK))
	for _, c := range r.Checks {
		mark := "OK"
		if !c.OK {
			if c.Critical {
				mark = "FAIL"
			} else {
				mark = "WARN"
			}
		}
		b.WriteString(fmt.Sprintf("  %s  %s — %s\n", mark, c.ID, c.Detail))
	}
	return b.String()
}
