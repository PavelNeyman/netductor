package nvr

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Tapo C200 notes (locked for implementers):
// - RTSP: Camera Account (not TP-Link cloud login); stream1 HD, stream2 SD; port 554.
// - ONVIF: Profile S on many firmwares, service port **2020** (TP-Link FAQ). Some HW/FW builds
//   expose RTSP only — PTZ then fails; do not depend on HA Tapo-Control plugins.
// - PTZ via ONVIF ContinuousMove is best-effort; presets/absolute often incomplete on C200.
// - Night/IR is on-camera automatic; NVR does not force night mode.
// - No two-way audio over Profile S.

// ONVIFPTZ moves the camera briefly (ContinuousMove then Stop).
// dir: left|right|up|down|stop  speed ~0.3  durationMs default 800.
func ONVIFPTZ(host, user, pass, dir string, durationMs int) string {
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	if host == "" || user == "" {
		return "error:host/user required"
	}
	if durationMs <= 0 {
		durationMs = 800
	}
	if durationMs > 5000 {
		durationMs = 5000
	}
	px, py := 0.0, 0.0
	switch strings.ToLower(dir) {
	case "left":
		px = -0.4
	case "right":
		px = 0.4
	case "up":
		py = 0.3
	case "down":
		py = -0.3
	case "stop", "home":
		// stop only / home not reliable on C200 ONVIF
	default:
		return "error:dir left|right|up|down|stop"
	}
	endpoint := "http://" + host + ":2020/onvif/PTZ"
	// Profile token is often "profile_1" or first profile — C200 frequently accepts Profile_1 / profile1
	profile := "Profile_1"
	if dir == "stop" || dir == "home" {
		body := soapEnvelope(user, pass, ptzStopXML(profile))
		code, msg := postSOAP(endpoint, body)
		return fmt.Sprintf("onvif:stop status=%d %s", code, truncateRunes(msg, 80))
	}
	body := soapEnvelope(user, pass, continuousMoveXML(profile, px, py))
	code, msg := postSOAP(endpoint, body)
	if code >= 400 {
		// retry alternate profile token names
		for _, p := range []string{"profile_1", "Profile1", "MainStream"} {
			body = soapEnvelope(user, pass, continuousMoveXML(p, px, py))
			code, msg = postSOAP(endpoint, body)
			if code < 400 {
				profile = p
				break
			}
		}
	}
	time.Sleep(time.Duration(durationMs) * time.Millisecond)
	_, _ = postSOAP(endpoint, soapEnvelope(user, pass, ptzStopXML(profile)))
	return fmt.Sprintf("onvif:move dir=%s profile=%s http=%d %s", dir, profile, code, truncateRunes(msg, 60))
}

func continuousMoveXML(profile string, x, y float64) string {
	return fmt.Sprintf(`<ContinuousMove xmlns="http://www.onvif.org/ver20/ptz/wsdl">
  <ProfileToken>%s</ProfileToken>
  <Velocity>
    <PanTilt xmlns="http://www.onvif.org/ver10/schema" x="%g" y="%g"/>
  </Velocity>
</ContinuousMove>`, profile, x, y)
}

func ptzStopXML(profile string) string {
	return fmt.Sprintf(`<Stop xmlns="http://www.onvif.org/ver20/ptz/wsdl">
  <ProfileToken>%s</ProfileToken>
  <PanTilt>true</PanTilt>
  <Zoom>true</Zoom>
</Stop>`, profile)
}

func soapEnvelope(user, pass, inner string) string {
	nonce := make([]byte, 16)
	_, _ = rand.Read(nonce)
	created := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	// PasswordDigest = Base64(SHA1(nonce + created + password))
	h := sha1.New()
	h.Write(nonce)
	h.Write([]byte(created))
	h.Write([]byte(pass))
	digest := base64.StdEncoding.EncodeToString(h.Sum(nil))
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
  xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
  xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
  <s:Header>
    <wsse:Security s:mustUnderstand="1">
      <wsse:UsernameToken>
        <wsse:Username>%s</wsse:Username>
        <wsse:Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</wsse:Password>
        <wsse:Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</wsse:Nonce>
        <wsu:Created>%s</wsu:Created>
      </wsse:UsernameToken>
    </wsse:Security>
  </s:Header>
  <s:Body>%s</s:Body>
</s:Envelope>`, xmlEsc(user), digest, nonceB64, created, inner)
}

func postSOAP(url, body string) (int, string) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(body)))
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return resp.StatusCode, string(b)
}

func xmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func truncateRunes(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n]
}
