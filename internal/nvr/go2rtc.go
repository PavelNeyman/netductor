package nvr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteGo2RTCConfig writes a go2rtc YAML for live (VPN-only consumers).
// Does not start the binary — operator runs go2rtc or optional addon later.
func WriteGo2RTCConfig() (string, error) {
	if err := ensure(); err != nil {
		return "", err
	}
	path := filepath.Join(dir(), "go2rtc.yaml")
	var b strings.Builder
	b.WriteString("# netductor-generated — bind to VPN/localhost only; do not expose publicly\n")
	b.WriteString("# go2rtc -c " + path + "\n")
	b.WriteString("api:\n  listen: \"127.0.0.1:1984\"\n")
	b.WriteString("rtsp:\n  listen: \"127.0.0.1:8554\"\n")
	b.WriteString("webrtc:\n  listen: \":8555\"\n  candidates: []\n")
	b.WriteString("streams:\n")
	for _, c := range ListCameras() {
		if !c.Enabled {
			continue
		}
		url := RTSPURL(c)
		if url == "" {
			b.WriteString(fmt.Sprintf("  # %s: missing RTSP secret/ip\n", c.ID))
			continue
		}
		name := c.ID
		if name == "" {
			name = c.Name
		}
		b.WriteString(fmt.Sprintf("  %s:\n    - \"%s\"\n", yamlKey(name), url))
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func yamlKey(s string) string {
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
	if s == "" {
		return "cam"
	}
	return s
}
