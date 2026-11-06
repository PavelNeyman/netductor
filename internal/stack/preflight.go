package stack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/fleet"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/upgrade"
)

// PreflightCheck is one gate before stack apply.
type PreflightCheck struct {
	ID       string `json:"id"`
	OK       bool   `json:"ok"`
	Critical bool   `json:"critical"`
	Detail   string `json:"detail"`
}

// PreflightReport aggregates apply readiness for tag.
type PreflightReport struct {
	Tag    string           `json:"tag"`
	OK     bool             `json:"ok"`
	Checks []PreflightCheck `json:"checks"`
}

// Preflight validates release existence, checksums, and basic fleet health.
// Critical failures always block Apply.
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

	// Release + SHA256SUMS (node + tg required). If GitHub's Release CDN is
	// unreachable, accept the local release store only when the manifest and
	// both binaries are present and independently hash-verified.
	platform := ndupdate.PlatformFromRuntime()
	nodeAsset := ndupdate.AssetName(ndupdate.ComponentNode, platform)
	tgAsset := ndupdate.AssetName(ndupdate.ComponentTG, platform)
	if nodeAsset == "" || tgAsset == "" {
		add(PreflightCheck{ID: "release_sums", OK: false, Critical: true, Detail: fmt.Sprintf("unsupported host platform %s/%s", platform.GOOS, platform.GOARCH)})
		return rep
	}
	need := []string{nodeAsset, tgAsset}
	sums, err := ndupdate.FetchSHA256SUMS(tag)
	if err != nil {
		if localErr := ndupdate.VerifyLocalReleaseAssets(tag, need...); localErr == nil {
			add(PreflightCheck{ID: "release_sums", OK: true, Critical: true, Detail: fmt.Sprintf("%s local mirror: SHA256SUMS and required assets verified", tag)})
		} else {
			mirror := filepath.Join(ndupdate.LocalReleasesDir(), tag)
			add(PreflightCheck{ID: "release_sums", OK: false, Critical: true, Detail: fmt.Sprintf(
				"SHA256SUMS unavailable remotely (%v); local mirror unusable (%v). Publish GitHub Release %s with SHA256SUMS + %s + %s, or place the same files under %s/",
				err, localErr, tag, nodeAsset, tgAsset, mirror)})
		}
	} else {
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

	// CF-15 phase A: if target ships upgrade.json with min_from, refuse when local is older.
	{
		local := upgrade.LocalVersion()
		if local == "" {
			if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "VERSION")); err == nil {
				local = strings.TrimSpace(string(b))
			}
		}
		meta, hasMeta, merr := upgrade.FetchMeta(tag)
		if merr != nil {
			add(PreflightCheck{ID: "upgrade_floor", OK: false, Critical: false, Detail: "upgrade.json: " + merr.Error()})
		} else if hasMeta && strings.TrimSpace(meta.MinFrom) != "" {
			if err := upgrade.CheckLocalAllowed(local, meta.MinFrom); err != nil {
				add(PreflightCheck{ID: "upgrade_floor", OK: false, Critical: true, Detail: err.Error()})
			} else {
				add(PreflightCheck{ID: "upgrade_floor", OK: true, Critical: true, Detail: fmt.Sprintf("local %s >= min_from %s", local, meta.MinFrom)})
			}
		} else {
			add(PreflightCheck{ID: "upgrade_floor", OK: true, Critical: false, Detail: "no upgrade.json min_from for tag (skip-version OK)"})
		}
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
