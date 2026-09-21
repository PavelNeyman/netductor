package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var aptUpdated bool

func waitAptLock(maxWait time.Duration) {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		// dpkg/apt locks
		busy := false
		for _, p := range []string{
			"/var/lib/dpkg/lock-frontend",
			"/var/lib/apt/lists/lock",
			"/var/lib/dpkg/lock",
		} {
			// fuser available on most debian
			if out, _ := exec.Command("fuser", p).CombinedOutput(); len(strings.TrimSpace(string(out))) > 0 {
				busy = true
				break
			}
		}
		if !busy {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func pkgInstalled(name string) bool {
	out, err := exec.Command("dpkg-query", "-W", "-f=${Status}", name).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "install ok installed")
}

func aptUpdateOnce() error {
	if aptUpdated {
		return nil
	}
	waitAptLock(90 * time.Second)
	cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteractive; apt-get update -y -o Acquire::Retries=3")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	aptUpdated = true
	return nil
}

// aptInstall installs only missing packages; one apt-get update per process.
func aptInstall(pkgs ...string) error {
	var need []string
	for _, p := range pkgs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if pkgInstalled(p) {
			continue
		}
		need = append(need, p)
	}
	if len(need) == 0 {
		return nil
	}
	if err := aptUpdateOnce(); err != nil {
		fmt.Fprintf(os.Stderr, "apt update warn: %v\n", err)
	}
	waitAptLock(90 * time.Second)
	args := []string{
		"-c",
		"export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a; " +
			"apt-get install -y --no-install-recommends " + strings.Join(need, " "),
	}
	cmd := exec.Command("bash", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
