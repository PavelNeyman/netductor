package logs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// ExportDir holds on-demand and alert-attached log bundles.
func ExportDir() string {
	return filepath.Join(paths.StateDir(), "log-exports")
}

// Rotate vacuums journal to KeepHours and purges old exports.
func Rotate() (string, error) {
	s := LoadSchedule()
	keep := fmt.Sprintf("%dh", s.KeepHours)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("logs rotate keep=%s at=%s\n", keep, time.Now().UTC().Format(time.RFC3339)))

	if out, err := exec.Command("journalctl", "--vacuum-time="+keep).CombinedOutput(); err != nil {
		b.WriteString(fmt.Sprintf("vacuum-time: %v %s\n", err, string(out)))
	} else {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		b.WriteString("vacuum-time: ok " + msg + "\n")
	}

	_ = os.MkdirAll(ExportDir(), 0o700)
	cutoff := time.Now().Add(-time.Duration(s.KeepHours) * time.Hour)
	n := 0
	ents, _ := os.ReadDir(ExportDir())
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(ExportDir(), e.Name())
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.ModTime().Before(cutoff) {
			_ = os.Remove(p)
			n++
		}
	}
	b.WriteString(fmt.Sprintf("exports_removed=%d\n", n))
	return b.String(), nil
}
