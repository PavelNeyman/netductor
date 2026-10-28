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

const (
	tarMaxMembers    = 10000
	tarMaxFileBytes  = 32 << 20  // 32 MiB per member
	tarMaxTotalBytes = 256 << 20 // 256 MiB total declared size
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

type tarScanOpts struct {
	// RequirePrefix, if set, requires every non-meta member path to start with it
	// (e.g. "config/" for configBackup archives).
	RequirePrefix string
	// AllowEmpty accepts archives with zero file members (dirs/meta only).
	AllowEmpty bool
}

// tarMembersSafe rejects absolute paths, .. components, links, exotic types,
// and enforces declared size budgets.
func tarMembersSafe(archivePath string) error {
	return tarMembersSafeOpts(archivePath, tarScanOpts{})
}

func tarMembersSafeOpts(archivePath string, opts tarScanOpts) error {
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
	n := 0
	var total int64
	fileMembers := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		n++
		if n > tarMaxMembers {
			return fmt.Errorf("too many tar members")
		}
		switch hdr.Typeflag {
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			continue
		}
		name := hdr.Name
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
			return fmt.Errorf("unsafe tar member: %q", name)
		}
		clean := path.Clean(name)
		if clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("unsafe tar member: %q", name)
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".." {
				return fmt.Errorf("unsafe tar member: %q", name)
			}
		}
		if opts.RequirePrefix != "" {
			pref := strings.TrimSuffix(opts.RequirePrefix, "/")
			if clean != pref && !strings.HasPrefix(clean, pref+"/") {
				return fmt.Errorf("tar member outside %s: %q", opts.RequirePrefix, name)
			}
		}
		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("tar links not allowed: %q", name)
		case tar.TypeReg, tar.TypeRegA:
			fileMembers++
			if hdr.Size < 0 || hdr.Size > tarMaxFileBytes {
				return fmt.Errorf("tar member too large: %q size=%d", name, hdr.Size)
			}
			total += hdr.Size
			if total > tarMaxTotalBytes {
				return fmt.Errorf("tar total size exceeds %d", tarMaxTotalBytes)
			}
		case tar.TypeDir:
			// ok
		default:
			return fmt.Errorf("unsupported tar type %c for %q", hdr.Typeflag, name)
		}
	}
	if fileMembers == 0 && !opts.AllowEmpty {
		return fmt.Errorf("tar has no file members")
	}
	return nil
}

// extractTarSafe validates members then extracts into destDir only.
func extractTarSafe(archivePath, destDir string) error {
	return extractTarSafeOpts(archivePath, destDir, tarScanOpts{}, nil)
}

// extractTarSafeOpts validates then extracts. stripArgs are extra tar flags
// (e.g. --strip-components=1 for config/ rooted archives).
func extractTarSafeOpts(archivePath, destDir string, opts tarScanOpts, stripArgs []string) error {
	if err := tarMembersSafeOpts(archivePath, opts); err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	args := []string{"-xzf", archivePath, "-C", destDir, "--no-same-owner"}
	args = append(args, stripArgs...)
	out, err := exec.Command("tar", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("extract: %s: %w", truncate(string(out), 200), err)
	}
	return nil
}
