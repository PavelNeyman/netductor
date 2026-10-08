package tapo

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// Pure-Go TPAP / SPAKE2+ (V4) — port of freeKC/tapo-v4-protocol (MIT).
// No Python. Used when classic/KLAP login fails on newer camera FW.

var (
	p256P, _ = new(big.Int).SetString("FFFFFFFF00000001000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFF", 16)
	p256N, _ = new(big.Int).SetString("FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16)
	p256A    = new(big.Int).Sub(p256P, big.NewInt(3))
	p256B, _ = new(big.Int).SetString("5AC635D8AA3A93E7B3EBBD55769886BC651D06B0CC53B0F63BCE3C3E27D2604B", 16)
	p256Gx, _ = new(big.Int).SetString("6B17D1F2E12C4247F8BCE6E563A440F277037D812DEB33A0F4A13945D898C296", 16)
	p256Gy, _ = new(big.Int).SetString("4FE342E2FE1A7F9B8EE7EB4A7C0F9E162BCE33576B315ECECBB6406837BF51F5", 16)
)

type ecPoint struct{ x, y *big.Int } // nil = infinity

func ptAdd(p1, p2 *ecPoint) *ecPoint {
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	if p1.x.Cmp(p2.x) == 0 && new(big.Int).Add(p1.y, p2.y).Mod(new(big.Int).Add(p1.y, p2.y), p256P).Sign() == 0 {
		return nil
	}
	var m *big.Int
	if p1.x.Cmp(p2.x) == 0 && p1.y.Cmp(p2.y) == 0 {
		// slope = (3x^2+a)/(2y)
		num := new(big.Int).Mul(p1.x, p1.x)
		num.Mul(num, big.NewInt(3)).Add(num, p256A).Mod(num, p256P)
		den := new(big.Int).Mul(p1.y, big.NewInt(2))
		den.ModInverse(den, p256P)
		m = num.Mul(num, den).Mod(num, p256P)
	} else {
		num := new(big.Int).Sub(p2.y, p1.y)
		den := new(big.Int).Sub(p2.x, p1.x)
		den.ModInverse(den, p256P)
		m = num.Mul(num, den).Mod(num, p256P)
	}
	x3 := new(big.Int).Mul(m, m)
	x3.Sub(x3, p1.x).Sub(x3, p2.x).Mod(x3, p256P)
	y3 := new(big.Int).Sub(p1.x, x3)
	y3.Mul(y3, m).Sub(y3, p1.y).Mod(y3, p256P)
	return &ecPoint{x3, y3}
}

func ptNeg(p *ecPoint) *ecPoint {
	if p == nil {
		return nil
	}
	ny := new(big.Int).Neg(p.y)
	ny.Mod(ny, p256P)
	return &ecPoint{new(big.Int).Set(p.x), ny}
}

func ptMul(k *big.Int, p *ecPoint) *ecPoint {
	if k.Sign() == 0 || p == nil {
		return nil
	}
	k = new(big.Int).Mod(k, p256N)
	var r *ecPoint
	addend := p
	kk := new(big.Int).Set(k)
	for kk.Sign() > 0 {
		if kk.Bit(0) == 1 {
			r = ptAdd(r, addend)
		}
		addend = ptAdd(addend, addend)
		kk.Rsh(kk, 1)
	}
	return r
}

func decodePoint(b []byte) (*ecPoint, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("empty point")
	}
	if b[0] == 0x04 && len(b) >= 65 {
		return &ecPoint{new(big.Int).SetBytes(b[1:33]), new(big.Int).SetBytes(b[33:65])}, nil
	}
	if (b[0] == 0x02 || b[0] == 0x03) && len(b) >= 33 {
		x := new(big.Int).SetBytes(b[1:33])
		// y^2 = x^3 + ax + b
		y2 := new(big.Int).Mul(x, x)
		y2.Mul(y2, x)
		ax := new(big.Int).Mul(p256A, x)
		y2.Add(y2, ax).Add(y2, p256B).Mod(y2, p256P)
		y := new(big.Int).ModSqrt(y2, p256P)
		if y == nil {
			return nil, fmt.Errorf("invalid compressed point")
		}
		if byte(y.Bit(0)) != (b[0] & 1) {
			y.Sub(p256P, y)
		}
		return &ecPoint{x, y}, nil
	}
	return nil, fmt.Errorf("bad point encoding")
}

func encodeUncompressed(p *ecPoint) []byte {
	out := make([]byte, 65)
	out[0] = 0x04
	xb := p.x.Bytes()
	yb := p.y.Bytes()
	copy(out[33-len(xb):33], xb)
	copy(out[65-len(yb):65], yb)
	return out
}

