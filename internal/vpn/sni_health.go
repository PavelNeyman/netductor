package vpn

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SNIHealth checks that the VLESS/Reality port accepts TCP.
// Do NOT attempt a stock crypto/tls handshake: Reality rejects it and sing-box
// logs "REALITY: processed invalid connection" for 127.0.0.1 every collect tick.
func SNIHealth() (ok bool, ms int64, detail string) {
	sniName := sni()
	port := vlessPort()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	t0 := time.Now()
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.Dial("tcp", addr)
	ms = time.Since(t0).Milliseconds()
	if err != nil {
		return false, ms, "dial: " + err.Error()
	}
	_ = conn.Close()
	return true, ms, fmt.Sprintf("sni=%s port_up dial_ms=%d (tcp-only; no plain TLS to Reality)", sniName, ms)
}

// SNIDialDown reports whether detail from SNIHealth means the port is unreachable.
func SNIDialDown(detail string) bool {
	return strings.HasPrefix(detail, "dial:")
}

func WriteSNIHealthMetric(ok bool, ms int64, detail string) {
	dir := filepath.Join(paths.StateDir(), "metrics")
	_ = os.MkdirAll(dir, 0o755)
	line := fmt.Sprintf("%d ok=%v ms=%d %s\n", time.Now().Unix(), ok, ms, detail)
	_ = os.WriteFile(filepath.Join(dir, "sni_health.txt"), []byte(line), 0o644)
}
