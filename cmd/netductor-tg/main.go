package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/policy"
	ndver "github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
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
	// Bot API returns HTTP 200 with {"ok":false,"description":"..."} on logical errors.
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if json.Unmarshal(data, &wr) == nil && !wr.OK {
		desc := wr.Description
		if desc == "" {
			desc = string(data)
		}
		if len(desc) > 200 {
			desc = desc[:200] + "…"
		}
		return data, fmt.Errorf("telegram %s: %s", method, desc)
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
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// htmlNeedsRich is true when classic parse_mode=HTML would strip controls/tables.
func htmlNeedsRich(html string) bool {
	return strings.Contains(html, "tg-button") ||
		strings.Contains(html, "<table") ||
		strings.Contains(html, "<details") ||
		strings.Contains(html, "<tg-button")
}

// sendRich prefers Bot API 10.1+ sendRichMessage (native tables).
// Classic fallback is ONLY for plain text — never when body has in-table actions.
func messageIDFromTG(body []byte) int {
	var wr struct {
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	_ = json.Unmarshal(body, &wr)
	return wr.Result.MessageID
}

func sendRich(token string, chat int64, html string, kb map[string]any) {
	payload := map[string]any{
		"chat_id": chat,
		"rich_message": map[string]any{
			"html": html,
		},
	}
	if kb != nil {
		payload["reply_markup"] = kb
	}
	if body, err := apiPost(token, "sendRichMessage", payload); err == nil {
		if mid := messageIDFromTG(body); mid > 0 {
			notify.SaveHubMsg(chat, mid)
		}
		return
	} else {
		fmt.Fprintf(os.Stderr, "sendRichMessage: %v body=%s\n", err, truncate(string(body), 200))
		if htmlNeedsRich(html) {
			payloadRetry := map[string]any{"chat_id": chat, "rich_message": map[string]any{"html": html}}
			if b2, err2 := apiPost(token, "sendRichMessage", payloadRetry); err2 != nil {
				fmt.Fprintf(os.Stderr, "sendRichMessage retry: %v body=%s\n", err2, truncate(string(b2), 200))
			} else if mid := messageIDFromTG(b2); mid > 0 {
				notify.SaveHubMsg(chat, mid)
			}
			return
		}
	}
	payload2 := map[string]any{"chat_id": chat, "text": html, "parse_mode": "HTML"}
	if kb != nil {
		payload2["reply_markup"] = kb
	}
	if body, err := apiPost(token, "sendMessage", payload2); err == nil {
		if mid := messageIDFromTG(body); mid > 0 {
			notify.SaveHubMsg(chat, mid)
		}
	}
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
	body, err := apiPost(token, "editMessageText", payload)
	if err == nil {
		return nil
	}
	fmt.Fprintf(os.Stderr, "editMessageText rich: %v body=%s\n", err, truncate(string(body), 200))
	if htmlNeedsRich(html) {
		// Force caller (reply) to delete + sendRich — classic edit would kill tg-buttons.
		return fmt.Errorf("rich edit failed: %w", err)
	}
	payload2 := map[string]any{
		"chat_id": chat, "message_id": msgID, "text": html, "parse_mode": "HTML",
	}
	if kb != nil {
		payload2["reply_markup"] = kb
	}
	_, err = apiPost(token, "editMessageText", payload2)
	return err
}

var topicsLastReconcile time.Time

func sendHTML(token string, chat int64, text string, kb map[string]any) {
	sendRich(token, chat, text, kb)
}

// sendEphemeralHTML tries Bot API ephemeral params; on failure sends normal message
// and schedules deleteMessage after ttl (default 120s). Used for session tokens.
func sendEphemeralHTML(token string, chat int64, text string, kb map[string]any, ttlSec int) {
	if ttlSec <= 0 {
		ttlSec = 120
	}
	payload := map[string]any{
		"chat_id":    chat,
		"text":       text,
		"parse_mode": "HTML",
		"ephemeral_message_parameters": map[string]any{
			"replace_callback_query_message": false,
		},
	}
	if kb != nil {
		payload["reply_markup"] = kb
	}
	// Prefer rich ephemeral if supported
	rich := map[string]any{
		"chat_id":      chat,
		"rich_message": map[string]any{"html": text},
		"ephemeral_message_parameters": map[string]any{
			"replace_callback_query_message": false,
		},
	}
	if kb != nil {
		rich["reply_markup"] = kb
	}
	if body, err := apiPost(token, "sendRichMessage", rich); err == nil {
		if mid := messageIDFromTG(body); mid > 0 {
			go scheduleDelete(token, chat, mid, ttlSec)
		}
		return
	}
	if body, err := apiPost(token, "sendMessage", payload); err == nil {
		if mid := messageIDFromTG(body); mid > 0 {
			go scheduleDelete(token, chat, mid, ttlSec)
		}
		return
	}
	// last resort: normal send + delete
	sendRich(token, chat, text+"\n\n<i>⏱ auto-delete ~"+fmt.Sprintf("%d", ttlSec)+"s</i>", kb)
}

func scheduleDelete(token string, chat int64, msgID, ttlSec int) {
	time.Sleep(time.Duration(ttlSec) * time.Second)
	_, _ = apiPost(token, "deleteMessage", map[string]any{
		"chat_id": chat, "message_id": msgID,
	})
}

// sendRichWithPhoto sends Bot API 10.2+ rich message with in-body photo + tg-buttons.
// photoID is the media id referenced as tg://photo?id=<photoID> in html (e.g. "qr1").
func sendRichWithPhoto(token string, chat int64, html, photoPath, photoID string, kb map[string]any) (int, error) {
	if photoID == "" {
		photoID = "qr1"
	}
	f, err := os.Open(photoPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	rm := map[string]any{
		"html": html,
		"media": []map[string]any{
			{
				"id": photoID,
				"media": map[string]any{
					"type":  "photo",
					"media": "attach://" + photoID,
				},
			},
		},
	}
	rmJSON, err := json.Marshal(rm)
	if err != nil {
		return 0, err
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", strconv.FormatInt(chat, 10))
	_ = w.WriteField("rich_message", string(rmJSON))
	if kb != nil {
		jb, _ := json.Marshal(kb)
		_ = w.WriteField("reply_markup", string(jb))
	}
	part, err := w.CreateFormFile(photoID, filepath.Base(photoPath))
	if err != nil {
		return 0, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return 0, err
	}
	if err := w.Close(); err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/sendRichMessage", &buf)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	_ = json.Unmarshal(body, &wr)
	if !wr.OK {
		return 0, fmt.Errorf("sendRichMessage photo: %s", wr.Description)
	}
	return wr.Result.MessageID, nil
}

// editRichWithPhoto tries in-place edit of a rich message (same message_id).
// Falls back to caller if API rejects (e.g. message is plain text).
func editRichWithPhoto(token string, chat int64, msgID int, html, photoPath, photoID string, kb map[string]any) error {
	if msgID <= 0 {
		return fmt.Errorf("no message_id")
	}
	if photoID == "" {
		photoID = "qr1"
	}
	f, err := os.Open(photoPath)
	if err != nil {
		return err
	}
	defer f.Close()
	rm := map[string]any{
		"html": html,
		"media": []map[string]any{
			{"id": photoID, "media": map[string]any{"type": "photo", "media": "attach://" + photoID}},
		},
	}
	rmJSON, err := json.Marshal(rm)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", strconv.FormatInt(chat, 10))
	_ = w.WriteField("message_id", strconv.Itoa(msgID))
	_ = w.WriteField("rich_message", string(rmJSON))
	if kb != nil {
		jb, _ := json.Marshal(kb)
		_ = w.WriteField("reply_markup", string(jb))
	}
	part, err := w.CreateFormFile(photoID, filepath.Base(photoPath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	// Prefer editMessageText with rich_message (Bot API 10.x); some clients accept media attach here.
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/editMessageText", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(body, &wr)
	if !wr.OK {
		return fmt.Errorf("editMessageText rich: %s", wr.Description)
	}
	return nil
}

// replyRichWithPhoto: try in-place edit; on failure send NEW first, delete old only after success.
// Deleting before send caused empty cards when sendRichWithPhoto failed (rapid VLESS↔Sub).
func replyRichWithPhoto(token string, chat int64, msgID int, html, photoPath, photoID string, kb map[string]any) {
	if msgID > 0 {
		if err := editRichWithPhoto(token, chat, msgID, html, photoPath, photoID, kb); err == nil {
			return
		} else {
			fmt.Fprintln(os.Stderr, "editRichWithPhoto:", err)
		}
	}
	if newID, err := sendRichWithPhoto(token, chat, html, photoPath, photoID, kb); err == nil {
		if msgID > 0 {
			_ = deleteMessage(token, chat, msgID)
		}
		_ = newID
		return
	} else {
		fmt.Fprintln(os.Stderr, "sendRichWithPhoto:", err)
	}
	cap := stripHTMLApprox(html)
	if len(cap) > 900 {
		cap = cap[:900] + "…"
	}
	if err := sendPhotoFile(token, chat, photoPath, cap, kb); err != nil {
		fmt.Fprintln(os.Stderr, "sendPhotoFile:", err)
		// Always keep the link text visible
		if msgID > 0 {
			if err2 := editHTML(token, chat, msgID, html, kb); err2 == nil {
				return
			}
			_ = deleteMessage(token, chat, msgID)
		}
		sendRich(token, chat, html, kb)
		return
	}
	if msgID > 0 {
		_ = deleteMessage(token, chat, msgID)
	}
}

func stripHTMLApprox(s string) string {
	out := make([]rune, 0, len(s))
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			out = append(out, r)
		}
	}
	return strings.TrimSpace(string(out))
}

func editHTML(token string, chat int64, msgID int, text string, kb map[string]any) error {
	return editRich(token, chat, msgID, text, kb)
}

func reply(token string, chat int64, msgID int, text string, kb map[string]any) {
	// Singleton Control hub: prefer editing stored hub message over spawning new menus.
	h := notify.LoadHubMsg()
	target := msgID
	if h.ChatID == chat && h.MessageID > 0 {
		target = h.MessageID
	}
	if target > 0 {
		if err := editHTML(token, chat, target, text, kb); err == nil {
			notify.SaveHubMsg(chat, target)
			return
		}
		fmt.Fprintln(os.Stderr, "reply edit failed, replace hub:", target)
		// Message gone (chat clear / delete for all) or uneditable — drop stale id and send fresh.
		notify.ClearHubMsg()
		sendHTML(token, chat, text, kb)
		if target > 0 {
			_ = deleteMessage(token, chat, target)
		}
		if msgID > 0 && msgID != target {
			_ = deleteMessage(token, chat, msgID)
		}
		return
	}
	sendHTML(token, chat, text, kb)
}

// forceHub clears singleton state, tries to delete the old hub message, and always sends a new menu.
// Use after user deleted hub "for me", cleared chat, or Topics recreate.
func forceHub(token string, chat int64, text string, kb map[string]any) {
	h := notify.LoadHubMsg()
	notify.ClearHubMsg()
	if h.MessageID > 0 && (h.ChatID == 0 || h.ChatID == chat) {
		_ = deleteMessage(token, chat, h.MessageID)
	}
	sendHTML(token, chat, text, kb)
}

func sendDocumentFile(token string, chat int64, path, caption string, kb map[string]any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", strconv.FormatInt(chat, 10))
	if caption != "" {
		_ = w.WriteField("caption", caption)
		_ = w.WriteField("parse_mode", "HTML")
	}
	if kb != nil {
		if jb, err := json.Marshal(kb); err == nil {
			_ = w.WriteField("reply_markup", string(jb))
		}
	}
	part, err := w.CreateFormFile("document", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/sendDocument", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	var wr struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(body, &wr)
	if !wr.OK {
		return fmt.Errorf("sendDocument: %s", wr.Description)
	}
	return nil
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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
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

// shareURIFrom keeps only importable URI lines (vless/ss/trojan).
func shareURIFrom(raw string) string {
	var lines []string
	for _, ln := range strings.Split(raw, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		low := strings.ToLower(ln)
		if strings.HasPrefix(low, "vless://") || strings.HasPrefix(low, "ss://") ||
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
	_ = lines
	return navKeyboard("m:cat:routers", parentRouters())
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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Printf("netductor-tg %s\n", ndver.Release)
			return
		case "help", "-h", "--help":
			fmt.Println("netductor-tg [version] — Telegram bot for netductor (systemd)")
			return
		}
	}

	ndconfig.Load()
	// Same as API/CLI: policy toggles must rewrite sing-box routes.
	policy.ApplyHook = vpn.ApplyAccessPolicies

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
	notify.MigrateAwayFromTopics()

	for {
		v := url.Values{}
		v.Set("timeout", "30")
		v.Set("offset", strconv.Itoa(offset))
		v.Set("allowed_updates", `["message","callback_query","channel_post","my_chat_member"]`)
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
			if u.MyChatMember != nil {
				rememberFromChatMember(u.MyChatMember)
				continue
			}
			if u.ChannelPost != nil {
				rememberFromMessage(u.ChannelPost)
				continue
			}
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
							sendHTML(token, admin, T("claim_ok"), mainKeyboard())
						} else {
							sendHTML(token, u.Message.Chat.ID, T("claim_fail"), nil)
							admin = 0
						}
						continue
					}
					sendHTML(token, u.Message.Chat.ID, T("operator_not_cfg"), nil)
					continue
				}
				handleMessage(token, u.Message, admin)
			}
		}
	}
}

func maskTokenKV(s string) string {
	// approved id token=abcdef... -> token=abcd…(hidden)
	const key = "token="
	i := strings.Index(s, key)
	if i < 0 {
		return s
	}
	rest := s[i+len(key):]
	end := len(rest)
	for j, c := range rest {
		if c == ' ' || c == '\n' || c == '\t' {
			end = j
			break
		}
	}
	tok := rest[:end]
	if len(tok) <= 8 {
		return s[:i+len(key)] + "****" + rest[end:]
	}
	return s[:i+len(key)] + tok[:4] + "…" + tok[len(tok)-2:] + " (full in CLI only)" + rest[end:]
}
