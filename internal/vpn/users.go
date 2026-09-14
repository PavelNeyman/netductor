package vpn

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,63}$`)

func Bin() string {
	return paths.VPNBin()
}
func Clients() string { return paths.ClientsDir() }

func ValidName(name string) bool {
	return name != "" && nameRe.MatchString(name)
}

func run(args ...string) (string, error) {
	out, err := exec.Command(Bin(), args...).CombinedOutput()
	return string(out), err
}

type User struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	UUID    string `json:"uuid"`
	Note    string `json:"note"`
	Created string `json:"created"`
}

func List() ([]User, error) {
	if users, err := ListNative(); err == nil {
		return users, nil
	}
	out, err := run("list")
	if err != nil {
		return nil, err
	}
	var users []User
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		p := strings.Split(line, "\t")
		if len(p) < 3 {
			continue
		}
		u := User{Name: p[0], Enabled: p[1] == "on", UUID: p[2]}
		if len(p) > 3 {
			u.Note = p[3]
		}
		if len(p) > 4 {
			u.Created = p[4]
		}
		users = append(users, u)
	}
	if users == nil {
		users = []User{}
	}
	return users, nil
}

func Add(name, note string) (string, error) {
	if out, err := AddNative(name, note); err == nil || out != "" {
		return out, err
	}
	return run("add", name, note)
}

func Note(name, note string) (string, error) {
	if err := SetNoteNative(name, note); err == nil {
		return "note updated " + name, nil
	}
	return run("note", name, note)
}
func Disable(name string) (string, error) {
	if err := SetEnabledNative(name, false); err == nil {
		return "disabled " + name, nil
	}
	return run("disable", name)
}
func Enable(name string) (string, error) {
	if err := SetEnabledNative(name, true); err == nil {
		return "enabled " + name, nil
	}
	return run("enable", name)
}
func Revoke(name string) (string, error) {
	if err := RevokeNative(name); err == nil {
		return "revoked " + name, nil
	}
	return run("revoke", name)
}

func ReadClient(name string, candidates ...string) (string, bool) {
	for _, c := range candidates {
		b, err := os.ReadFile(filepath.Join(Clients(), name, c))
		if err == nil {
			return strings.TrimSpace(string(b)), true
		}
	}
	return "", false
}

func QRPath(name string) string {
	return filepath.Join(Clients(), name, "qr.png")
}

func Rename(oldName, newName string) (string, error) {
	if err := RenameNative(oldName, newName); err != nil {
		return "", err
	}
	return newName, nil
}
