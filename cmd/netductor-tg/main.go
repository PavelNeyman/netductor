package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func claimAdmin(chatID int64) {
	// Production: admin id must be pre-provisioned (file or NETDUCTOR_TG_ADMIN).
	// First-message claim only if NETDUCTOR_TG_CLAIM_FIRST=1.
	if os.Getenv("NETDUCTOR_TG_CLAIM_FIRST") != "1" {
		return
	}
	if readOptional(chatFile) != "" {
		return
	}
	_ = os.MkdirAll("/etc/netductor/secrets", 0o700)
	_ = os.WriteFile(chatFile, []byte(fmt.Sprintf("%d\n", chatID)), 0o600)
}

func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(b))
}

func readOptional(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func apiPost(token, method string, payload any) ([]byte, error) {
	body, _ := json.Marshal(payload)
	resp, err := http.Post("https://api.telegram.org/bot"+token+"/"+method, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		msg := string(data)
		if len(msg) > 200 {
			msg = msg[:200] + "…"
		}
		return data, fmt.Errorf("telegram %s HTTP %d: %s", method, resp.StatusCode, msg)
	}
	return data, nil
}

func apiGet(token, method string, v url.Values) ([]byte, error) {
	u := "https://api.telegram.org/bot" + token + "/" + method + "?" + v.Encode()
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}


// sendRich prefers Bot API 10.1+ sendRichMessage (native tables); falls back to HTML sendMessage.
func sendRich(token string, chat int64, html string, kb map[string]any) {
	payload := map[string]any{
		"chat_id": chat,
		"rich_message": map[string]any{
			"html": html,
		},
	}
	// Prefer in-body <tg-button-row>; still attach reply_markup as fallback for older clients.
	if kb != nil {
		payload["reply_markup"] = kb
	}
	if body, err := apiPost(token, "sendRichMessage", payload); err == nil {
		return
	} else {
		fmt.Fprintf(os.Stderr, "sendRichMessage: %v body=%s\n", err, truncate(string(body), 200))
	}
	// fallback classic (strips unknown tags)
	payload2 := map[string]any{"chat_id": chat, "text": html, "parse_mode": "HTML"}
	if kb != nil {
		payload2["reply_markup"] = kb
	}
	_, _ = apiPost(token, "sendMessage", payload2)
}

func editRich(token string, chat int64, msgID int, html string, kb map[string]any) error {
	payload := map[string]any{
		"chat_id":    chat,
		"message_id": msgID,
		"rich_message": map[string]any{
			"html": html,
		},
	}
	if kb != nil {
		payload["reply_markup"] = kb
	}
	if _, err := apiPost(token, "editMessageText", payload); err == nil {
		return nil
	}
	// fallback classic HTML
	payload2 := map[string]any{
		"chat_id": chat, "message_id": msgID, "text": html, "parse_mode": "HTML",
	}
	if kb != nil {
		payload2["reply_markup"] = kb
	}
	_, err := apiPost(token, "editMessageText", payload2)
	return err
}


func sendHTML(token string, chat int64, text string, kb map[string]any) {
	sendRich(token, chat, text, kb)
}

func editHTML(token string, chat int64, msgID int, text string, kb map[string]any) error {
	return editRich(token, chat, msgID, text, kb)
}

func reply(token string, chat int64, msgID int, text string, kb map[string]any) {
	if msgID > 0 {
		if err := editHTML(token, chat, msgID, text, kb); err == nil {
			return
		}
		// Photo (or other non-text) message: cannot editMessageText — replace in place.
		_ = deleteMessage(token, chat, msgID)
	}
	sendHTML(token, chat, text, kb)
}


func sendPhotoFile(token string, chat int64, path, caption string, kb map[string]any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// Telegram caption max 1024
	if len(caption) > 1000 {
		caption = caption[:1000] + "…"
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", strconv.FormatInt(chat, 10))
	if caption != "" {
		_ = w.WriteField("caption", caption)
		_ = w.WriteField("parse_mode", "HTML")
	}
	if kb != nil {
		jb, _ := json.Marshal(kb)
		_ = w.WriteField("reply_markup", string(jb))
	}
	part, err := w.CreateFormFile("photo", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/sendPhoto", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendPhoto %s: %s", resp.Status, string(body))
	}
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(body, &wr)
	if !wr.OK {
		return fmt.Errorf("sendPhoto: %s", wr.Description)
	}
	return nil
}


func deleteMessage(token string, chat int64, msgID int) error {
	return apiPOST(token, "deleteMessage", map[string]any{
		"chat_id": chat, "message_id": msgID,
	})
}

func apiPOST(token, method string, fields map[string]any) error {
	form := url.Values{}
	for k, v := range fields {
		switch t := v.(type) {
		case int:
			form.Set(k, strconv.Itoa(t))
		case int64:
			form.Set(k, strconv.FormatInt(t, 10))
		case string:
			form.Set(k, t)
		default:
			b, _ := json.Marshal(t)
			form.Set(k, string(b))
		}
	}
	resp, err := http.PostForm("https://api.telegram.org/bot"+token+"/"+method, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(body, &wr)
	if !wr.OK {
		return fmt.Errorf("%s: %s", method, wr.Description)
	}
	return nil
}

func editPhotoFile(token string, chat int64, msgID int, path, caption string, kb map[string]any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(caption) > 1000 {
		caption = caption[:1000] + "…"
	}
	media := map[string]any{
		"type": "photo", "media": "attach://photo", "parse_mode": "HTML",
	}
	if caption != "" {
		media["caption"] = caption
	}
	mb, _ := json.Marshal(media)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", strconv.FormatInt(chat, 10))
	_ = w.WriteField("message_id", strconv.Itoa(msgID))
	_ = w.WriteField("media", string(mb))
	if kb != nil {
		jb, _ := json.Marshal(kb)
		_ = w.WriteField("reply_markup", string(jb))
	}
	part, err := w.CreateFormFile("photo", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/editMessageMedia", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(body, &wr)
	if !wr.OK {
		return fmt.Errorf("editMessageMedia: %s", wr.Description)
	}
	return nil
}

func answerCallback(token, id string) {
	_, _ = apiPost(token, "answerCallbackQuery", map[string]any{"callback_query_id": id})
}

func answerCallbackText(token, id, text string) {
	_, _ = apiPost(token, "answerCallbackQuery", map[string]any{
		"callback_query_id": id,
		"text":              text,
		"show_alert":        false,
	})
}

func formatVPNList(s string) string {
	if getLang() == "en" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		p := strings.Split(line, "\t")
		if len(p) >= 2 {
			if p[1] == "on" {
				p[1] = "вкл"
			} else if p[1] == "off" {
				p[1] = "выкл"
			}
			lines[i] = strings.Join(p, "\t")
		}
	}
	return strings.Join(lines, "\n")
}

func netductorBin() string {
	for _, p := range []string{"/usr/local/bin/netductor", "/usr/bin/netductor"} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return "netductor"
}

func runVPN(args ...string) string {
	full := append([]string{"vpn"}, args...)
	cmd := exec.Command(netductorBin(), full...)
	// stdout only — CLI writes diagnostics (e.g. "via relay:...") to stderr
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return strings.TrimSpace(string(ee.Stderr) + "\n" + err.Error())
		}
		msg := string(out)
		if msg != "" {
			msg += "\n"
		}
		return strings.TrimSpace(msg + err.Error())
	}
	return string(out)
}

