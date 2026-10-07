package notify

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// SendDocument uploads a local file to alerts chat (channel or admin).
// Best-effort; returns error if Telegram rejects (size limits etc.).
func SendDocument(path, caption string) error {
	tok := secret("telegram_bot_token")
	chat := secret("telegram_admin_id")
	if ac := AlertsChatID(); ac != "" {
		chat = ac
	}
	if tok == "" || chat == "" {
		return fmt.Errorf("telegram secrets not configured")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, _ := f.Stat()
	if st != nil && st.Size() > 45*1024*1024 {
		return fmt.Errorf("file too large for telegram (%d bytes)", st.Size())
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", chat)
	if caption != "" {
		_ = w.WriteField("caption", caption)
		_ = w.WriteField("parse_mode", "HTML")
	}
	part, err := w.CreateFormFile("document", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	_ = w.Close()
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", tok)
	resp, err := http.Post(u, w.FormDataContentType(), &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendDocument HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
