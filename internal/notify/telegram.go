package notify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func secret(name string) string {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func postTG(tok string, method string, vals url.Values) error {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/%s", tok, method)
	resp, err := http.PostForm(u, vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(b, &wr)
	if resp.StatusCode >= 300 || !wr.OK {
		desc := wr.Description
		if desc == "" {
			desc = strings.TrimSpace(string(b))
		}
		return fmt.Errorf("telegram HTTP %d: %s", resp.StatusCode, desc)
	}
	return nil
}

// Telegram sends HTML to alerts channel if set, else admin DM (no topics).
func Telegram(msg string) error {
	return sendTelegramHTML(msg)
}