func hkdfSHA256(ikm, salt, info []byte, length int) []byte {
	if salt == nil {
		salt = make([]byte, 32)
	}
	mac := hmac.New(sha256.New, salt)
	mac.Write(ikm)
	prk := mac.Sum(nil)
	var out, t []byte
	i := byte(1)
	for len(out) < length {
		m := hmac.New(sha256.New, prk)
		m.Write(t)
		m.Write(info)
		m.Write([]byte{i})
		t = m.Sum(nil)
		out = append(out, t...)
		i++
	}
	return out[:length]
}

func lenPrefixed(chunks ...[]byte) []byte {
	var out []byte
	for _, c := range chunks {
		var le [8]byte
		binary.LittleEndian.PutUint64(le[:], uint64(len(c)))
		out = append(out, le[:]...)
		out = append(out, c...)
	}
	return out
}

// sha256Crypt is Unix $5$ (Drepper) — C200 extra_crypt password_shadow pid=5.
func sha256Crypt(key, prefix string) string {
	const cryptB64 = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	cryptOrder := [][3]int{{0, 10, 20}, {21, 1, 11}, {12, 22, 2}, {3, 13, 23}, {24, 4, 14},
		{15, 25, 5}, {6, 16, 26}, {27, 7, 17}, {18, 28, 8}, {9, 19, 29}}
	rest := prefix
	if strings.HasPrefix(prefix, "$5$") {
		rest = prefix[3:]
	}
	rounds := 5000
	explicit := false
	if strings.HasPrefix(rest, "rounds=") {
		if i := strings.IndexByte(rest, '$'); i > 0 {
			var r int
			fmt.Sscanf(rest[7:i], "%d", &r)
			if r < 1000 {
				r = 1000
			}
			if r > 999999999 {
				r = 999999999
			}
			rounds, explicit, rest = r, true, rest[i+1:]
		}
	}
	salt := rest
	if i := strings.IndexByte(rest, '$'); i >= 0 {
		salt = rest[:i]
	}
	if len(salt) > 16 {
		salt = salt[:16]
	}
	k, s := []byte(key), []byte(salt)
	rep := func(d []byte, n int) []byte {
		out := make([]byte, 0, n)
		for len(out) < n {
			out = append(out, d...)
		}
		return out[:n]
	}
	b := sha256.Sum256(append(append(k, s...), k...))
	a := sha256.New()
	a.Write(k)
	a.Write(s)
	a.Write(rep(b[:], len(k)))
	for n := len(k); n > 0; n >>= 1 {
		if n&1 != 0 {
			a.Write(b[:])
		} else {
			a.Write(k)
		}
	}
	adig := a.Sum(nil)
	p := rep(func() []byte { h := sha256.Sum256(bytes.Repeat(k, len(k))); return h[:] }(), len(k))
	sb := rep(func() []byte {
		h := sha256.Sum256(bytes.Repeat(s, 16+int(adig[0])))
		return h[:]
	}(), len(s))
	c := adig
	for i := 0; i < rounds; i++ {
		h := sha256.New()
		if i&1 != 0 {
			h.Write(p)
		} else {
			h.Write(c)
		}
		if i%3 != 0 {
			h.Write(sb)
		}
		if i%7 != 0 {
			h.Write(p)
		}
		if i&1 != 0 {
			h.Write(c)
		} else {
			h.Write(p)
		}
		c = h.Sum(nil)
	}
	crypt64 := func(value int, n int) string {
		out := make([]byte, n)
		for i := 0; i < n; i++ {
			out[i] = cryptB64[value&0x3f]
			value >>= 6
		}
		return string(out)
	}
	var enc strings.Builder
	for _, o := range cryptOrder {
		enc.WriteString(crypt64((int(c[o[0]])<<16)|(int(c[o[1]])<<8)|int(c[o[2]]), 4))
	}
	enc.WriteString(crypt64((int(c[31])<<8)|int(c[30]), 3))
	rs := ""
	if explicit {
		rs = fmt.Sprintf("rounds=%d$", rounds)
	}
	return fmt.Sprintf("$5$%s%s$%s", rs, salt, enc.String())
}

func applyExtraCrypt(passcode string, extra map[string]any) (string, error) {
	if extra == nil {
		return passcode, nil
	}
	kind, _ := extra["type"].(string)
	params, _ := extra["params"].(map[string]any)
	if strings.ToLower(kind) != "password_shadow" {
		return "", fmt.Errorf("extra_crypt type %q", kind)
	}
	pid := 0
	switch v := params["passwd_id"].(type) {
	case float64:
		pid = int(v)
	case string:
		fmt.Sscanf(v, "%d", &pid)
	}
	switch pid {
	case 5:
		pref, _ := params["passwd_prefix"].(string)
		return sha256Crypt(passcode, pref), nil
	case 2:
		sum := sha1.Sum([]byte(passcode))
		return hex.EncodeToString(sum[:]), nil
	default:
		return "", fmt.Errorf("extra_crypt passwd_id %d", pid)
	}
}

