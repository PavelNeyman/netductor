package nvr

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	recMu   sync.Mutex
	recCmds = map[string]*exec.Cmd{} // camera id -> ffmpeg
)

// StartRecorder launches ffmpeg on primary (only if RTSP is routable from VPS).
// Prefer site-agent nvr_record_* when cameras are behind NAT.
// Does NOT auto-restart on failure (avoids CPU spin); operator/UI restarts.
func StartRecorder(c Camera) error {
	cfg := LoadConfig()
	if !cfg.RecordEnabled || !c.Record || !c.Enabled {
		return fmt.Errorf("recording disabled")
	}
	url := RTSPURL(c)
	if url == "" {
		return fmt.Errorf("no rtsp url (ip/secret?)")
	}
	dir := filepath.Join(cfg.Path, c.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	seg := cfg.SegmentSec
	if seg <= 0 {
		seg = 300
	}
	pattern := filepath.Join(dir, "%Y-%m-%d_%H-%M-%S.mp4")
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-rtsp_transport", "tcp",
		"-timeout", "5000000",
		"-rw_timeout", "5000000",
		"-stimeout", "5000000",
		"-i", url,
		"-c", "copy",
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", seg),
		"-segment_atclocktime", "1",
		"-strftime", "1",
		"-reset_timestamps", "1",
		"-break_non_keyframes", "1",
		pattern,
	}
	recMu.Lock()
	defer recMu.Unlock()
	if old, ok := recCmds[c.ID]; ok && old.Process != nil {
		_ = old.Process.Kill()
		delete(recCmds, c.ID)
	}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	recCmds[c.ID] = cmd
	go func(id string, p *exec.Cmd) {
		_ = p.Wait()
		recMu.Lock()
		if recCmds[id] == p {
			delete(recCmds, id)
		}
		recMu.Unlock()
	}(c.ID, cmd)
	return nil
}

// StopRecorder stops ffmpeg for camera id.
func StopRecorder(id string) {
	recMu.Lock()
	defer recMu.Unlock()
	if c, ok := recCmds[id]; ok && c.Process != nil {
		_ = c.Process.Kill()
		delete(recCmds, id)
	}
}

// RecorderRunning reports active recorders.
func RecorderRunning() map[string]bool {
	recMu.Lock()
	defer recMu.Unlock()
	out := map[string]bool{}
	for id, c := range recCmds {
		out[id] = c.Process != nil
	}
	return out
}
