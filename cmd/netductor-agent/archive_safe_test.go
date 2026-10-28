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

func writeTarGz(t *testing.T, path string, files map[string]string, extra func(*tar.Writer)) {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if extra != nil {
		extra(tw)
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
	writeTarGz(t, p, map[string]string{"ok.txt": "hi"}, func(tw *tar.Writer) {
		hdr := &tar.Header{Name: "../etc/passwd", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte("x"))
	})
	if err := tarMembersSafe(p); err == nil {
		t.Fatal("expected reject")
	}
}

func TestTarMembersSafeRejectsAbsolute(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "abs.tar.gz")
	writeTarGz(t, p, nil, func(tw *tar.Writer) {
		hdr := &tar.Header{Name: "/etc/passwd", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte("x"))
	})
	if err := tarMembersSafe(p); err == nil {
		t.Fatal("expected reject")
	}
}

func TestTarMembersSafeRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "link.tar.gz")
	writeTarGz(t, p, map[string]string{"a": "1"}, func(tw *tar.Writer) {
		hdr := &tar.Header{Name: "evil", Mode: 0o777, Typeflag: tar.TypeSymlink, Linkname: "network"}
		_ = tw.WriteHeader(hdr)
	})
	if err := tarMembersSafe(p); err == nil {
		t.Fatal("symlink must fail")
	}
}

func TestTarMembersSafeConfigPrefix(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.tar.gz")
	writeTarGz(t, p, map[string]string{"config/network": "x", "config/dhcp": "y"}, nil)
	if err := tarMembersSafeOpts(p, tarScanOpts{RequirePrefix: "config/"}); err != nil {
		t.Fatal(err)
	}
	p2 := filepath.Join(dir, "bad.tar.gz")
	writeTarGz(t, p2, map[string]string{"other/network": "x"}, nil)
	if err := tarMembersSafeOpts(p2, tarScanOpts{RequirePrefix: "config/"}); err == nil {
		t.Fatal("outside prefix must fail")
	}
}

func TestTarMembersSafeOK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ok.tar.gz")
	writeTarGz(t, p, map[string]string{"config/network": "x", "config/dhcp": "y"}, nil)
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
