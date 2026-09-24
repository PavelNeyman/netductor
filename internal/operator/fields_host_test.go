package operator

import "testing"

func TestValidHost(t *testing.T) {
	if ValidHost("1.2.3.4") != true {
		t.Fatal("ip")
	}
	if ValidHost("evil;rm -rf") {
		t.Fatal("meta")
	}
	if ValidHost("") {
		t.Fatal("empty")
	}
}
