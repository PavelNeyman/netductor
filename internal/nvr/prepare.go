package nvr

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PrepareStorage creates segment dirs and prints/runs gocryptfs guidance.
// Full LUKS needs root block device — we document; gocryptfs is optional if installed.
func PrepareStorage(printOnly bool) error {
	cfg := LoadConfig()
	if err := os.MkdirAll(cfg.Path, 0o700); err != nil {
		return err
	}
	cipherDir := filepath.Join(dir(), "cipher")
	plainGuide := cfg.Path
	_ = os.MkdirAll(cipherDir, 0o700)

	fmt.Println("NVR path:", cfg.Path)
	fmt.Println("Retention: days=", cfg.RetentionDays, "max_gb=", cfg.MaxGB, "min_free_gb=", cfg.MinFreeGB)
	fmt.Println()
	fmt.Println("Encryption (recommended):")
	fmt.Println("  1) Install gocryptfs: apt install gocryptfs")
	fmt.Println("  2) gocryptfs -init", cipherDir)
	fmt.Println("  3) Mount before record:")
	fmt.Printf("     gocryptfs %s %s\n", cipherDir, plainGuide)
	fmt.Println("  4) Or LUKS volume mounted at", plainGuide)
	fmt.Println("  Unlock after reboot is operator responsibility (see PLAN-NVR-TAPO).")
	fmt.Println()
	if printOnly {
		return nil
	}
	if _, err := exec.LookPath("gocryptfs"); err != nil {
		fmt.Println("gocryptfs not in PATH — dirs created, encrypt later.")
		return nil
	}
	// Do not auto-init (needs passphrase interactively).
	fmt.Println("gocryptfs found. Run init/mount manually with a passphrase.")
	return nil
}
