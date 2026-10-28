package main

import "testing"

func TestValidMAC(t *testing.T) {
	ok, err := validMAC("AA-BB-CC-DD-EE-FF")
	if err != nil || ok != "aa:bb:cc:dd:ee:ff" {
		t.Fatal(ok, err)
	}
	if _, err := validMAC("aa:bb\ncc:dd:ee:ff"); err == nil {
		t.Fatal("CRLF must fail")
	}
	if _, err := validMAC("not-a-mac"); err == nil {
		t.Fatal("garbage")
	}
}

func TestValidIPv4(t *testing.T) {
	ip, err := validIPv4("192.168.1.50")
	if err != nil || ip != "192.168.1.50" {
		t.Fatal(ip, err)
	}
	if _, err := validIPv4("999.1.1.1"); err == nil {
		t.Fatal()
	}
	if _, err := validIPv4("1.2.3.4\nset foo"); err == nil {
		t.Fatal("inject")
	}
}

func TestValidDHCPName(t *testing.T) {
	if _, err := validDHCPName("tapo1"); err != nil {
		t.Fatal(err)
	}
	if _, err := validDHCPName("bad name"); err == nil {
		t.Fatal("space")
	}
	if _, err := validDHCPName("x\nset dhcp"); err == nil {
		t.Fatal("newline")
	}
}

func TestParseGatewayCIDR(t *testing.T) {
	gw, n, err := parseGatewayCIDR("192.168.50.1/24")
	if err != nil || gw != "192.168.50.1" || n == nil {
		t.Fatal(gw, n, err)
	}
	if _, _, err := parseGatewayCIDR("192.168.50.1/24\nset x"); err == nil {
		t.Fatal("newline")
	}
	if _, _, err := parseGatewayCIDR("not-cidr"); err == nil {
		t.Fatal()
	}
}

func TestSafeUCIValue(t *testing.T) {
	if _, err := safeUCIValue("Guest"); err != nil {
		t.Fatal(err)
	}
	if _, err := safeUCIValue("a'b"); err == nil {
		t.Fatal("quote")
	}
	if _, err := safeUCIValue("a\nb"); err == nil {
		t.Fatal("nl")
	}
}

func TestDhcpStaticHostRejectsInjection(t *testing.T) {
	// No real uci binary needed — validation fails first.
	if got := dhcpStaticHost("mac=aa:bb:cc:dd:ee:ff|ip=1.2.3.4\nset evil|name=x"); got != "error:invalid arg" {
		t.Fatal(got)
	}
	if got := dhcpStaticHost("mac=bad|ip=1.2.3.4"); got != "error:invalid mac" {
		t.Fatal(got)
	}
	if got := dhcpStaticHost("mac=aa:bb:cc:dd:ee:ff|ip=not-ip"); got != "error:invalid ip" {
		t.Fatal(got)
	}
}

func TestValidUCIPath(t *testing.T) {
	if _, err := validUCIPath("network.lan.ipaddr"); err != nil {
		t.Fatal(err)
	}
	if _, err := validUCIPath("dhcp.@host[0].mac"); err != nil {
		t.Fatal(err)
	}
	if _, err := validUCIPath("a;rm -rf"); err == nil {
		t.Fatal("shell metachar")
	}
	if _, err := validUCIPath("x\ny"); err == nil {
		t.Fatal("newline")
	}
}

func TestValidBackupName(t *testing.T) {
	if _, err := validBackupName("cfg-2026.tar.gz"); err != nil {
		t.Fatal(err)
	}
	if _, err := validBackupName("../etc/passwd"); err == nil {
		t.Fatal("traversal")
	}
	if _, err := validBackupName("a/b"); err == nil {
		t.Fatal("slash")
	}
}

func TestSafeLocalArchivePath(t *testing.T) {
	if _, err := safeLocalArchivePath("/mnt/netductor/x.tar.gz", "/mnt/netductor"); err != nil {
		t.Fatal(err)
	}
	if _, err := safeLocalArchivePath("/etc/passwd", "/mnt/netductor"); err == nil {
		t.Fatal("outside")
	}
	if _, err := safeLocalArchivePath("/mnt/netductor/../etc/passwd", "/mnt/netductor"); err == nil {
		t.Fatal("dotdot")
	}
}
