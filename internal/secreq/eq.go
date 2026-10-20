// Package secreq compares secrets in constant time.
package secreq

import (
	"crypto/sha256"
	"crypto/subtle"
)

// Eq reports whether a and b are equal without leaking the mismatch index.
// Empty string never matches. Inputs are hashed so the compare is fixed-length.
func Eq(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	x := sha256.Sum256([]byte(a))
	y := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}
