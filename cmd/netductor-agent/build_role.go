package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
		if err := buildHeartbeat(client, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "build heartbeat: %v\n", err)
		}
		time.Sleep(time.Duration(iv) * time.Second)
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
	return nil
}
