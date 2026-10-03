package install

import (
	"strings"
	"testing"
)

func TestListenPortsFromSS(t *testing.T) {
	ss := `
tcp   LISTEN 0 128 0.0.0.0:22 0.0.0.0:*
tcp   LISTEN 0 128 0.0.0.0:10050 0.0.0.0:*
tcp   LISTEN 0 128 127.0.0.1:9100 0.0.0.0:*
tcp   LISTEN 0 128 0.0.0.0:100500 0.0.0.0:*
`
	got := listenPortsFromSS(ss, []string{"10050", "9100", "10051"})
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	j := strings.Join(got, ",")
	if !strings.Contains(j, "10050") || !strings.Contains(j, "9100") {
		t.Fatalf("got %v", got)
	}
}

func TestPackageNameUnwanted(t *testing.T) {
	if !packageNameUnwanted("zabbix-agent-timeweb") {
		t.Fatal("timeweb")
	}
	if !packageNameUnwanted("puppet-agent") {
		t.Fatal("puppet-agent")
	}
	if packageNameUnwanted("libsomething-puppet-utils") {
		t.Fatal("false positive puppet utils")
	}
	if !packageNameUnwanted("puppet") {
		t.Fatal("exact puppet")
	}
}
