package tapo

import (
	"bytes"
	"testing"
)

func TestAESCCMRoundTrip(t *testing.T) {
	key := make([]byte, 16)
	nonce := make([]byte, 12)
	for i := range key {
		key[i] = byte(i)
	}
	for i := range nonce {
		nonce[i] = byte(i + 1)
	}
	c, err := newAESCCM(key)
	if err != nil {
		t.Fatal(err)
	}
	pt := []byte(`{"method":"multipleRequest","params":{"requests":[]}}`)
	ct, err := c.seal(nonce, pt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.open(nonce, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, pt) {
		t.Fatalf("mismatch %s vs %s", got, pt)
	}
}