// shareURIFrom keeps only importable URI lines (vless/hysteria2/ss/trojan).
func shareURIFrom(raw string) string {
	var lines []string
	for _, ln := range strings.Split(raw, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		low := strings.ToLower(ln)
		if strings.HasPrefix(low, "vless://") || strings.HasPrefix(low, "hysteria2://") ||
			strings.HasPrefix(low, "hy2://") || strings.HasPrefix(low, "ss://") ||
			strings.HasPrefix(low, "trojan://") {
			lines = append(lines, ln)
		}
	}
	return strings.Join(lines, "\n")
}

func runND(args ...string) string {
	cmd := exec.Command(netductorBin(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if msg != "" {
			msg += string([]byte{10})
		}
		return strings.TrimSpace(msg + err.Error())
	}
	return string(out)
}

func statusInline() string {
	var b strings.Builder
	host, _ := os.Hostname()
	ru := getLang() != "en"
	act, inact := "active", "inactive"
	if ru {
		b.WriteString("Status Netductor host ")
		b.WriteString(host)
		b.WriteByte(10)
		act, inact = "active-ru", "inactive-ru"
	} else {
		b.WriteString("Netductor status on ")
		b.WriteString(host)
		b.WriteByte(10)
	}
	// localized labels
	if ru {
		act, inact = "активен", "неактивен"
		// rewrite header properly
		var hdr strings.Builder
		hdr.WriteString("Статус Netductor на ")
		hdr.WriteString(host)
		hdr.WriteByte(10)
		b.Reset()
		b.WriteString(hdr.String())
	}
	mapSt := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "active" {
			return act
		}
		if s == "" || (ru && s != "active") {
			return inact
		}
		return s
	}
	for _, u := range []string{"sing-box", "blocky", "netductor-api", "netductor-api", "netductor-telegram-bot", "netductor-telegram-bot"} {
		out, _ := exec.Command("systemctl", "is-active", u).Output()
		b.WriteString(u + ": " + mapSt(string(out)))
		b.WriteByte(10)
	}
	if out, err := exec.Command("systemctl", "is-active", "netductor-metrics.timer").Output(); err == nil {
		b.WriteString("metrics-timer: " + mapSt(string(out)))
		b.WriteByte(10)
	}
	if out, err := exec.Command("docker", "ps", "--format", "{{.Names}}").Output(); err == nil {
		names := strings.TrimSpace(string(out))
		names = strings.ReplaceAll(names, string([]byte{10}), " ")
		b.WriteString("docker: " + names)
		b.WriteByte(10)
	}
	// identity
	if ipb, err := os.ReadFile("/etc/netductor/public_ip"); err == nil {
		ip := strings.TrimSpace(string(ipb))
		if ip != "" {
			if ru {
				b.WriteString("IP: ")
			} else {
				b.WriteString("IP: ")
			}
			b.WriteString(ip)
			b.WriteByte(10)
		}
	}
	if idb, err := os.ReadFile("/etc/netductor/node_id"); err == nil {
		id := strings.TrimSpace(string(idb))
		if id != "" {
			b.WriteString("node: ")
			b.WriteString(id)
			b.WriteByte(10)
		}
	}
	b.WriteByte(10)
	if ru {
		b.WriteString("Пользователи VPN:")
	} else {
		b.WriteString("VPN users:")
	}
	b.WriteByte(10)
	b.WriteString(formatVPNList(runVPN("list")))
	return b.String()
}