type tpapSession struct {
	stok      string
	seq       int
	key       []byte
	nonce0    []byte
	expiresAt time.Time
}

func (c *Client) postLoginJSON(params map[string]any) (map[string]any, error) {
	body := map[string]any{"method": "login", "params": params}
	raw, _ := json.Marshal(body)
	url := "https://" + c.controlHost() + "/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("requestByApp", "true")
	req.Header.Set("User-Agent", "Tapo CameraClient Android")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("tpap login json: %w body=%s", err, string(b))
	}
	return out, nil
}

func (c *Client) loginTPAPOnce(credentialHash string) error {
	pwd := c.cloudPass()
	if pwd == "" {
		return fmt.Errorf("tpap: empty cloud password")
	}
	userRandom := make([]byte, 32)
	_, _ = rand.Read(userRandom)
	userRandomB64 := base64.StdEncoding.EncodeToString(userRandom)

	reg, err := c.postLoginJSON(map[string]any{
		"sub_method":    "pake_register",
		"username":      "admin",
		"user_random":   userRandomB64,
		"cipher_suites": []int{1},
		"passcode_type": "userpw",
	})
	if err != nil {
		return err
	}
	res, _ := reg["result"].(map[string]any)
	if res == nil {
		return fmt.Errorf("tpap pake_register: %v", reg)
	}
	devSaltB64, _ := res["dev_salt"].(string)
	devRandomB64, _ := res["dev_random"].(string)
	devShareB64, _ := res["dev_share"].(string)
	iters := 1000
	switch v := res["iterations"].(type) {
	case float64:
		iters = int(v)
	}
	devSalt, err := base64.StdEncoding.DecodeString(devSaltB64)
	if err != nil {
		return err
	}
	devShare, err := base64.StdEncoding.DecodeString(devShareB64)
	if err != nil {
		return err
	}
	Y, err := decodePoint(devShare)
	if err != nil {
		return err
	}
	devRandom, err := base64.StdEncoding.DecodeString(devRandomB64)
	if err != nil {
		return err
	}

	var credential string
	if credentialHash == "sha256" {
		credential = strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(pwd))))
	} else {
		credential = fmt.Sprintf("%x", md5.Sum([]byte(pwd)))
	}
	if extra, ok := res["extra_crypt"].(map[string]any); ok {
		credential, err = applyExtraCrypt(credential, extra)
		if err != nil {
			return err
		}
	}

	dk := pbkdf2.Key([]byte(credential), devSalt, iters, 80, sha256.New)
	w0 := new(big.Int).SetBytes(dk[0:40])
	w0.Mod(w0, p256N)
	w1 := new(big.Int).SetBytes(dk[40:80])
	w1.Mod(w1, p256N)

	M, _ := decodePoint(mustHex("02886e2f97ace46e55ba9dd7242579f2993b64e16ef3dcab95afd497333d8fa12f"))
	Npt, _ := decodePoint(mustHex("03d8bbd6c639c62937b04d997f38c3770719c629d7014d49a24b4f98baa1292b49"))
	G := &ecPoint{p256Gx, p256Gy}

	xb := make([]byte, 32)
	_, _ = rand.Read(xb)
	x := new(big.Int).SetBytes(xb)
	x.Mod(x, new(big.Int).Sub(p256N, big.NewInt(1)))
	x.Add(x, big.NewInt(1))
	X := ptAdd(ptMul(x, G), ptMul(w0, M))
	H := ptAdd(Y, ptNeg(ptMul(w0, Npt)))
	Z := ptMul(x, H)
	V := ptMul(w1, H)

	Xb, Yb := encodeUncompressed(X), encodeUncompressed(Y)
	Mb, Nb := encodeUncompressed(M), encodeUncompressed(Npt)
	Zb, Vb := encodeUncompressed(Z), encodeUncompressed(V)
	w0b := make([]byte, 32)
	w0.FillBytes(w0b)

	ctx := sha256.Sum256(append(append([]byte("PAKE V1"), userRandom...), devRandom...))
	TT := lenPrefixed(ctx[:], nil, nil, Mb, Nb, Xb, Yb, Zb, Vb, w0b)
	ke := sha256.Sum256(TT)
	conf := hkdfSHA256(ke[:], nil, []byte("ConfirmationKeys"), 64)
	KcA, KcB := conf[:32], conf[32:]
	shared := hkdfSHA256(ke[:], nil, []byte("SharedKey"), 32)

	macA := hmac.New(sha256.New, KcA)
	macA.Write(Yb)
	cA := macA.Sum(nil)

	share, err := c.postLoginJSON(map[string]any{
		"sub_method":   "pake_share",
		"user_share":   base64.StdEncoding.EncodeToString(Xb),
		"user_confirm": base64.StdEncoding.EncodeToString(cA),
	})
	if err != nil {
		return err
	}
	sres, _ := share["result"].(map[string]any)
	if sres == nil {
		return fmt.Errorf("tpap pake_share: %v", share)
	}
	devConfB64, _ := sres["dev_confirm"].(string)
	devConf, _ := base64.StdEncoding.DecodeString(devConfB64)
	macB := hmac.New(sha256.New, KcB)
	macB.Write(Xb)
	expectB := macB.Sum(nil)
	if !hmac.Equal(devConf, expectB) {
		return fmt.Errorf("tpap: dev_confirm mismatch (wrong cloud password?)")
	}
	stok, _ := sres["stok"].(string)
	seq := 0
	switch v := sres["start_seq"].(type) {
	case float64:
		seq = int(v)
	}
	bk := hkdfSHA256(shared, []byte("tp-kdf-salt-aes128-key"), []byte("tp-kdf-info-aes128-key"), 32)
	bn := hkdfSHA256(shared, []byte("tp-kdf-salt-aes128-iv"), []byte("tp-kdf-info-aes128-iv"), 32)
	c.stok = stok
	c.tpap = true
	c.tpapSess = &tpapSession{stok: stok, seq: seq, key: bk[:16], nonce0: bn[:12], expiresAt: time.Now().Add(time.Hour)}
	return nil
}

