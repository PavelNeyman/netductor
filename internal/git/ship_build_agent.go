package gitstore

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ReasonBuildAgent = "build_agent"
	AgentPending     = "pending"
	AgentClaimed     = "claimed"
	AgentDone        = "done"
	AgentFailed      = "failed"
)

// ShipJob agent fields stored in JSON via extra file per job.
type AgentBuildMeta struct {
	Status     string `json:"status"` // pending|claimed|done|failed
	HostID     string `json:"host_id,omitempty"`
	ClaimedAt  string `json:"claimed_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	Log        string `json:"log,omitempty"`
	Error      string `json:"error,omitempty"`
}

func agentMetaPath(id string) string {
	return filepath.Join(ShipJobsDir(), sanitize(id), "agent.json")
}

func SaveAgentMeta(id string, m AgentBuildMeta) error {
	_ = os.MkdirAll(filepath.Dir(agentMetaPath(id)), 0o700)
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(agentMetaPath(id), append(b, '\n'), 0o600)
}

func LoadAgentMeta(id string) AgentBuildMeta {
	b, err := os.ReadFile(agentMetaPath(id))
	if err != nil {
		return AgentBuildMeta{}
	}
	var m AgentBuildMeta
	_ = json.Unmarshal(b, &m)
	return m
}

// PrepareAgentBuild packs worktree for agent and parks job in waiting_external.
func PrepareAgentBuild(j *ShipJob) error {
	work, err := prepareProjectWorktree(j.Project, j.Ref)
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	art := ShipJobArtifactsDir(j.ID)
	_ = os.MkdirAll(art, 0o755)
	tarPath := filepath.Join(art, "work.tgz")
	if err := tarGzDir(work, tarPath); err != nil {
		return fmt.Errorf("pack worktree: %w", err)
	}
	_ = SaveAgentMeta(j.ID, AgentBuildMeta{Status: AgentPending})
	j.State = ShipWaitingExternal
	j.Reason = ReasonBuildAgent
	j.VpsSlot = false
	j.WaitSince = time.Now().UTC().Format(time.RFC3339)
	return SaveShipJob(j) // ptr
}

func tarGzDir(src, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rf, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, rf)
		rf.Close()
		return err
	})
}

// ClaimAgentBuildJob finds oldest pending build_agent job and assigns host.
func ClaimAgentBuildJob(hostID string) (*ShipJob, string, error) {
	shipMu.Lock()
	defer shipMu.Unlock()
	list, err := listShipJobsUnlocked()
	if err != nil {
		return nil, "", err
	}
	// oldest first
	sortByCreated := list
	for i := 0; i < len(sortByCreated); i++ {
		for j := i + 1; j < len(sortByCreated); j++ {
			if sortByCreated[j].CreatedAt < sortByCreated[i].CreatedAt {
				sortByCreated[i], sortByCreated[j] = sortByCreated[j], sortByCreated[i]
			}
		}
	}
	for _, j := range sortByCreated {
		if j.State != ShipWaitingExternal || j.Reason != ReasonBuildAgent {
			continue
		}
		m := LoadAgentMeta(j.ID)
		if m.Status != AgentPending && m.Status != "" {
			if m.Status == AgentClaimed {
				continue
			}
			if m.Status != AgentPending {
				continue
			}
		}
		if m.Status == "" {
			m.Status = AgentPending
		}
		m.Status = AgentClaimed
		m.HostID = hostID
		m.ClaimedAt = time.Now().UTC().Format(time.RFC3339)
		_ = SaveAgentMeta(j.ID, m)
		tarPath := filepath.Join(ShipJobArtifactsDir(j.ID), "work.tgz")
		return &j, tarPath, nil
	}
	return nil, "", nil
}

// CompleteAgentBuild marks job and re-queues for publish on primary.
func CompleteAgentBuild(jobID, hostID string, ok bool, logText, errMsg string) error {
	shipMu.Lock()
	defer shipMu.Unlock()
	b, err := os.ReadFile(shipJobPath(jobID))
	if err != nil {
		return err
	}
	var j ShipJob
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	m := LoadAgentMeta(jobID)
	m.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	m.Log = trimTail(logText, 8000)
	m.HostID = hostID
	if ok {
		m.Status = AgentDone
		m.Error = ""
		for i := range j.Steps {
			n := j.Steps[i].Name
			if n == "test_build" || n == "build" || strings.Contains(n, "build") {
				j.Steps[i].Status = "ok"
				j.Steps[i].FinishedAt = m.FinishedAt
				if j.Cursor <= i {
					j.Cursor = i + 1
				}
			}
		}
		j.State = ShipQueued
		j.Reason = ""
		j.VpsSlot = false
		j.Error = ""
	} else {
		m.Status = AgentFailed
		m.Error = errMsg
		j.State = ShipFailed
		j.Error = errMsg
		j.VpsSlot = false
	}
	j.LogTail = trimTail(m.Log, 2000)
	j.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = SaveAgentMeta(jobID, m)
	return saveShipJobUnlocked(&j)
}



// ExtractAndBuild is used by agent on build-host (local disk).
func ExtractAndBuild(workTGZ, outDir string) (log string, err error) {
	var b strings.Builder
	work := filepath.Join(outDir, "src")
	_ = os.RemoveAll(work)
	_ = os.MkdirAll(work, 0o755)
	cmd := exec.Command("tar", "-xzf", workTGZ, "-C", work)
	out, e := cmd.CombinedOutput()
	b.Write(out)
	if e != nil {
		return b.String(), fmt.Errorf("extract: %w", e)
	}
	// Prefer project ship script / Makefile / go test
	try := [][]string{
		{"bash", "-lc", "cd '" + work + "' && (test -f ci/netductor.yml && echo yaml-ok; go test ./... 2>&1); (cd '" + work + "' && CGO_ENABLED=0 go build -o /tmp/nd-out ./... 2>&1 || true)"},
	}
	for _, args := range try {
		cmd = exec.Command(args[0], args[1:]...)
		out, e = cmd.CombinedOutput()
		b.Write(out)
		if e != nil {
			return b.String(), e
		}
	}
	// copy binaries if any
	_ = os.MkdirAll(filepath.Join(outDir, "dist"), 0o755)
	_ = exec.Command("bash", "-lc", "cp -a "+work+"/dist/* "+outDir+"/dist/ 2>/dev/null || true").Run()
	return b.String(), nil
}
