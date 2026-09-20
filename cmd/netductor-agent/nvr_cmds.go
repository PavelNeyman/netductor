package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"github.com/PavelNeyman/netductor/internal/nvr"
	"github.com/PavelNeyman/netductor/internal/tapo"
)

func nvrDir(cfg config) string {
	if cfg.NVRDir != "" {
		return cfg.NVRDir
	}
	return "/tmp/netductor-nvr"
}

func nvrMaxBytes(cfg config) int64 {
	mb := cfg.NVRMaxMB
	if mb <= 0 {
		mb = 24 // safe default for 128MB RAM tmpfs
	}
	if mb > 4096 {
		mb = 4096
	}
	return int64(mb) * 1024 * 1024
}

func nvrRecordStart(client *http.Client, cfg config, arg string) string {
	parts := strings.Split(arg, "|")
	if len(parts) < 2 {
		return "error:arg camera_id|rtsp_url[|segment_sec]"
	}
	camID := strings.TrimSpace(parts[0])
	url := strings.TrimSpace(parts[1])
	seg := 300
	if len(parts) >= 3 {
		if n, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil && n > 0 {
			seg = n
		}
	}
	if camID == "" || url == "" {
		return "error:empty"
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "error:ffmpeg not installed"
	}
	nvrRecMu.Lock()
	running := 0
	for id, c := range nvrRecCmds {
		if id != camID && c != nil && c.Process != nil {
			running++
		}
	}
	nvrRecMu.Unlock()
	if running >= nvrMaxConcurrent {
		return "error:max concurrent records (" + strconv.Itoa(nvrMaxConcurrent) + ")"
	}
	dir := filepath.Join(nvrDir(cfg), camID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "error:" + err.Error()
	}
	nvrRecordStop(camID)
	stop := make(chan struct{})
	nvrRecMu.Lock()
	nvrRecStop[camID] = stop
	nvrRecMu.Unlock()
	go nvrSupervise(client, cfg, camID, url, seg, dir, stop)
	// dir already under nvrDir(cfg)
	go nvrUploadLoop(client, cfg, camID, dir, stop)
	return "ok:recording:" + camID + " (supervised, max " + strconv.Itoa(nvrMaxConcurrent) + " cams)"
}

