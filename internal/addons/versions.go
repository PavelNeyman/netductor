package addons

import (
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Versions reports managed addon images. stack apply does not upgrade these.
// Desired tag lives in /etc/netductor/addons/<name>.tag; running tag comes from docker.
func Versions() map[string]any {
	lampac := imageStatus("netductor-lampac", "lampac")
	if lampac["update_available"] == true {
		notify.AlertOnce("addon-lampac-update", "Lampac image update available: running "+str(lampac["running"])+" desired "+str(lampac["desired"]))
	}
	return map[string]any{
		"note": "netductor addons update [name|all] pulls images and recreates; new addons call Register",
		"items": All(),
		"lampac": lampac,
	}
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
	return string(b)
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
