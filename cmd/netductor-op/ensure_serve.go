package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ensureOpServe starts netductor-op serve in background if health fails.
func ensureOpServe() error {
	base := opBaseURL()
	if opHealthOK(base) {
		return nil
	}
	bin, err := os.Executable()
	if err != nil {
		bin = "netductor-op"
	}
	bin, _ = filepath.EvalSymlinks(bin)
	cmd := exec.Command(bin, "serve")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start serve: %w (run: netductor-op serve)", err)
	}
	// wait up to ~5s for health
	for i := 0; i < 25; i++ {
		time.Sleep(200 * time.Millisecond)
		if opHealthOK(base) {
			fmt.Fprintf(os.Stderr, "op serve auto-started pid=%d\n", cmd.Process.Pid)
			return nil
		}
	}
	return fmt.Errorf("op serve not healthy at %s — run: netductor-op serve", base)
}