// nvrSupervise runs ffmpeg with backoff when RTSP is down (no tight restart loop).
func nvrSupervise(client *http.Client, cfg config, camID, url string, seg int, dir string, stop chan struct{}) {
	backoff := 5 * time.Second
	const maxBackoff = 5 * time.Minute
	for {
		select {
		case <-stop:
			return
		default:
		}
		// cheap TCP check before spawning ffmpeg
		if err := nvrTCPCheckRTSP(url); err != nil {
			fmt.Fprintf(os.Stderr, "nvr %s: offline %v; retry in %s\n", camID, err, backoff)
			select {
			case <-stop:
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		backoff = 5 * time.Second
		_ = nvrTrimTmp(cfg)
		cmd := exec.Command("ffmpeg",
			"-hide_banner", "-loglevel", "error",
			"-rtsp_transport", "tcp",
			"-timeout", "5000000", // 5s in microseconds (some builds)
			"-rw_timeout", "5000000",
			"-stimeout", "5000000",
			"-i", url,
			"-c", "copy",
			"-f", "segment",
			"-segment_time", strconv.Itoa(seg),
			"-segment_atclocktime", "1",
			"-strftime", "1",
			"-reset_timestamps", "1",
			"-break_non_keyframes", "1",
			filepath.Join(dir, "%Y%m%d-%H%M%S.mp4"),
		)
		nvrRecMu.Lock()
		nvrRecCmds[camID] = cmd
		nvrRecMu.Unlock()
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "nvr %s: start %v\n", camID, err)
			select {
			case <-stop:
				return
			case <-time.After(backoff):
			}
			continue
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-stop:
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
			nvrRecMu.Lock()
			delete(nvrRecCmds, camID)
			nvrRecMu.Unlock()
			return
		case err := <-done:
			nvrRecMu.Lock()
			if nvrRecCmds[camID] == cmd {
				delete(nvrRecCmds, camID)
			}
			nvrRecMu.Unlock()
			fmt.Fprintf(os.Stderr, "nvr %s: ffmpeg exited %v; retry in %s\n", camID, err, backoff)
			select {
			case <-stop:
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func nvrTCPCheckRTSP(url string) error {
	hostport := url
	if strings.HasPrefix(url, "rtsp://") {
		u := url[7:]
		if i := strings.Index(u, "@"); i >= 0 {
			u = u[i+1:]
		}
		if i := strings.IndexAny(u, "/?"); i >= 0 {
			u = u[:i]
		}
		hostport = u
	}
	if !strings.Contains(hostport, ":") {
		hostport += ":554"
	}
	c, err := net.DialTimeout("tcp", hostport, 3*time.Second)
	if err != nil {
		return err
	}
	_ = c.Close()
	return nil
}

func nvrTrimTmp(cfg config) error {
	root := nvrDir(cfg)
	limit := nvrMaxBytes(cfg)
	var files []struct {
		path string
		mod  time.Time
		size int64
	}
	var total int64
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		files = append(files, struct {
			path string
			mod  time.Time
			size int64
		}{path, info.ModTime(), info.Size()})
		return nil
	})
	if total <= limit {
		return nil
	}
	// oldest first
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].mod.Before(files[i].mod) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
	for _, f := range files {
		if total <= limit*8/10 {
			break
		}
		_ = os.Remove(f.path)
		total -= f.size
	}
	return nil
}




func cameraPTZ(arg string) string {
	// Preferred: native Go port of pytapo (HA Tapo-Control protocol).
	// Fallback: python scripts/tapo_control.py, then ONVIF :2020.
	parts := strings.Split(arg, "|")
	if len(parts) < 4 {
		return "error:arg ip|user|pass|left|right|up|down|stop|night:auto|privacy:off"
	}
	ip, user, pass, dir := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), parts[2], strings.TrimSpace(parts[3])
	step := 10
	if len(parts) >= 5 {
		if n, err := strconv.Atoi(strings.TrimSpace(parts[4])); err == nil {
			step = n
		}
	}
	out := tapo.Control(ip, user, pass, dir, step)
	if strings.HasPrefix(out, "tapo-go:") && !strings.Contains(out, "tapo-go:login:") && !strings.Contains(out, "tapo-go:err:") {
		return out
	}
	// if login failed, still try python helper / ONVIF
	if pyOut, ok := tryTapoControlPy(ip, user, pass, dir, step); ok {
		return pyOut + " | first=" + out
	}
	if strings.HasPrefix(dir, "night:") || strings.HasPrefix(dir, "privacy:") {
		return out + " | need working tapo-go or pytapo"
	}
	onv := nvr.ONVIFPTZ(ip, user, pass, dir, 800)
	return onv + " | first=" + out
}

func tryTapoControlPy(ip, user, pass, dir string, step int) (string, bool) {
	py, err := exec.LookPath("python3")
	if err != nil {
		py, err = exec.LookPath("python")
		if err != nil {
			return "", false
		}
	}
	script := "/opt/netductor/scripts/tapo_control.py"
	if _, err := os.Stat(script); err != nil {
		script = "/usr/share/netductor/tapo_control.py"
	}
	if _, err := os.Stat(script); err != nil {
		// try next to agent binary
		if exe, e := os.Executable(); e == nil {
			cand := filepath.Join(filepath.Dir(exe), "tapo_control.py")
			if _, err := os.Stat(cand); err == nil {
				script = cand
			}
		}
	}
	if _, err := os.Stat(script); err != nil {
		return "", false
	}
	var args []string
	if strings.HasPrefix(dir, "night:") {
		mode := strings.TrimPrefix(dir, "night:")
		args = []string{script, ip, user, pass, "night", mode}
	} else if strings.HasPrefix(dir, "privacy:") {
		mode := strings.TrimPrefix(dir, "privacy:")
		args = []string{script, ip, user, pass, "privacy", mode}
	} else {
		args = []string{script, ip, user, pass, "move", dir, strconv.Itoa(step)}
	}
	cmd := exec.Command(py, args...)
	b, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(b))
	if err != nil {
		return "pytapo:" + out + " err:" + err.Error(), true
	}
	return "pytapo:" + out, true
}

