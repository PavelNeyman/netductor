// Package tapo is a minimal Go port of JurajNyiri/pytapo secure local control
// (same protocol as HomeAssistant-Tapo-Control) for Tapo C200-class cameras.
// Scope: login (secure + legacy), motorMove, day/night, privacy (lens mask).
package tapo

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type hashMethod int

const (
	hashMD5 hashMethod = iota
	hashSHA256
)

// Client holds a short-lived session to a Tapo camera.
type Client struct {
	Host     string
	User     string
	Password string
	Port     int // control port, default 443

	http *http.Client
	stok string
	seq  int
	lsk  []byte
	ivb  []byte
	cnonce string
	hash   hashMethod
	secure bool
	klap   *klapSession
	hashedMD5    string
	hashedSHA256 string
}

// New creates a client (does not login yet).
func New(host, user, password string) *Client {
	return &Client{
		Host:         host,
		User:         user,
		Password:     password,
		Port:         443,
		http: &http.Client{
			Timeout: 12 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // local camera self-signed
			},
		},
		hashedMD5:    strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(password)))),
		hashedSHA256: strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(password)))),
	}
}

func (c *Client) controlHost() string {
	if c.Port > 0 && c.Port != 443 {
		return fmt.Sprintf("%s:%d", c.Host, c.Port)
	}
	return c.Host
}

func nonce8() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func (c *Client) hashedPassword() string {
	if c.hash == hashSHA256 {
		return c.hashedSHA256
	}
	return c.hashedMD5
}

// Login performs stok handshake (secure encrypt_type 3 or legacy hashed password).
// If the camera speaks KLAP (newer FW), uses KLAP instead.
func (c *Client) Login() error {
	if probeKLAP(c.Host, c.Port) || probeKLAP(c.Host, 80) {
		if err := c.loginKLAP(); err == nil {
			return nil
		}
		// fall through to classic
	}
	c.secure = c.probeSecure()
	c.cnonce = nonce8()
	url := "https://" + c.controlHost()
	var body map[string]any
	if c.secure {
		body = map[string]any{
			"method": "login",
			"params": map[string]any{
				"cnonce":       c.cnonce,
				"encrypt_type": "3",
				"username":     c.User,
			},
		}
	} else {
		body = map[string]any{
			"method": "login",
			"params": map[string]any{
				"hashed":   true,
				"password": c.hashedMD5,
				"username": c.User,
			},
		}
	}
	res, err := c.postJSON(url, body, nil)
	if err != nil {
		if e2 := c.loginKLAP(); e2 == nil {
			return nil
		}
		return err
	}
	if c.secure {
		data, _ := res["result"].(map[string]any)
		inner, _ := data["data"].(map[string]any)
		nonce, _ := inner["nonce"].(string)
		devConf, _ := inner["device_confirm"].(string)
		if nonce == "" || devConf == "" {
			return fmt.Errorf("tapo: secure login missing nonce/confirm: %v", res)
		}
		if !c.validateConfirm(nonce, devConf) {
			return fmt.Errorf("tapo: device_confirm mismatch (wrong password or Third-Party Compatibility off)")
		}
		digest := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(c.hashedPassword()+c.cnonce+nonce))))
		digestPasswd := digest + c.cnonce + nonce
		body2 := map[string]any{
			"method": "login",
			"params": map[string]any{
				"cnonce":        c.cnonce,
				"encrypt_type":  "3",
				"digest_passwd": digestPasswd,
				"username":      c.User,
			},
		}
		res2, err := c.postJSON(url, body2, nil)
		if err != nil {
			return err
		}
		r2, _ := res2["result"].(map[string]any)
		stok, _ := r2["stok"].(string)
		if stok == "" {
			if e2 := c.loginKLAP(); e2 == nil {
				return nil
			}
			return fmt.Errorf("tapo: no stok after digest login: %v", res2)
		}
		c.stok = stok
		if ss, ok := r2["start_seq"].(float64); ok {
			c.seq = int(ss)
		}
		c.lsk = c.token("lsk", nonce)
		c.ivb = c.token("ivb", nonce)
		return nil
	}
	// legacy
	c.hash = hashMD5
	r, _ := res["result"].(map[string]any)
	stok, _ := r["stok"].(string)
	if stok == "" {
		if e2 := c.loginKLAP(); e2 == nil {
			return nil
		}
		return fmt.Errorf("tapo: legacy login failed: %v", res)
	}
	c.stok = stok
	return nil
}

func (c *Client) probeSecure() bool {
	cnonce := nonce8()
	body := map[string]any{
		"method": "login",
		"params": map[string]any{
			"encrypt_type": "3",
			"username":     c.User,
			"cnonce":       cnonce,
		},
	}
	res, err := c.postJSON("https://"+c.controlHost(), body, nil)
	if err != nil {
		return false
	}
	// error_code -40413 + encrypt_type contains 3
	ec, _ := res["error_code"].(float64)
	if int(ec) != -40413 {
		return false
	}
	data, _ := res["result"].(map[string]any)
	inner, _ := data["data"].(map[string]any)
	et, _ := inner["encrypt_type"].(string)
	if et == "" {
		// sometimes array
		if arr, ok := inner["encrypt_type"].([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok && s == "3" {
					return true
				}
			}
		}
		return false
	}
	return strings.Contains(et, "3")
}

