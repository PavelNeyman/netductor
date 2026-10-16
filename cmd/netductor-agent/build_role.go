package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func runBuildAgent(cfg config) {
	fmt.Fprintf(os.Stderr, "netductor-agent role=build device=%s\n", cfg.DeviceID)
	client := agentHTTPClient()
	iv := cfg.Interval
	if iv < 15 {
		iv = 30
	}
	for {
		refreshBuildServer(&cfg)
		if err := buildHeartbeat(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "build heartbeat: %v\n", err)
		}
		if err := buildClaimAndRun(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "build claim: %v\n", err)
		}
		time.Sleep(time.Duration(iv) * time.Second)
	}
}

func refreshBuildServer(cfg *config) {
	if b, err := os.ReadFile("/etc/netductor/secrets/build_core_url"); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			cfg.Server = s
		}
	}
}

func buildHeartbeat(client *http.Client, cfg config) error {
	docker := false
	if _, err := exec.LookPath("docker"); err == nil {
		docker = true
	}
	uname, _ := exec.Command("uname", "-a").Output()
	diskFree := 0.0
	if out, err := exec.Command("df", "-BG", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			f := strings.Fields(lines[1])
			if len(f) >= 4 {
				diskFree, _ = strconv.ParseFloat(strings.TrimSuffix(f[3], "G"), 64)
			}
		}
	}
	payload := map[string]any{
		"device_id": cfg.DeviceID, "role": "build", "agent_version": version,
		"uname": strings.TrimSpace(string(uname)), "docker": docker, "disk_free_gb": diskFree,
		"goos": runtime.GOOS, "goarch": runtime.GOARCH,
	}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(cfg.Server, "/") + "/api/build-host/heartbeat"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	var hb struct {
		OK   bool     `json:"ok"`
		Cmds []string `json:"cmds"`
	}
	_ = json.NewDecoder(res.Body).Decode(&hb)
	for _, c := range hb.Cmds {
		switch c {
		case "backup_pull":
			if err := buildBackupPull(client, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "backup_pull: %v\n", err)
			}
		case "recovery_arm":
			armBuildRecovery(30 * time.Minute)
		default:
			fmt.Fprintf(os.Stderr, "build-host: ignore unknown cmd %q\n", c)
		}
	}
	return nil
}

func buildBackupPull(client *http.Client, cfg config) error {
	base := strings.TrimRight(cfg.Server, "/")
	url := base + "/api/build-host/backup/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	dir := "/var/lib/netductor-build/backups/peers/core"
	_ = os.MkdirAll(dir, 0o700)
	name := res.Header.Get("X-Backup-Name")
	if name == "" {
		name = "backup-latest.ndenc"
	}
	dst := filepath.Join(dir, name)
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	f.Close()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "backup_pull saved %s\n", dst)
	return nil
}

func buildClaimAndRun(client *http.Client, cfg config) error {
	base := strings.TrimRight(cfg.Server, "/")
	body, _ := json.Marshal(map[string]any{"device_id": cfg.DeviceID})
	req, err := http.NewRequest(http.MethodPost, base+"/api/build-host/jobs/claim", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var out struct {
		OK  bool `json:"ok"`
		Job *struct {
			ID      string `json:"id"`
			Project string `json:"project"`
			Ref     string `json:"ref"`
			WorkURL string `json:"work_url"`
		} `json:"job"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	if out.Job == nil || out.Job.ID == "" {
		return nil
	}
	fmt.Fprintf(os.Stderr, "claimed job %s project=%s\n", out.Job.ID, out.Job.Project)
	workDir := filepath.Join(os.TempDir(), "nd-build-"+out.Job.ID)
	_ = os.RemoveAll(workDir)
	_ = os.MkdirAll(workDir, 0o755)
	tgz := filepath.Join(workDir, "work.tgz")
	if err := buildDownloadFile(client, base+out.Job.WorkURL, cfg.Token, tgz); err != nil {
		return reportResult(client, cfg, out.Job.ID, false, "", err.Error())
	}
	log, err := extractAndGoBuild(tgz, workDir)
	ok := err == nil
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	if ok {
		distTGZ := filepath.Join(workDir, "dist.tgz")
		_ = exec.Command("bash", "-lc", "cd '"+workDir+"' && tar -czf dist.tgz -C dist . 2>/dev/null || tar -czf dist.tgz outbin 2>/dev/null || true").Run()
		if st, e := os.Stat(distTGZ); e == nil && st.Size() > 0 {
			_ = uploadArtifacts(client, cfg, out.Job.ID, distTGZ)
		}
	}
	return reportResult(client, cfg, out.Job.ID, ok, log, errMsg)
}

func buildDownloadFile(client *http.Client, url, token, dst string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("download %d", res.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, res.Body)
	return err
}

func extractAndGoBuild(tgz, workDir string) (string, error) {
	var b strings.Builder
	src := filepath.Join(workDir, "src")
	_ = os.MkdirAll(src, 0o755)
	cmd := exec.Command("tar", "-xzf", tgz, "-C", src)
	out, err := cmd.CombinedOutput()
	b.Write(out)
	if err != nil {
		return b.String(), err
	}
	cmd = exec.Command("bash", "-lc", "cd '"+src+"' && go test ./... 2>&1; CGO_ENABLED=0 go build -o '"+workDir+"/outbin' ./... 2>&1 || CGO_ENABLED=0 go build -o '"+workDir+"/outbin' . 2>&1")
	out, err = cmd.CombinedOutput()
	b.Write(out)
	return b.String(), err
}

func reportResult(client *http.Client, cfg config, jobID string, ok bool, log, errMsg string) error {
	base := strings.TrimRight(cfg.Server, "/")
	body, _ := json.Marshal(map[string]any{
		"ok": ok, "log": log, "error": errMsg, "host_id": cfg.DeviceID,
	})
	req, err := http.NewRequest(http.MethodPost, base+"/api/build-host/jobs/"+jobID+"/result", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("result status %d", res.StatusCode)
	}
	return nil
}


func uploadArtifacts(client *http.Client, cfg config, jobID, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	base := strings.TrimRight(cfg.Server, "/")
	req, err := http.NewRequest(http.MethodPost, base+"/api/build-host/jobs/"+jobID+"/artifacts", f)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/gzip")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("artifacts status %d", res.StatusCode)
	}
	return nil
}