func nvrDiskInfo(cfg config) string {
	dir := nvrDir(cfg)
	_ = os.MkdirAll(dir, 0o700)
	out, err := exec.Command("df", "-h", dir).CombinedOutput()
	info := map[string]any{"dir": dir, "max_mb": cfg.NVRMaxMB, "df": strings.TrimSpace(string(out))}
	if err != nil {
		info["error"] = err.Error()
	}
	// rough used
	var used int64
	_ = filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err == nil && fi != nil && !fi.IsDir() {
			used += fi.Size()
		}
		return nil
	})
	info["used_bytes"] = used
	info["limit_bytes"] = nvrMaxBytes(cfg)
	b, _ := json.Marshal(info)
	return string(b)
}

func nvrRecordStatus() string {
	nvrRecMu.Lock()
	defer nvrRecMu.Unlock()
	var ids []string
	for id, c := range nvrRecCmds {
		if c != nil && c.Process != nil {
			ids = append(ids, id)
		}
	}
	b, _ := json.Marshal(map[string]any{"active": ids, "count": len(ids), "max": nvrMaxConcurrent})
	return string(b)
}

func nvrRecordStop(camID string) string {
	camID = strings.TrimSpace(camID)
	nvrRecMu.Lock()
	if ch, ok := nvrRecStop[camID]; ok {
		select {
		case <-ch:
		default:
			close(ch)
		}
		delete(nvrRecStop, camID)
	}
	cmd := nvrRecCmds[camID]
	delete(nvrRecCmds, camID)
	nvrRecMu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		return "ok:stopped:" + camID
	}
	return "ok:not_running:" + camID
}

func nvrUploadLoop(client *http.Client, cfg config, camID, dir string, stop chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			nvrUploadDirOnce(client, cfg, camID, dir)
			return
		case <-ticker.C:
			nvrUploadDirOnce(client, cfg, camID, dir)
			_ = nvrTrimTmp(cfg)
		}
	}
}

func nvrUploadDirOnce(client *http.Client, cfg config, camID, dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		low := strings.ToLower(name)
		if !strings.HasSuffix(low, ".mp4") && !strings.HasSuffix(low, ".mkv") && !strings.HasSuffix(low, ".ts") {
			continue
		}
		path := filepath.Join(dir, name)
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(fi.ModTime()) < 8*time.Second {
			continue
		}
		if err := nvrUploadFile(client, cfg, camID, path); err == nil {
			_ = os.Remove(path)
		}
	}
}

func nvrUploadFile(client *http.Client, cfg config, camID, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("camera_id", camID)
	part, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	_ = w.Close()
	req, err := http.NewRequest(http.MethodPost, cfg.Server+"/api/nvr/ingest", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	tok := loadDeviceToken()
	if tok == "" {
		tok = cfg.Token
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("ingest %d %s", resp.StatusCode, string(b))
	}
	return nil
}


func rtspProbe(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "error:empty"
	}
	// Accept full rtsp URL or host:port
	hostport := arg
	path := ""
	if strings.HasPrefix(arg, "rtsp://") {
		// rtsp://user:pass@host:port/path
		u := arg[7:]
		if i := strings.Index(u, "@"); i >= 0 {
			u = u[i+1:]
		}
		if i := strings.IndexAny(u, "/?"); i >= 0 {
			path = u[i:]
			u = u[:i]
		}
		hostport = u
	}
	if !strings.Contains(hostport, ":") {
		hostport = hostport + ":554"
	}
	conn, err := net.DialTimeout("tcp", hostport, 5*time.Second)
	if err != nil {
		return "error:tcp:" + err.Error()
	}
	_ = conn.Close()
	out := "ok:tcp:" + hostport
	if path != "" {
		out += " path=" + path
	}
	// optional ffprobe if present (no password echo)
	if _, err := exec.LookPath("ffprobe"); err == nil && strings.HasPrefix(arg, "rtsp://") {
		cmd := exec.Command("ffprobe", "-v", "error", "-rtsp_transport", "tcp",
			"-timeout", "5000000", "-rw_timeout", "5000000",
			"-show_entries", "stream=codec_type", "-of", "csv=p=0", arg)
		cmd.Stdout = nil
		done := make(chan struct{})
		var b []byte
		var err error
		go func() {
			b, err = cmd.CombinedOutput()
			close(done)
		}()
		select {
		case <-done:
			if err != nil {
				out += "; ffprobe:fail:" + truncate(string(b), 120)
			} else {
				out += "; ffprobe:ok:" + truncate(strings.TrimSpace(string(b)), 80)
			}
		case <-time.After(8 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			out += "; ffprobe:timeout"
		}
	}
	return out
}


