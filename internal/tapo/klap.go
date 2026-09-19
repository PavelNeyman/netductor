package tapo

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// KLAP session (python-kasa KlapTransport / V2), used by newer Tapo firmware.
type klapSession struct {
	localSeed  []byte
	remoteSeed []byte
	authHash   []byte
	key        []byte
	iv         []byte // 12 bytes
	seq        int32
	sig        []byte // 28 bytes
	cookie     string
	baseURL    string // e.g. http://ip/app or https://ip:443/app
	http       *http.Client
	version    int // 1 or 2
}

func md5b(b []byte) []byte {
	h := md5.Sum(b)
	return h[:]
}

func sha1b(b []byte) []byte {
	h := sha1.Sum(b)
	return h[:]
}

func sha256b(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

func authHashV1(user, pass string) []byte {
	return md5b(append(md5b([]byte(user)), md5b([]byte(pass))...))
}

func authHashV2(user, pass string) []byte {
	return sha256b(append(sha1b([]byte(user)), sha1b([]byte(pass))...))
}

func handshake1Hash(v int, local, remote, auth []byte) []byte {
	if v == 2 {
		return sha256b(append(append(local, remote...), auth...))
	}
	return sha256b(append(local, auth...))
}

func handshake2Hash(v int, local, remote, auth []byte) []byte {
	if v == 2 {
		return sha256b(append(append(remote, local...), auth...))
	}
	return sha256b(append(remote, auth...))
}

func packSeq(seq int32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(seq))
	return b
}

func newKlapHTTP() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: 12 * time.Second,
		Jar:     jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

// probeKLAP returns true if device answers KLAP-style on http://host:port
func probeKLAP(host string, port int) bool {
	if port <= 0 {
		port = 80
	}
	u := fmt.Sprintf("http://%s:%d/", host, port)
	cl := &http.Client{Timeout: 3 * time.Second}
	resp, err := cl.Get(u)
	if err != nil {
		// try bare host port 80
		if port != 80 {
			return probeKLAP(host, 80)
		}
		return false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return strings.Contains(string(b), "200 OK") || resp.StatusCode == 200
}

func (c *Client) loginKLAP() error {
	bases := []string{
		fmt.Sprintf("http://%s/app", c.Host),
		fmt.Sprintf("http://%s:80/app", c.Host),
		fmt.Sprintf("https://%s/app", c.Host),
		fmt.Sprintf("https://%s:443/app", c.Host),
	}
	if c.Port > 0 && c.Port != 80 && c.Port != 443 {
		bases = append([]string{fmt.Sprintf("http://%s:%d/app", c.Host, c.Port)}, bases...)
		bases = append([]string{fmt.Sprintf("https://%s:%d/app", c.Host, c.Port)}, bases...)
	}
	var last error
	for _, base := range bases {
		for _, ver := range []int{2, 1} {
			s, err := klapHandshake(base, c.User, c.Password, ver)
			if err == nil {
				c.klap = s
				c.secure = true
				return nil
			}
			last = err
		}
	}
	if last == nil {
		last = fmt.Errorf("klap failed")
	}
	return last
}

func klapHandshake(base, user, pass string, version int) (*klapSession, error) {
	cl := newKlapHTTP()
	local := make([]byte, 16)
	_, _ = rand.Read(local)
	auth := authHashV1(user, pass)
	if version == 2 {
		auth = authHashV2(user, pass)
	}
	// handshake1
	u1 := strings.TrimRight(base, "/") + "/handshake1"
	req, err := http.NewRequest(http.MethodPost, u1, bytes.NewReader(local))
	if err != nil {
		return nil, err
	}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("handshake1 status %d", resp.StatusCode)
	}
	if len(body) < 48 {
		return nil, fmt.Errorf("handshake1 short body %d", len(body))
	}
	remote := body[0:16]
	serverHash := body[16:48]
	localHash := handshake1Hash(version, local, remote, auth)
	if !bytes.Equal(localHash, serverHash) {
		return nil, fmt.Errorf("handshake1 auth mismatch (v%d)", version)
	}
	// session cookie
	cookie := ""
	for _, c := range resp.Cookies() {
		if c.Name == "TP_SESSIONID" {
			cookie = c.Value
		}
	}
	if cookie == "" {
		// try Set-Cookie parse from jar
		if u, err := url.Parse(base); err == nil {
			for _, c := range cl.Jar.Cookies(u) {
				if c.Name == "TP_SESSIONID" {
					cookie = c.Value
				}
			}
		}
	}
	// handshake2
	u2 := strings.TrimRight(base, "/") + "/handshake2"
	payload := handshake2Hash(version, local, remote, auth)
	req2, err := http.NewRequest(http.MethodPost, u2, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if cookie != "" {
		req2.AddCookie(&http.Cookie{Name: "TP_SESSIONID", Value: cookie})
	}
	resp2, err := cl.Do(req2)
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		return nil, fmt.Errorf("handshake2 status %d", resp2.StatusCode)
	}
	s := &klapSession{
		localSeed:  local,
		remoteSeed: remote,
		authHash:   auth,
		cookie:     cookie,
		baseURL:    strings.TrimRight(base, "/"),
		http:       cl,
		version:    version,
	}
	s.key = sha256b(append(append(append([]byte("lsk"), local...), remote...), auth...))[:16]
	fulliv := sha256b(append(append(append([]byte("iv"), local...), remote...), auth...))
	s.iv = fulliv[:12]
	s.seq = int32(binary.BigEndian.Uint32(fulliv[12:16]))
	s.sig = sha256b(append(append(append([]byte("ldk"), local...), remote...), auth...))[:28]
	return s, nil
}

func (s *klapSession) encrypt(msg []byte) (payload []byte, seq int32, err error) {
	s.seq++
	seq = s.seq
	ivSeq := append(append([]byte{}, s.iv...), packSeq(seq)...)
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, 0, err
	}
	plain := pkcs7Pad(msg, block.BlockSize())
	ct := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, ivSeq).CryptBlocks(ct, plain)
	sig := sha256b(append(append(s.sig, packSeq(seq)...), ct...))
	return append(sig, ct...), seq, nil
}

func (s *klapSession) decrypt(msg []byte) ([]byte, error) {
	if len(msg) < 32 {
		return nil, fmt.Errorf("short response")
	}
	// response is ciphertext only after 32-byte sig (same as kasa)
	ct := msg[32:]
	ivSeq := append(append([]byte{}, s.iv...), packSeq(s.seq)...)
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	if len(ct)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("bad ct len")
	}
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, ivSeq).CryptBlocks(out, ct)
	return pkcs7Unpad(out)
}

func (s *klapSession) send(req map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	payload, seq, err := s.encrypt(raw)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/request?seq=%d", s.baseURL, seq)
	httpReq, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if s.cookie != "" {
		httpReq.AddCookie(&http.Cookie{Name: "TP_SESSIONID", Value: s.cookie})
	}
	resp, err := s.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("klap request status %d", resp.StatusCode)
	}
	pt, err := s.decrypt(body)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, fmt.Errorf("klap json: %w body=%s", err, string(pt))
	}
	return out, nil
}
