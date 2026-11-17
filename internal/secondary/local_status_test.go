package secondary

import "testing"

func TestCollectLocalStatusShape(t *testing.T) {
	st := CollectLocalStatus()
	if st.Role != "egress" || st.Version == "" || st.TS == "" {
		t.Fatalf("%+v", st)
	}
}
