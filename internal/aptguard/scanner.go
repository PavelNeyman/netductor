package aptguard

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Severity string

const (
	SevOK   Severity = "OK"
	SevWarn Severity = "WARN"
	SevFail Severity = "FAIL"
	SevInfo Severity = "INFO"
)

type Finding struct {
	Code       string   `json:"code"`
	Severity   Severity `json:"severity"`
	SourceFile string   `json:"source_file,omitempty"`
	Message    string   `json:"message"`
}

type SourceEntry struct {
	File       string   `json:"file"`
	Enabled    bool     `json:"enabled"`
	Type       string   `json:"type"` // deb|deb-src
	URIs       []string `json:"uris"`
	Suites     []string `json:"suites"`
	Components []string `json:"components"`
	SignedBy   string   `json:"signed_by,omitempty"`
	Raw        string   `json:"raw,omitempty"`
}

type Report struct {
	ServerID      string        `json:"server_id"`
	Module        string        `json:"module"`
	Status        Severity      `json:"status"`
	CheckedAt     time.Time     `json:"checked_at"`
	PolicyVersion int           `json:"policy_version"`
	Sources       []SourceEntry `json:"sources"`
	Findings      []Finding     `json:"findings"`
}

// ScanHost reads local APT sources (read-only) and compares to policy.
func ScanHost(serverID string, policy Policy, root string) (Report, error) {
	if root == "" {
		root = "/"
	}
	if err := policy.Validate(); err != nil {
		return Report{}, err
	}
	rep := Report{
		ServerID:      serverID,
		Module:        "apt-repository-guard",
		Status:        SevOK,
		CheckedAt:     time.Now().UTC(),
		PolicyVersion: policy.Version,
	}
	sources, parseFindings, err := collectSources(root)
	if err != nil {
		return Report{}, err
	}
	rep.Sources = sources
	rep.Findings = append(rep.Findings, parseFindings...)

	for _, s := range sources {
		if !s.Enabled {
			rep.Findings = append(rep.Findings, Finding{
				Code: "SOURCE_DISABLED", Severity: SevInfo, SourceFile: s.File,
				Message: "source present but disabled",
			})
			continue
		}
		match := matchPolicy(policy, s)
		switch match {
		case "ok":
			// fine
		case "unknown":
			rep.Findings = append(rep.Findings, Finding{
				Code: "UNKNOWN_REPOSITORY", Severity: SevWarn, SourceFile: s.File,
				Message: fmt.Sprintf("repository not in policy: %v", redactURIs(s.URIs)),
			})
		case "forbidden":
			rep.Findings = append(rep.Findings, Finding{
				Code: "FORBIDDEN_REPOSITORY", Severity: SevFail, SourceFile: s.File,
				Message: fmt.Sprintf("repository disabled in policy: %v", redactURIs(s.URIs)),
			})
		case "mismatch":
			rep.Findings = append(rep.Findings, Finding{
				Code: "POLICY_MISMATCH", Severity: SevFail, SourceFile: s.File,
				Message: fmt.Sprintf("URI matches policy but suite/components/type/signed-by differ: %v suites=%v", redactURIs(s.URIs), s.Suites),
			})
		}
	}

	// Required repos present?
	for _, r := range policy.Repos {
		if !r.Required || !r.Enabled {
			continue
		}
		if !requiredPresent(sources, r) {
			rep.Findings = append(rep.Findings, Finding{
				Code: "MISSING_REQUIRED", Severity: SevFail,
				Message: fmt.Sprintf("required repository %q not found enabled", r.ID),
			})
		}
	}

	rep.Status = aggregate(rep.Findings)
	return rep, nil
}

func aggregate(fs []Finding) Severity {
	st := SevOK
	for _, f := range fs {
		if f.Severity == SevFail {
			return SevFail
		}
		if f.Severity == SevWarn {
			st = SevWarn
		}
	}
	return st
}

func matchPolicy(p Policy, s SourceEntry) string {
	for _, r := range p.Repos {
		if !uriOverlap(r.URIs, s.URIs) {
			continue
		}
		if !r.Enabled {
			return "forbidden"
		}
		// type: policy assumes deb; reject deb-src as mismatch for required enforcement
		if s.Type == "deb-src" {
			return "mismatch"
		}
		if len(r.Suites) > 0 && len(s.Suites) > 0 {
			okSuite := false
			for _, want := range r.Suites {
				for _, got := range s.Suites {
					if want == got {
						okSuite = true
						break
					}
				}
			}
			if !okSuite {
				return "mismatch"
			}
		}
		if len(r.Components) > 0 && len(s.Components) > 0 {
			// at least one component overlap
			okComp := false
			for _, want := range r.Components {
				for _, got := range s.Components {
					if want == got {
						okComp = true
						break
					}
				}
			}
			if !okComp {
				return "mismatch"
			}
		}
		if len(r.SignedBy) > 0 && s.SignedBy != "" {
			okSig := false
			for _, sb := range r.SignedBy {
				if sb == s.SignedBy {
					okSig = true
					break
				}
			}
			if !okSig {
				return "mismatch"
			}
		}
		return "ok"
	}
	return "unknown"
}