func statusText() string {
	env := append(os.Environ(), "FRESHVPS_LANG="+getLang())
	if st, err := os.Stat(statusSh); err == nil && st.Mode()&0111 != 0 {
		cmd := exec.Command(statusSh)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err == nil && len(bytes.TrimSpace(out)) > 0 {
			return string(out)
		}
	}
	if _, err := os.Stat(statusSh); err == nil {
		cmd := exec.Command("bash", statusSh)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err == nil && len(bytes.TrimSpace(out)) > 0 {
			return string(out)
		}
	}
	return statusInline()
}


func readyText() string {
	b, err := os.ReadFile("/etc/netductor/READY.txt")
	if err != nil {
		return ""
	}
	text := string(b)
	ru := getLang() != "en"
	// Prefer language section if bilingual file present
	if strings.Contains(text, "=== RU ===") && strings.Contains(text, "=== EN ===") {
		var part string
		if ru {
			i := strings.Index(text, "=== RU ===")
			j := strings.Index(text, "=== EN ===")
			if i >= 0 && j > i {
				part = strings.TrimSpace(text[i+len("=== RU ==="):j])
			}
		} else {
			i := strings.Index(text, "=== EN ===")
			if i >= 0 {
				part = strings.TrimSpace(text[i+len("=== EN ==="):])
			}
		}
		if part != "" {
			return part
		}
	}
	return strings.TrimSpace(text)
}


func routersText() string {
	out, err := exec.Command(netductorBin(), "edge", "list").CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out) + " " + err.Error())
	}
	return strings.TrimSpace(string(out))
}

