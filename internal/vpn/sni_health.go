package vpn

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SNIHealth tries TLS handshake to local Reality port with configured SNI.
func SNIHealth() (ok bool, ms int64, detail string) {
	sniName := sni()
	port := vlessPort()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	t0 := time.Now()
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return false, 0, err.Error()
	}
	defer conn.Close()
	cfg := &tls.Config{ServerName: sniName, InsecureSkipVerify: true, NextProtos: []string{"h2", "http/1.1"}}
	tconn := tls.Client(conn, cfg)
	_ = tconn.SetDeadline(time.Now().Add(3 * time.Second))
	err = tconn.Handshake()
	ms = time.Since(t0).Milliseconds()
	if err != nil {
		// Reality may fail standard TLS; still records latency to accept
		return false, ms, fmt.Sprintf("sni=%s handshake: %v", sniName, err)
	}
	return true, ms, fmt.Sprintf("sni=%s ok", sniName)
}

func WriteSNIHealthMetric(ok bool, ms int64, detail string) {
	dir := filepath.Join(paths.StateDir(), "metrics")
	_ = os.MkdirAll(dir, 0o755)
	line := fmt.Sprintf("%d ok=%v ms=%d %s\n", time.Now().Unix(), ok, ms, detail)
	_ = os.WriteFile(filepath.Join(dir, "sni_health.txt"), []byte(line), 0o644)
}