func (c *Client) validateConfirm(nonce, deviceConfirm string) bool {
	sha := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(c.cnonce+c.hashedSHA256+nonce))))
	md := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(c.cnonce+c.hashedMD5+nonce))))
	if deviceConfirm == sha+nonce+c.cnonce {
		c.hash = hashSHA256
		return true
	}
	if deviceConfirm == md+nonce+c.cnonce {
		c.hash = hashMD5
		return true
	}
	return false
}

func (c *Client) token(tokenType, nonce string) []byte {
	hashedKey := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(c.cnonce+c.hashedPassword()+nonce))))
	sum := sha256.Sum256([]byte(tokenType + c.cnonce + nonce + hashedKey))
	return sum[:16]
}

func pkcs7Pad(b []byte, block int) []byte {
	pad := block - len(b)%block
	return append(b, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func pkcs7Unpad(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("empty")
	}
	pad := int(b[len(b)-1])
	if pad == 0 || pad > len(b) {
		return nil, fmt.Errorf("bad pad")
	}
	return b[:len(b)-pad], nil
}

func (c *Client) encrypt(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.lsk)
	if err != nil {
		return nil, err
	}
	plain = pkcs7Pad(plain, block.BlockSize())
	out := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, c.ivb).CryptBlocks(out, plain)
	return out, nil
}

func (c *Client) decrypt(ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.lsk)
	if err != nil {
		return nil, err
	}
	if len(ct)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("bad ciphertext len")
	}
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, c.ivb).CryptBlocks(out, ct)
	return pkcs7Unpad(out)
}

func (c *Client) tag(requestJSON []byte) string {
	t1 := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(c.hashedPassword()+c.cnonce))))
	t2 := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256(append(append([]byte(t1), requestJSON...), []byte(fmt.Sprintf("%d", c.seq))...))))
	return t2
}

// Execute sends a multipleRequest-style method via securePassthrough or plain stok.
func (c *Client) Execute(method string, params map[string]any) (map[string]any, error) {
	if c.klap == nil && c.stok == "" {
		if err := c.Login(); err != nil {
			return nil, err
		}
	}
	inner := map[string]any{
		"method": "multipleRequest",
		"params": map[string]any{
			"requests": []map[string]any{
				{"method": method, "params": params},
			},
		},
	}
	raw, _ := json.Marshal(inner)
	if c.klap != nil {
		return c.klap.send(inner)
	}
	url := fmt.Sprintf("https://%s/stok=%s/ds", c.controlHost(), c.stok)
	if c.secure && c.lsk != nil {
		ct, err := c.encrypt(raw)
		if err != nil {
			return nil, err
		}
		body := map[string]any{
			"method": "securePassthrough",
			"params": map[string]any{
				"request": base64.StdEncoding.EncodeToString(ct),
			},
		}
		hdr := map[string]string{
			"Seq":      fmt.Sprintf("%d", c.seq),
			"Tapo_tag": c.tag(raw),
		}
		c.seq++
		res, err := c.postJSON(url, body, hdr)
		if err != nil {
			return nil, err
		}
		if r, ok := res["result"].(map[string]any); ok {
			if enc, ok := r["response"].(string); ok {
				bin, err := base64.StdEncoding.DecodeString(enc)
				if err != nil {
					return res, err
				}
				pt, err := c.decrypt(bin)
				if err != nil {
					return res, err
				}
				var out map[string]any
				if err := json.Unmarshal(pt, &out); err != nil {
					return nil, err
				}
				return out, nil
			}
		}
		return res, nil
	}
	// legacy plain
	return c.postJSON(url, inner, nil)
}

func (c *Client) postJSON(url string, body map[string]any, extra map[string]string) (map[string]any, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", c.controlHost())
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("tapo: status %d body %s", resp.StatusCode, string(data))
	}
	return out, nil
}

// MoveMotor pans/tilts (pytapo moveMotor / motorMove).
func (c *Client) MoveMotor(x, y int) (map[string]any, error) {
	return c.Execute("motorMove", map[string]any{
		"motor": map[string]any{
			"move": map[string]any{
				"x_coord": fmt.Sprintf("%d", x),
				"y_coord": fmt.Sprintf("%d", y),
			},
		},
	})
}

// SetDayNight mode: on|off|auto (pytapo setDayNightModeConfig).
func (c *Client) SetDayNight(mode string) (map[string]any, error) {
	return c.Execute("setDayNightModeConfig", map[string]any{
		"image": map[string]any{
			"common": map[string]any{"inf_type": mode},
		},
	})
}

// SetPrivacy enables lens mask (pytapo setLensMaskConfig).
func (c *Client) SetPrivacy(on bool) (map[string]any, error) {
	en := "off"
	if on {
		en = "on"
	}
	return c.Execute("setLensMaskConfig", map[string]any{
		"lens_mask": map[string]any{
			"lens_mask_info": map[string]any{"enabled": en},
		},
	})
}