func mustHex(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

func (c *Client) cloudPass() string {
	if c.CloudPassword != "" {
		return c.CloudPassword
	}
	return c.Password
}

// loginTPAP tries md5 then sha256 credential hash (matches freeKC / C200 variants).
func (c *Client) loginTPAP() error {
	var last error
	for _, h := range []string{"md5", "sha256"} {
		if err := c.loginTPAPOnce(h); err == nil {
			return nil
		} else {
			last = err
		}
	}
	return last
}

func (c *Client) tpapNonce(seq int) []byte {
	n := make([]byte, 12)
	copy(n, c.tpapSess.nonce0[:8])
	binary.BigEndian.PutUint32(n[8:], uint32(seq))
	return n
}

func (c *Client) executeTPAP(method string, params map[string]any) (map[string]any, error) {
	if c.tpapSess == nil || c.tpapSess.stok == "" {
		if err := c.loginTPAP(); err != nil {
			return nil, err
		}
	}
	if params == nil {
		params = map[string]any{}
	}
	innerObj := map[string]any{
		"method": "multipleRequest",
		"params": map[string]any{
			"requests": []map[string]any{{"method": method, "params": params}},
		},
	}
	inner, _ := json.Marshal(innerObj)
	// compact
	var buf bytes.Buffer
	_ = json.Compact(&buf, inner)
	inner = buf.Bytes()

	seq := c.tpapSess.seq
	c.tpapSess.seq++
	ccm, err := newAESCCM(c.tpapSess.key)
	if err != nil {
		return nil, err
	}
	ct, err := ccm.seal(c.tpapNonce(seq), inner)
	if err != nil {
		return nil, err
	}
	body := make([]byte, 4+len(ct))
	binary.BigEndian.PutUint32(body[:4], uint32(seq))
	copy(body[4:], ct)

	url := fmt.Sprintf("https://%s/stok=%s/ds", c.controlHost(), c.tpapSess.stok)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("requestByApp", "true")
	req.Header.Set("User-Agent", "Tapo CameraClient Android")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 && raw[0] == '{' {
		return nil, fmt.Errorf("tpap /ds refused: %s", string(raw))
	}
	if len(raw) < 4+16 {
		return nil, fmt.Errorf("tpap /ds short reply %d", len(raw))
	}
	respSeq := int(binary.BigEndian.Uint32(raw[:4]))
	pt, err := ccm.open(c.tpapNonce(respSeq), raw[4:])
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, err
	}
	if code, ok := out["error_code"].(float64); ok && code != 0 {
		return out, fmt.Errorf("tpap error_code=%v", code)
	}
	// unwrap first response if multipleRequest
	if res, ok := out["result"].(map[string]any); ok {
		if responses, ok := res["responses"].([]any); ok && len(responses) > 0 {
			if r0, ok := responses[0].(map[string]any); ok {
				return r0, nil
			}
		}
	}
	return out, nil
}
