package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

// readBodyLimited reads at most limit bytes. If more data is available, returns error
// (does not silently truncate).
func readBodyLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("body exceeds %d bytes", limit)
	}
	return data, nil
}

// tarMembersSafe rejects absolute paths, .. components, unsafe links, and exotic types.
func tarMembersSafe(archivePath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".tgz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
		defer gr.Close()
		r = gr
	}
	tr := tar.NewReader(r)
	const maxMembers = 10000
	n := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		n++
		if n > maxMembers {
			return fmt.Errorf("too many tar members")
		}
		name := hdr.Name
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
			return fmt.Errorf("unsafe tar member: %q", name)
		}
		// path.Clean collapses .. ; reject if result escapes or still has ..
		clean := path.Clean(name)
		if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
			return fmt.Errorf("unsafe tar member: %q", name)
		}
		// Also reject raw ".." path elements before Clean tricks
		for _, part := range strings.Split(name, "/") {
			if part == ".." {
				return fmt.Errorf("unsafe tar member: %q", name)
			}
		}
		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			link := hdr.Linkname
			if link == "" || strings.HasPrefix(link, "/") || strings.Contains(link, "..") {
				return fmt.Errorf("unsafe tar link: %q -> %q", name, link)
			}
		case tar.TypeReg, tar.TypeRegA, tar.TypeDir, tar.TypeXGlobalHeader, tar.TypeXHeader, tar.TypeGNUSparse:
			// allowed
		default:
			return fmt.Errorf("unsupported tar type %c for %q", hdr.Typeflag, name)
		}
	}
	return nil
}

// extractTarSafe validates members then extracts into destDir only.
func extractTarSafe(archivePath, destDir string) error {
	if err := tarMembersSafe(archivePath); err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	out, err := exec.Command("tar", "-xzf", archivePath, "-C", destDir, "--no-same-owner").CombinedOutput()
	if err != nil {
		return fmt.Errorf("extract: %s: %w", truncate(string(out), 200), err)
	}
	return nil
}
