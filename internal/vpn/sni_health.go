package vpn

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SNIHealth checks that the VLESS/Reality port accepts TCP.
// A full TLS handshake with stock crypto/tls almost always fails on Reality
// (server expects REALITY auth) — that is NOT treated as downtime.
func SNIHealth() (ok bool, ms int64, detail string) {
	sniName := sni()
	port := vlessPort()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	t0 := time.Now()
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return false, 0, "dial: " + err.Error()
	}
	defer conn.Close()
	msDial := time.Since(t0).Milliseconds()

	cfg := &tls.Config{ServerName: sniName, InsecureSkipVerify: true, NextProtos: []string{"h2", "http/1.1"}}
	tconn := tls.Client(conn, cfg)
	_ = tconn.SetDeadline(time.Now().Add(2 * time.Second))
	err = tconn.Handshake()
	ms = time.Since(t0).Milliseconds()
	if err != nil {
		// Port is open; Reality rejects vanilla TLS — expected.
		return true, msDial, fmt.Sprintf("sni=%s port_up dial_ms=%d (reality rejects plain TLS: %v)", sniName, msDial, err)
	}
	return true, ms, fmt.Sprintf("sni=%s tls_ok", sniName)
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
