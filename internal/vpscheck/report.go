package vpscheck

import (
	"encoding/json"
	"strings"
	"time"
)

// Profile selects which checks matter most.
type Profile string

const (
	ProfileGeneric   Profile = "generic"
	ProfilePrimary   Profile = "primary"   // abroad exit / control
	ProfileSecondary Profile = "secondary" // RU entry
)

// Status values for a single check.
const (
	StatusOK      = "ok"
	StatusWarn    = "warn"
	StatusFail    = "fail"
	StatusSkip    = "skip"
	StatusInfo    = "info"
)

// Result is one named probe.
type Result struct {
	ID         string `json:"id"`
	TitleEN    string `json:"title_en"`
	TitleRU    string `json:"title_ru"`
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// Report is the full pre-check scorecard.
type Report struct {
	At        time.Time `json:"at"`
	Host      string    `json:"host,omitempty"`
	Profile   Profile   `json:"profile"`
	PublicIP  string    `json:"public_ip,omitempty"`
	Region    string    `json:"region,omitempty"`
	Results   []Result  `json:"results"`
	OK        int       `json:"ok"`
	Warn      int       `json:"warn"`
	Fail      int       `json:"fail"`
	SummaryEN string    `json:"summary_en"`
	SummaryRU string    `json:"summary_ru"`
}

func (r *Report) tally() {
	r.OK, r.Warn, r.Fail = 0, 0, 0
	for _, x := range r.Results {
		switch x.Status {
		case StatusOK:
			r.OK++
		case StatusWarn:
			r.Warn++
		case StatusFail:
			r.Fail++
		}
	}
	switch {
	case r.Fail > 0:
		r.SummaryEN = "FAIL — do not use for production without fixing fails"
		r.SummaryRU = "FAIL — не брать в прод, пока не закрыты fail"
	case r.Warn > 0:
		r.SummaryEN = "WARN — usable with caveats"
		r.SummaryRU = "WARN — можно, но с оговорками"
	default:
		r.SummaryEN = "OK — no hard failures"
		r.SummaryRU = "OK — жёстких провалов нет"
	}
}

func (r Report) JSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

func (r Report) Text() string {
	var b strings.Builder
	b.WriteString("=== netductor vps-check ===\n")
	b.WriteString("profile=" + string(r.Profile) + " host=" + r.Host + "\n")
	if r.PublicIP != "" {
		b.WriteString("public_ip=" + r.PublicIP + " region=" + r.Region + "\n")
	}
	for _, x := range r.Results {
		b.WriteString(x.Status + "\t" + x.ID + "\t" + x.Detail + "\n")
	}
	b.WriteString("summary: " + r.SummaryEN + " (ok=" + itoa(r.OK) + " warn=" + itoa(r.Warn) + " fail=" + itoa(r.Fail) + ")\n")
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [16]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	return string(d[i:])
}

func ParseProfile(s string) Profile {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "primary", "p", "abroad", "core":
		return ProfilePrimary
	case "secondary", "s", "ru", "relay":
		return ProfileSecondary
	default:
		return ProfileGeneric
	}
}