func uriOverlap(a, b []string) bool {
	for _, x := range a {
		x = strings.TrimRight(strings.TrimSpace(x), "/")
		for _, y := range b {
			y = strings.TrimRight(strings.TrimSpace(y), "/")
			if strings.EqualFold(x, y) {
				return true
			}
		}
	}
	return false
}

func requiredPresent(sources []SourceEntry, r RepositorySpec) bool {
	for _, s := range sources {
		if !s.Enabled {
			continue
		}
		if uriOverlap(r.URIs, s.URIs) {
			return true
		}
	}
	return false
}

func collectSources(root string) ([]SourceEntry, []Finding, error) {
	var out []SourceEntry
	var findings []Finding
	main := filepath.Join(root, "etc/apt/sources.list")
	if st, err := os.Stat(main); err == nil && !st.IsDir() {
		ents, fs, err := parseListFile(main)
		if err != nil {
			findings = append(findings, Finding{Code: "PARSE_ERROR", Severity: SevFail, SourceFile: main, Message: err.Error()})
		} else {
			out = append(out, ents...)
			findings = append(findings, fs...)
		}
	}
	dir := filepath.Join(root, "etc/apt/sources.list.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, findings, nil
		}
		return out, findings, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(dir, name)
		switch {
		case strings.HasSuffix(name, ".list"):
			ents, fs, err := parseListFile(path)
			if err != nil {
				findings = append(findings, Finding{Code: "PARSE_ERROR", Severity: SevFail, SourceFile: path, Message: err.Error()})
				continue
			}
			out = append(out, ents...)
			findings = append(findings, fs...)
		case strings.HasSuffix(name, ".sources"):
			ents, fs, err := parseDeb822File(path)
			if err != nil {
				findings = append(findings, Finding{Code: "PARSE_ERROR", Severity: SevFail, SourceFile: path, Message: err.Error()})
				continue
			}
			out = append(out, ents...)
			findings = append(findings, fs...)
		}
	}
	return out, findings, nil
}

func parseListFile(path string) ([]SourceEntry, []Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var out []SourceEntry
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			// disabled one-line: # deb http://...
			trim := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if strings.HasPrefix(trim, "deb") {
				ent, ok := parseDebLine(path, trim, false)
				if ok {
					out = append(out, ent)
				}
			}
			continue
		}
		ent, ok := parseDebLine(path, line, true)
		if !ok {
			return out, []Finding{{Code: "PARSE_ERROR", Severity: SevFail, SourceFile: path,
				Message: fmt.Sprintf("line %d: unparseable", lineNo)}}, nil
		}
		out = append(out, ent)
	}
	return out, nil, sc.Err()
}

func parseDebLine(file, line string, enabled bool) (SourceEntry, bool) {
	// deb [options] uri suite component...
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return SourceEntry{}, false
	}
	typ := fields[0]
	if typ != "deb" && typ != "deb-src" {
		return SourceEntry{}, false
	}
	i := 1
	if strings.HasPrefix(fields[i], "[") {
		// skip options until ]
		for i < len(fields) && !strings.HasSuffix(fields[i], "]") {
			i++
		}
		if i >= len(fields) {
			return SourceEntry{}, false
		}
		i++
	}
	if i+1 >= len(fields) {
		return SourceEntry{}, false
	}
	uri := fields[i]
	suite := fields[i+1]
	comps := fields[i+2:]
	return SourceEntry{
		File: file, Enabled: enabled, Type: typ,
		URIs: []string{redactURI(uri)}, Suites: []string{suite}, Components: comps,
		Raw: "", // never store raw (may contain credentials)
	}, true
}

func parseDeb822File(path string) ([]SourceEntry, []Finding, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	blocks := strings.Split(string(b), "\n\n")
	var out []SourceEntry
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		ent := SourceEntry{File: path, Enabled: true, Type: "deb"}
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			switch strings.ToLower(k) {
			case "types":
				fields := strings.Fields(v)
				if len(fields) == 0 {
					continue
				}
				ent.Type = fields[0]
			case "uris", "uri":
				ent.URIs = strings.Fields(v)
			case "suites", "suite":
				ent.Suites = strings.Fields(v)
			case "components", "component":
				ent.Components = strings.Fields(v)
			case "signed-by":
				ent.SignedBy = v
			case "enabled":
				ent.Enabled = !strings.EqualFold(v, "no") && v != "false" && v != "0"
			}
		}
		if len(ent.URIs) == 0 {
			continue
		}
		out = append(out, ent)
	}
	return out, nil, nil
}

func redactURI(u string) string {
	// strip userinfo user:pass@
	if i := strings.Index(u, "://"); i >= 0 {
		rest := u[i+3:]
		if at := strings.Index(rest, "@"); at >= 0 {
			return u[:i+3] + "***@" + rest[at+1:]
		}
	}
	return u
}

func redactURIs(us []string) []string {
	out := make([]string, len(us))
	for i, u := range us {
		out[i] = redactURI(u)
	}
	return out
}
