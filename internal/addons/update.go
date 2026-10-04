package addons

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/registry"
)

// Updater is one managed addon. New addons register in init via Register.
type Updater struct {
	Name    string
	Check   func() map[string]any
	Apply   func() error
}

var updaters []Updater

func Register(u Updater) {
	updaters = append(updaters, u)
}

func init() {
	Register(Updater{Name: "lampac", Check: func() map[string]any { return imageStatus("netductor-lampac", "lampac") }, Apply: updateLampac})
	Register(Updater{Name: "registry", Check: func() map[string]any { return imageStatus("netductor-registry", "registry") }, Apply: updateRegistry})
	Register(Updater{Name: "git", Check: gitCheck, Apply: updateGit})
}

func All() []map[string]any {
	var out []map[string]any
	for _, u := range updaters {
		st := u.Check()
		st["name"] = u.Name
		out = append(out, st)
	}
	return out
}

func Update(name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	var done []string
	var errs []string
	for _, u := range updaters {
		if name != "" && name != "all" && u.Name != name {
			continue
		}
		if err := u.Apply(); err != nil {
			errs = append(errs, u.Name+": "+err.Error())
			notify.AlertOnce("addon-update-fail-"+u.Name, "addon update failed "+u.Name+": "+err.Error())
			continue
		}
		done = append(done, u.Name)
	}
	if len(done) == 0 && len(errs) == 0 {
		return nil, fmt.Errorf("unknown addon %q", name)
	}
	return map[string]any{"updated": done, "errors": errs, "versions": All()}, nil
}

func updateLampac() error {
	img := "ghcr.io/lampac-nextgen/lampac:latest"
	if out, err := exec.Command("docker", "pull", img).CombinedOutput(); err != nil {
		return fmt.Errorf("pull: %s", strings.TrimSpace(string(out)))
	}
	return install.InstallLampac()
}

func updateRegistry() error {
	if out, err := exec.Command("docker", "pull", "registry:2").CombinedOutput(); err != nil {
		return fmt.Errorf("pull: %s", strings.TrimSpace(string(out)))
	}
	_, err := registry.Ensure()
	return err
}

func gitCheck() map[string]any {
	out, _ := exec.Command("git", "--version").Output()
	return map[string]any{"running": strings.TrimSpace(string(out)), "update_available": false, "managed_by": "apt"}
}

func updateGit() error {
	cmd := exec.Command("apt-get", "install", "-y", "git")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}