func pendingText() string {
	out, err := exec.Command(netductorBin(), "edge", "pending").CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

func templatesText() string {
	out, err := exec.Command(netductorBin(), "edge", "templates").CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

func pendingKeyboard(lines string) map[string]any {
	rows := [][]map[string]any{}
	for _, line := range strings.Split(lines, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		id := strings.Fields(line)[0]
		rows = append(rows, []map[string]any{
			btn("✅ "+id, "e:appr:"+id, "success"),
			btn("🚫 "+id, "e:deny:"+id, "danger"),
		})
	}
	rows = append(rows, []map[string]any{btn(T("back_routers"), "m:cat:routers", "primary"), btn(T("main_menu"), "m:menu", "")})
	return map[string]any{"inline_keyboard": rows}
}

func menuText() string {
	return T("menu_title") + "\n\n" + T("menu_hint2")
}

func helpText() string {
	return T("help_title") + "\n\n" + T("help_body")
}

var chatState = map[int64]string{}
var chatExtra = map[int64]string{}

func setState(chat int64, st, extra string) {
	if st == "" {
		delete(chatState, chat)
		delete(chatExtra, chat)
		return
	}
	chatState[chat] = st
	chatExtra[chat] = extra
}

func setBotCommands(token string) {
	// descriptions in both langs is limited; use English short + user can /lang
	cmds := []map[string]string{
		{"command": "menu", "description": "Menu / Меню"},
		{"command": "status", "description": "Status / Статус"},
		{"command": "vpn_list", "description": "VPN users"},
		{"command": "lang", "description": "Language / Язык"},
		{"command": "help", "description": "Help / Справка"},
	}
	_, _ = apiPost(token, "setMyCommands", map[string]any{"commands": cmds})
}

func main() {
	token := mustRead(tokenFile)
	if token == "" {
		fmt.Fprintln(os.Stderr, "telegram_bot_token empty")
		os.Exit(1)
	}
	adminStr := readOptional(chatFile)
	if adminStr == "" {
		adminStr = strings.TrimSpace(os.Getenv("NETDUCTOR_TG_ADMIN"))
	}
	admin, _ := strconv.ParseInt(adminStr, 10, 64)
	if admin == 0 {
		fmt.Fprintln(os.Stderr, "telegram_admin_id empty — set file or NETDUCTOR_TG_ADMIN (CLAIM_FIRST=1 only for bootstrap)")
	}
	// default language ru for this project
	if _, err := os.Stat(langFile); err != nil {
		setLang("ru")
	}
	setBotCommands(token)
	offset := 0
	for {
		v := url.Values{}
		v.Set("timeout", "30")
		v.Set("offset", strconv.Itoa(offset))
		v.Set("allowed_updates", `["message","callback_query"]`)
		body, err := apiGet(token, "getUpdates", v)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		var wrap struct {
			OK     bool              `json:"ok"`
			Result []json.RawMessage `json:"result"`
		}
		if json.Unmarshal(body, &wrap) != nil || !wrap.OK {
			time.Sleep(2 * time.Second)
			continue
		}
		for _, raw := range wrap.Result {
			var u update
			if json.Unmarshal(raw, &u) != nil {
				continue
			}
			offset = u.UpdateID + 1
			if u.CallbackQuery != nil {
				if admin == 0 {
					continue
				}
				handleCallback(token, u.CallbackQuery, admin)
				continue
			}
			if u.Message != nil {
				if admin == 0 {
					txt := strings.TrimSpace(u.Message.Text)
					// Only claim if explicitly allowed; never open bot to first random chatter.
					if os.Getenv("NETDUCTOR_TG_CLAIM_FIRST") == "1" && (txt == "/start" || strings.HasPrefix(txt, "/start ")) {
						admin = u.Message.Chat.ID
						claimAdmin(admin)
						if readOptional(chatFile) != "" {
							sendHTML(token, admin, "✅ Admin claimed for this chat. Use /menu", mainKeyboard())
						} else {
							sendHTML(token, u.Message.Chat.ID, "Claim failed: write /etc/netductor/secrets/telegram_admin_id", nil)
							admin = 0
						}
						continue
					}
					sendHTML(token, u.Message.Chat.ID, "Operator not configured. Set telegram_admin_id or NETDUCTOR_TG_ADMIN on the VPS.", nil)
					continue
				}
				handleMessage(token, u.Message, admin)
			}
		}
	}
}
