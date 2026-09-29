package notify

import (
	"fmt"
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

func alertsThreadID() string {
	return secret("telegram_alerts_thread_id")
}

func postTG(tok string, method string, vals url.Values) error {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/%s", tok, method)
	resp, err := http.PostForm(u, vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram HTTP %d", resp.StatusCode)
	}
	return nil
}

// Telegram sends via sendRichMessage when possible.
func Telegram(msg string) error {
	tok := secret("telegram_bot_token")
	chat := secret("telegram_admin_id")
	if tok == "" || chat == "" {
		return fmt.Errorf("telegram secrets not configured")
	}
	// try rich
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendRichMessage", tok)
	th := alertsThreadID()
	body := fmt.Sprintf(`{"chat_id":%s,"rich_message":{"html":%q}}`, chat, msg)
	if th != "" {
		body = fmt.Sprintf(`{"chat_id":%s,"message_thread_id":%s,"rich_message":{"html":%q}}`, chat, th, msg)
	}
	resp, err := http.Post(u, "application/json", strings.NewReader(body))
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
	}
	vals := url.Values{"chat_id": {chat}, "text": {msg}, "parse_mode": {"HTML"}}
	if th := alertsThreadID(); th != "" {
		vals.Set("message_thread_id", th)
	}
	return postTG(tok, "sendMessage", vals)
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
