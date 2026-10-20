package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/cli18n"
	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/version"
)

func detectRole() string {
	return firewall.DetectRole()
}

func activeUnit(unit string) bool {
	out, _ := exec.Command("systemctl", "is-active", unit).Output()
	return strings.TrimSpace(string(out)) == "active"
}

func sshdT(key string) string {
	out, err := exec.Command("sshd", "-T").Output()
	if err != nil {
		return ""
	}
	key = strings.ToLower(key)
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(strings.ToLower(line))
		if len(f) >= 2 && f[0] == key {
			return f[1]
		}
	}
	return ""
}

func listeningOnAll(port string) bool {
	for _, flag := range []string{"-tlnp", "-ulnp"} {
		out, err := exec.Command("ss", flag).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, ":"+port) {
				continue
			}
			if strings.Contains(line, "*:"+port) || strings.Contains(line, "0.0.0.0:"+port) || strings.Contains(line, "[::]:"+port) {
				return true
			}
		}
	}
	return false
}

func listeningLocalhost(port string) bool {
	for _, flag := range []string{"-tlnp", "-ulnp"} {
		out, err := exec.Command("ss", flag).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "127.0.0.1:"+port) || strings.Contains(line, "[::1]:"+port) {
				return true
			}
		}
	}
	return false
}

// DoctorCheck is one health line for CLI + JSON API.
type DoctorCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"` // ok | fail | warn | info
	Detail string `json:"detail,omitempty"`
}

// DoctorReport is structured doctor output.
type DoctorReport struct {
	OK      bool          `json:"ok"`
	Role    string        `json:"role"`
	Host    string        `json:"host"`
	Checks  []DoctorCheck `json:"checks"`
	Summary struct {
		OK   int `json:"ok"`
		Fail int `json:"fail"`
		Warn int `json:"warn"`
	} `json:"summary"`
}

// CollectDoctor runs the same checks as CLI doctor and returns JSON-friendly report.
func CollectDoctor() DoctorReport {
	doctorQuiet = true
	defer func() { doctorQuiet = false }()
	code := runDoctorNative()
	// runDoctorNative fills lastDoctorReport
	r := lastDoctorReport
	r.OK = code == 0 && r.Summary.Fail == 0
	return r
}

var lastDoctorReport DoctorReport
var doctorQuiet bool

func doctorPrintf(format string, args ...any) {
	if doctorQuiet {
		return
	}
	fmt.Printf(format, args...)
}
func doctorPrintln(args ...any) {
	if doctorQuiet {
		return
	}
	fmt.Println(args...)
}

type doctorHooks struct {
	check          func(string, bool)
	warnCheck      func(string, bool)
	exists         func(string) bool
	curlOK         func(string) bool
	curlOKInsecure func(string) bool
	etc            string
	state          string
	role           string
	host           string
	ok, fail, warn *int
}

func runDoctorNative() int {
	ok, fail, warn := 0, 0, 0
	lastDoctorReport = DoctorReport{}
	check := func(name string, good bool) {
		st := "ok"
		if good {
			doctorPrintf("%s %s\n", cli18n.T("doctor.ok"), name)
			ok++
		} else {
			doctorPrintf("%s %s\n", cli18n.T("doctor.fail"), name)
			fail++
			st = "fail"
		}
		lastDoctorReport.Checks = append(lastDoctorReport.Checks, DoctorCheck{ID: name, Status: st})
	}
	warnCheck := func(name string, good bool) {
		st := "ok"
		if good {
			doctorPrintf("%s %s\n", cli18n.T("doctor.ok"), name)
			ok++
		} else {
			doctorPrintf("%s %s\n", cli18n.T("doctor.warn"), name)
			warn++
			st = "warn"
		}
		lastDoctorReport.Checks = append(lastDoctorReport.Checks, DoctorCheck{ID: name, Status: st})
	}
	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}
	curlOK := func(url string) bool {
		return exec.Command("curl", "-fsS", "--max-time", "2", "-o", "/dev/null", url).Run() == nil
	}
	curlOKInsecure := func(url string) bool {
		return exec.Command("curl", "-fskS", "--max-time", "2", "-o", "/dev/null", url).Run() == nil
	}

	host, _ := os.Hostname()
	role := detectRole()
	lastDoctorReport.Role = role
	lastDoctorReport.Host = host
	doctorPrintf(cli18n.T("doctor.header")+"\n", role, host)
	etc := paths.EtcDir()
	state := paths.StateDir()

	// Prefer live VERSION / binary over stale state/installed_version (often left from first install).
	if ver := version.Running(); ver != "" {
		doctorPrintf(cli18n.T("doctor.installed_version")+"\n", ver)
		_ = os.WriteFile(filepath.Join(state, "installed_version"), []byte(ver+"\n"), 0o644)
	} else if b, err := os.ReadFile(filepath.Join(state, "installed_version")); err == nil {
		doctorPrintf(cli18n.T("doctor.installed_version")+"\n", strings.TrimSpace(string(b)))
	}
	doctorRunChecks(doctorHooks{
		check: check, warnCheck: warnCheck, exists: exists,
		curlOK: curlOK, curlOKInsecure: curlOKInsecure,
		etc: etc, state: state, role: role, host: host,
		ok: &ok, fail: &fail, warn: &warn,
	})
	lastDoctorReport.Summary.OK = ok
	lastDoctorReport.Summary.Fail = fail
	lastDoctorReport.Summary.Warn = warn
	lastDoctorReport.OK = fail == 0
	if fail > 0 {
		return 1
	}

	return 0
}

func dirHasDockerLampac() bool {
	return exec.Command("docker", "ps", "-q", "-f", "name=netductor-lampac").Run() == nil
}

func dockerLampacHealthy() bool {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Health.Status}}", "netductor-lampac").Output()
	if err != nil {
		return exec.Command("docker", "ps", "-q", "-f", "name=netductor-lampac", "-f", "status=running").Run() == nil
	}
	s := strings.TrimSpace(string(out))
	return s == "healthy" || s == ""
}

func mustReadFile(name string) []byte {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil
	}
	return b
}

func latestGitHubTag() string {
	// Best-effort: try gh or curl to GitHub API. Silent on failure.
	if out, err := exec.Command("gh", "api", "repos/PavelNeyman/netductor/releases/latest", "--jq", ".tag_name").CombinedOutput(); err == nil {
		return strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("curl", "-fsS", "--max-time", "5", "https://api.github.com/repos/PavelNeyman/netductor/releases/latest").CombinedOutput(); err == nil {
		// crude parse
		s := string(out)
		if i := strings.Index(s, `"tag_name":`); i >= 0 {
			rest := s[i+len(`"tag_name":`):]
			if j := strings.Index(rest, `"`); j >= 0 {
				rest = rest[j+1:]
				if k := strings.Index(rest, `"`); k >= 0 {
					return rest[:k]
				}
			}
		}
	}
	return ""
}
