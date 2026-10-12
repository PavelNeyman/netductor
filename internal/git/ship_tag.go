package gitstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// NextProjectTag returns vM.m.p+1 from published artifact dir names (v*), or v0.0.1.
func NextProjectTag(project string) string {
	project = sanitize(project)
	dir := filepath.Join(ArtifactDir(), project)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "v0.0.1"
	}
	best := ""
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasPrefix(n, "v") {
			continue
		}
		if best == "" || verGreater(n, best) {
			best = n
		}
	}
	if best == "" {
		return "v0.0.1"
	}
	return "v" + bumpPatchStr(strings.TrimPrefix(best, "v"))
}

func bumpPatchStr(ver string) string {
	parts := strings.Split(ver, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	maj, _ := strconv.Atoi(parts[0])
	min, _ := strconv.Atoi(parts[1])
	patStr := parts[2]
	if i := strings.IndexAny(patStr, "-+"); i >= 0 {
		patStr = patStr[:i]
	}
	pat, _ := strconv.Atoi(patStr)
	return fmt.Sprintf("%d.%d.%d", maj, min, pat+1)
}

func verGreater(a, b string) bool {
	pa := parseVer(strings.TrimPrefix(a, "v"))
	pb := parseVer(strings.TrimPrefix(b, "v"))
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseVer(s string) [3]int {
	var out [3]int
	parts := strings.Split(s, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		p := parts[i]
		if j := strings.IndexAny(p, "-+"); j >= 0 {
			p = p[:j]
		}
		out[i], _ = strconv.Atoi(p)
	}
	return out
}
