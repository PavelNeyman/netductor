package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Default locations for mTLS client material on the router.
var mtlsDirs = []string{
	"/etc/netductor-agent/mtls",
	"/etc/netductor/secrets/mtls",
}

func mtlsRequiredFlagPath() string {
	return filepath.Join(agentDir(), "mtls_required")
}

// markMTLSRequired sets sticky flag after successful mTLS material install/use.
func markMTLSRequired() {
	_ = os.MkdirAll(agentDir(), 0o700)
	_ = os.WriteFile(mtlsRequiredFlagPath(), []byte("1\n"), 0o600)
}

func mtlsRequired() bool {
	if os.Getenv("NETDUCTOR_MTLS_REQUIRED") == "1" {
		return true
	}
	if os.Getenv("NETDUCTOR_MTLS_REQUIRED") == "0" {
		return false
	}
	_, err := os.Stat(mtlsRequiredFlagPath())
	return err == nil
}

func findMTLSDir() string {
	for _, d := range mtlsDirs {
		if fileOK(filepath.Join(d, "ca.crt")) &&
			fileOK(filepath.Join(d, "client.crt")) &&
			fileOK(filepath.Join(d, "client.key")) {
			return d
		}
	}
	return ""
}

func fileOK(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Size() > 0
}

// agentHTTPClient returns an HTTP client; with mTLS certs uses TLS 1.3 client auth.
// Once mTLS was required (flag or complete material used), missing/broken material fails closed —
// no silent bearer fallback (review P1).
func agentHTTPClient() *http.Client {
	c := &http.Client{Timeout: 120 * time.Second}
	dir := findMTLSDir()
	if dir == "" {
		if mtlsRequired() {
			err := fmt.Errorf("mtls required but ca.crt/client.crt/client.key incomplete or missing")
			fmt.Fprintln(os.Stderr, err)
			c.Transport = &failClosedRoundTripper{err: err}
			return c
		}
		return c // bootstrap / pre-enroll only
	}
	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, "client.crt"), filepath.Join(dir, "client.key"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "mtls load client (fail-closed):", err)
		c.Transport = &failClosedRoundTripper{err: err}
		return c
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "mtls load ca (fail-closed):", err)
		c.Transport = &failClosedRoundTripper{err: err}
		return c
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		err := fmt.Errorf("mtls: no CA certs parsed")
		fmt.Fprintln(os.Stderr, err)
		c.Transport = &failClosedRoundTripper{err: err}
		return c
	}
	markMTLSRequired()
	c.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      pool,
			MinVersion:   tls.VersionTLS13,
			ServerName:   "netductor-agent-server",
		},
	}
	fmt.Fprintln(os.Stderr, "mtls: client certs from", dir)
	return c
}

type failClosedRoundTripper struct{ err error }

func (f *failClosedRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, fmt.Errorf("mtls required but unavailable: %w", f.err)
	}
	return nil, fmt.Errorf("mtls required but unavailable")
}

// normalizeServerURL forces https://host:8789 when mTLS material is present or required.
func normalizeServerURL(server string) string {
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	if findMTLSDir() == "" && !mtlsRequired() {
		return server
	}
	rest := server
	rest = strings.TrimPrefix(rest, "https://")
	rest = strings.TrimPrefix(rest, "http://")
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	if !strings.Contains(rest, ":") {
		rest += ":8789"
	}
	return "https://" + rest
}
