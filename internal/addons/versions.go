package addons

import (
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Versions is the single status shape for managed addons (R2).
// stack apply does not upgrade these — use `netductor addons update`.
func Versions() map[string]any {
	items := All()
	for _, st := range items {
		if st["name"] == "lampac" && st["update_available"] == true {
			notify.AlertOnce("addon-lampac-update",
				"Lampac image update available: running "+str(st["running"])+" desired "+str(st["desired"]))
		}
	}
	out := map[string]any{
		"note":  "netductor addons update [name|all] pulls images and recreates; new addons call Register",
		"items": items,
	}
	// Compat: some UIs still read versions["lampac"] directly
	for _, st := range items {
		if st["name"] == "lampac" {
			out["lampac"] = st
			break
		}
	}
	return out
}

func imageStatus(container, name string) map[string]any {
	running := strings.TrimSpace(run("docker", "inspect", "-f", "{{.Config.Image}}", container))
	desired := strings.TrimSpace(readTag(name))
	upd := desired != "" && running != "" && desired != running
	return map[string]any{
		"running": running, "desired": desired, "update_available": upd,
	}
}

func readTag(name string) string {
	b, err := os.ReadFile(paths.EtcDir() + "/addons/" + name + ".tag")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func run(bin string, args ...string) string {
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
