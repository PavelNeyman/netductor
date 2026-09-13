package vpn

import "testing"

func TestSingBoxClientJSONMinimal(t *testing.T) {
	e := ClientEndpoints{Name: "t", UUID: "11111111-1111-1111-1111-111111111111",
		CoreHost: "1.2.3.4", CorePort: 443, CoreSNI: "ya.ru", CorePBK: "pbk", CoreSID: "ab"}
	b, err := SingBoxClientJSON(e)
	if err != nil || len(b) < 20 {
		t.Fatal(err, string(b))
	}
}
