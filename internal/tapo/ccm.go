package tapo

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"errors"
)

// aesCCM implements AES-128-CCM with fixed tag size 16 (as Tapo /ds uses).
type aesCCM struct {
	b cipher.Block
}

func newAESCCM(key []byte) (*aesCCM, error) {
	if len(key) != 16 {
		return nil, errors.New("ccm: key must be 16 bytes")
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &aesCCM{b: b}, nil
}

func (c *aesCCM) seal(nonce, plaintext []byte) ([]byte, error) {
	if len(nonce) != 12 {
		return nil, errors.New("ccm: nonce must be 12 bytes")
	}
	tagSize := 16
	L := 15 - len(nonce) // 3
	// CBC-MAC over B0 || len-prefixed plaintext (no AAD)
	b0 := make([]byte, 16)
	b0[0] = byte(((tagSize-2)/2)<<3 | (L - 1))
	copy(b0[1:], nonce)
	ln := len(plaintext)
	b0[13] = byte(ln >> 16)
	b0[14] = byte(ln >> 8)
	b0[15] = byte(ln)

	x := make([]byte, 16)
	c.b.Encrypt(x, b0)
	// payload blocks
	for i := 0; i < len(plaintext); {
		var block [16]byte
		n := copy(block[:], plaintext[i:])
		i += n
		for j := 0; j < 16; j++ {
			x[j] ^= block[j]
		}
		c.b.Encrypt(x, x)
	}
	tag := make([]byte, tagSize)
	copy(tag, x[:tagSize])

	// CTR encrypt
	ctr := make([]byte, 16)
	ctr[0] = byte(L - 1)
	copy(ctr[1:], nonce)
	s0 := make([]byte, 16)
	c.b.Encrypt(s0, ctr)
	for i := 0; i < tagSize; i++ {
		tag[i] ^= s0[i]
	}
	out := make([]byte, len(plaintext)+tagSize)
	copy(out[len(plaintext):], tag)
	for off := 0; off < len(plaintext); {
		incCTR(ctr)
		s := make([]byte, 16)
		c.b.Encrypt(s, ctr)
		n := 16
		if left := len(plaintext) - off; left < n {
			n = left
		}
		for i := 0; i < n; i++ {
			out[off+i] = plaintext[off+i] ^ s[i]
		}
		off += n
	}
	return out, nil
}

func (c *aesCCM) open(nonce, ciphertextAndTag []byte) ([]byte, error) {
	if len(nonce) != 12 {
		return nil, errors.New("ccm: nonce must be 12 bytes")
	}
	tagSize := 16
	if len(ciphertextAndTag) < tagSize {
		return nil, errors.New("ccm: short ciphertext")
	}
	ct := ciphertextAndTag[:len(ciphertextAndTag)-tagSize]
	tag := ciphertextAndTag[len(ciphertextAndTag)-tagSize:]
	L := 15 - len(nonce)
	ctr := make([]byte, 16)
	ctr[0] = byte(L - 1)
	copy(ctr[1:], nonce)
	s0 := make([]byte, 16)
	c.b.Encrypt(s0, ctr)
	pt := make([]byte, len(ct))
	for off := 0; off < len(ct); {
		incCTR(ctr)
		s := make([]byte, 16)
		c.b.Encrypt(s, ctr)
		n := 16
		if left := len(ct) - off; left < n {
			n = left
		}
		for i := 0; i < n; i++ {
			pt[off+i] = ct[off+i] ^ s[i]
		}
		off += n
	}
	// recompute tag
	b0 := make([]byte, 16)
	b0[0] = byte(((tagSize-2)/2)<<3 | (L - 1))
	copy(b0[1:], nonce)
	ln := len(pt)
	b0[13] = byte(ln >> 16)
	b0[14] = byte(ln >> 8)
	b0[15] = byte(ln)
	x := make([]byte, 16)
	c.b.Encrypt(x, b0)
	for i := 0; i < len(pt); {
		var block [16]byte
		n := copy(block[:], pt[i:])
		i += n
		for j := 0; j < 16; j++ {
			x[j] ^= block[j]
		}
		c.b.Encrypt(x, x)
	}
	expect := make([]byte, tagSize)
	copy(expect, x[:tagSize])
	for i := 0; i < tagSize; i++ {
		expect[i] ^= s0[i]
	}
	if subtle.ConstantTimeCompare(expect, tag) != 1 {
		return nil, errors.New("ccm: authentication failed")
	}
	return pt, nil
}

func incCTR(ctr []byte) {
	// increment last L bytes (L=3 → last 3); simpler: treat full counter from byte 13
	for i := 15; i >= 13; i-- {
		ctr[i]++
		if ctr[i] != 0 {
			return
		}
	}
}
