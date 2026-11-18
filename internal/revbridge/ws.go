package revbridge

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type wsConn struct {
	net.Conn
	bufr     *bufio.Reader
	isClient bool
	rbuf     []byte // leftover payload from previous frame
}

func upgradeServerWS(c net.Conn) (net.Conn, error) {
	br := bufio.NewReader(c)
	req, err := http.ReadRequest(br)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		return nil, fmt.Errorf("not websocket upgrade")
	}
	key := req.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}
	accept := wsAccept(key)
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := c.Write([]byte(resp)); err != nil {
		return nil, err
	}
	return &wsConn{Conn: c, bufr: br, isClient: false}, nil
}

func dialClientWS(c net.Conn, host string) (net.Conn, error) {
	key := make([]byte, 16)
	_, _ = rand.Read(key)
	keyB64 := base64.StdEncoding.EncodeToString(key)
	req := fmt.Sprintf("GET /nd-rev HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", host, keyB64)
	if _, err := c.Write([]byte(req)); err != nil {
		return nil, err
	}
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 101 {
		return nil, fmt.Errorf("ws upgrade status %d", resp.StatusCode)
	}
	return &wsConn{Conn: c, bufr: br, isClient: true}, nil
}

func wsAccept(key string) string {
	h := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h[:])
}

func (w *wsConn) Read(p []byte) (int, error) {
	if len(w.rbuf) > 0 {
		n := copy(p, w.rbuf)
		w.rbuf = w.rbuf[n:]
		return n, nil
	}
	for {
		b0, err := w.bufr.ReadByte()
		if err != nil {
			return 0, err
		}
		b1, err := w.bufr.ReadByte()
		if err != nil {
			return 0, err
		}
		opcode := b0 & 0x0f
		masked := b1&0x80 != 0
		n := int(b1 & 0x7f)
		if n == 126 {
			var ext [2]byte
			if _, err := io.ReadFull(w.bufr, ext[:]); err != nil {
				return 0, err
			}
			n = int(binary.BigEndian.Uint16(ext[:]))
		} else if n == 127 {
			var ext [8]byte
			if _, err := io.ReadFull(w.bufr, ext[:]); err != nil {
				return 0, err
			}
			u := binary.BigEndian.Uint64(ext[:])
			// Cap before int conversion / allocation (DoS / G115).
			const maxWSFrame = 64 << 20 // 64 MiB
			if u > maxWSFrame {
				return 0, fmt.Errorf("websocket frame too large: %d", u)
			}
			n = int(u)
		}
		if n < 0 || n > 64<<20 {
			return 0, fmt.Errorf("websocket frame length invalid: %d", n)
		}
		var mask [4]byte
		if masked {
			if _, err := io.ReadFull(w.bufr, mask[:]); err != nil {
				return 0, err
			}
		}
		payload := make([]byte, n)
		if n > 0 {
			if _, err := io.ReadFull(w.bufr, payload); err != nil {
				return 0, err
			}
		}
		if masked {
			for i := range payload {
				payload[i] ^= mask[i%4]
			}
		}
		switch opcode {
		case 0x8:
			return 0, io.EOF
		case 0x9:
			_ = w.writeFrame(0xA, payload)
			continue
		case 0xA:
			continue
		case 0x1, 0x2:
			if len(payload) <= len(p) {
				return copy(p, payload), nil
			}
			ncopy := copy(p, payload)
			w.rbuf = append([]byte(nil), payload[ncopy:]...)
			return ncopy, nil
		default:
			continue
		}
	}
}

func (w *wsConn) Write(p []byte) (int, error) {
	// Split large writes into frames to keep peers happy.
	const maxChunk = 32 * 1024
	sent := 0
	for sent < len(p) {
		end := sent + maxChunk
		if end > len(p) {
			end = len(p)
		}
		if err := w.writeFrame(0x2, p[sent:end]); err != nil {
			return sent, err
		}
		sent = end
	}
	return sent, nil
}

func (w *wsConn) writeFrame(opcode byte, payload []byte) error {
	hdr := []byte{0x80 | opcode}
	n := len(payload)
	maskBit := byte(0)
	if w.isClient {
		maskBit = 0x80
	}
	switch {
	case n < 126:
		hdr = append(hdr, maskBit|byte(n))
	case n < 65536:
		hdr = append(hdr, maskBit|126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, maskBit|127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		hdr = append(hdr, ext[:]...)
	}
	var mask [4]byte
	out := payload
	if w.isClient {
		_, _ = rand.Read(mask[:])
		hdr = append(hdr, mask[:]...)
		buf := make([]byte, n)
		for i := 0; i < n; i++ {
			buf[i] = payload[i] ^ mask[i%4]
		}
		out = buf
	}
	if _, err := w.Conn.Write(hdr); err != nil {
		return err
	}
	if n > 0 {
		_, err := w.Conn.Write(out)
		return err
	}
	return nil
}
