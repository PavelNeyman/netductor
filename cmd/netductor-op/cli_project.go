package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runProjectBuild is the Mac-side entry from TG notifications.
// Full Xcode/local build stays with the developer; this CLI documents the contract
// and optionally clones the public/private GitHub repo for local work.
func runProjectBuild(args []string) int {
	// netductor-op project build <name> --ref <ref>
	if len(args) < 1 || args[0] != "build" {
		fmt.Fprintln(os.Stderr, "usage: netductor-op project build <name> [--ref REF]")
		return 2
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: netductor-op project build <name> [--ref REF]")
		return 2
	}
	name := args[1]
	ref := "main"
	for i := 2; i < len(args); i++ {
		if args[i] == "--ref" && i+1 < len(args) {
			ref = args[i+1]
			i++
		}
	}
	fmt.Printf("host=mac build: project=%s ref=%s\n", name, ref)
	fmt.Println("1) Clone/pull from GitHub (private: need repos token)")
	fmt.Printf("   git clone --branch %s git@github.com:PavelNeyman/%s.git\n", ref, name)
	fmt.Println("2) Build on this Mac (Xcode / xcodebuild / swift / …)")
	fmt.Println("3) Upload artifacts to VPS local release or scp")
	fmt.Println("4) On primary: netductor git project queue done <queue_id>")
	// Try open repo if already present under ~/src or cwd
	if st, err := os.Stat(name); err == nil && st.IsDir() {
		fmt.Println("found local dir", name)
		_ = exec.Command("git", "-C", name, "fetch", "--all").Run()
		_ = exec.Command("git", "-C", name, "checkout", ref).Run()
	}
	_ = strings.TrimSpace(ref)
	return 0
}
