package gitstore

import (
	"strings"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EnqueueBuildHostCmd appends a cmd for an online (or any) build host agent.
func allowedBuildHostCmd(cmd string) bool {
	switch strings.TrimSpace(cmd) {
	case "backup_pull", "recovery_arm":
		return true
	default:
		return false
	}
}

func EnqueueBuildHostCmd(id, cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if !allowedBuildHostCmd(cmd) {
		return fmt.Errorf("build-host cmd not allowed: %q", cmd)
	}
	if _, err := LoadBuildHostState(id); err != nil {
		return err
	}
	path := filepath.Join(buildHostsStateDir(), sanitize(id)+".cmds.json")
	var cmds []string
	if b, e := os.ReadFile(path); e == nil {
		_ = json.Unmarshal(b, &cmds)
	}
	cmds = append(cmds, cmd)
	b, _ := json.MarshalIndent(cmds, "", "  ")
	_ = os.MkdirAll(buildHostsStateDir(), 0o700)
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// TakeBuildHostCmds returns and clears pending cmds for host.
func TakeBuildHostCmds(id string) []string {
	path := filepath.Join(buildHostsStateDir(), sanitize(id)+".cmds.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cmds []string
	_ = json.Unmarshal(b, &cmds)
	_ = os.Remove(path)
	return cmds
}

// EnqueueBackupPullAllBuildHosts stages mirror + queues backup_pull on online hosts.
func EnqueueBackupPullAllBuildHosts() (n int, err error) {
	_ = QueueBuildHostBackupMirror()
	for _, s := range ListBuildHostStates() {
		if !s.Online {
			continue
		}
		if e := EnqueueBuildHostCmd(s.ID, "backup_pull"); e == nil {
			n++
		}
	}
	if n == 0 {
		return 0, fmt.Errorf("no online build-host")
	}
	return n, nil
}

// RecordBuildHostHeartbeat also drains nothing — cmds returned by API.
func TouchBuildHostSeen(id string) {
	s, err := LoadBuildHostState(id)
	if err != nil {
		return
	}
	s.LastSeen = time.Now().UTC().Format(time.RFC3339)
	s.Online = true
	_ = SaveBuildHostState(s)
}
