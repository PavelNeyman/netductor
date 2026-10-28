package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeTarGz(t *testing.T, path string, files map[string]string, evilName string) {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if evilName != "" {
		hdr := &tar.Header{Name: evilName, Mode: 0o644, Size: 1}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	_ = tw.Close()
	_ = gw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTarMembersSafeRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "evil.tar.gz")
	writeTarGz(t, p, map[string]string{"ok.txt": "hi"}, "../etc/passwd")
	if err := tarMembersSafe(p); err == nil {
		t.Fatal("expected reject")
	}
}

func TestTarMembersSafeRejectsAbsolute(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "abs.tar.gz")
	writeTarGz(t, p, nil, "/etc/passwd")
	if err := tarMembersSafe(p); err == nil {
		t.Fatal("expected reject")
	}
}

func TestTarMembersSafeOK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ok.tar.gz")
	writeTarGz(t, p, map[string]string{"config/network": "x", "config/dhcp": "y"}, "")
	if err := tarMembersSafe(p); err != nil {
		t.Fatal(err)
	}
}

func TestReadBodyLimited(t *testing.T) {
	r := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), 100)))
	b, err := readBodyLimited(r, 50)
	if err == nil || b != nil {
		t.Fatalf("expected oversize error, got %v %d", err, len(b))
	}
	r2 := io.NopCloser(bytes.NewReader([]byte("hello")))
	b2, err := readBodyLimited(r2, 50)
	if err != nil || string(b2) != "hello" {
		t.Fatal(err, string(b2))
	}
}
