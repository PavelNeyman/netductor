package tlsle

import "testing"

func TestFindLiveCertEmpty(t *testing.T) {
	_ = findLiveCert(nil)
	_ = findLiveCert([]string{"unlikely-domain-netductor-test.invalid"})
}
