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
func agentHTTPClient() *http.Client {
	c := &http.Client{Timeout: 120 * time.Second}
	dir := findMTLSDir()
	if dir == "" {
		return c
	}
	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, "client.crt"), filepath.Join(dir, "client.key"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "mtls load client:", err)
		return c
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "mtls load ca:", err)
		return c
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		fmt.Fprintln(os.Stderr, "mtls: no CA certs parsed")
		return c
	}
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

// normalizeServerURL forces https://host:8789 when mTLS material is present.
func normalizeServerURL(server string) string {
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	if findMTLSDir() == "" {
		return server
	}
	rest := server
	rest = strings.TrimPrefix(rest, "https://")
	rest = strings.TrimPrefix(rest, "http://")
	host := rest
	if i := strings.Index(rest, "/"); i >= 0 {
		host = rest[:i]
	}
	if i := strings.LastIndex(host, ":"); i > 0 {
		// strip port
		host = host[:i]
	}
	return "https://" + host + ":8789"
}
