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

func htmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
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

// TelegramCmd reports a finished node command with a small table.
func TelegramCmd(node, cmd string, ok bool, log string) error {
	mark := "✅"
	if !ok {
		mark = "❌"
	}
	if len(log) > 800 {
		log = log[len(log)-800:]
	}
	msg := fmt.Sprintf(
		"%s <b>%s</b>\n\n<table bordered striped>\n<tr><th>field</th><th>value</th></tr>\n<tr><td>node</td><td>%s</td></tr>\n<tr><td>cmd</td><td>%s</td></tr>\n<tr><td>ok</td><td>%v</td></tr>\n</table>\n<details><summary>log</summary>\n<pre>%s</pre>\n</details>",
		mark, htmlEsc(cmd), htmlEsc(node), htmlEsc(cmd), ok, htmlEsc(log),
	)
	return Telegram(msg)
}
