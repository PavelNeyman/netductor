package tlsle

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CertInfo is a live LE cert under /etc/letsencrypt/live.
type CertInfo struct {
	Name     string
	Path     string
	NotAfter time.Time
	DaysLeft int
}

// ListLiveCerts scans /etc/letsencrypt/live/*/fullchain.pem.
func ListLiveCerts() []CertInfo {
	root := "/etc/letsencrypt/live"
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []CertInfo
	now := time.Now()
	for _, e := range ents {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(root, e.Name(), "fullchain.pem")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		block, _ := pem.Decode(b)
		if block == nil {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		days := int(cert.NotAfter.Sub(now).Hours() / 24)
		out = append(out, CertInfo{Name: e.Name(), Path: path, NotAfter: cert.NotAfter, DaysLeft: days})
	}
	return out
}
