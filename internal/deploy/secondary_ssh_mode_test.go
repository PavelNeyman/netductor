package deploy

import "testing"

func TestResolveSecondarySSHInitial(t *testing.T) {
	mode, port, _, err := ResolveSecondarySSH(SecondaryOpts{SecondaryPass: "rootpw"})
	if err != nil || mode != SSHModeInitial || port != 22 {
		t.Fatalf("got %s %d %v", mode, port, err)
	}
	mode, port, _, err = ResolveSecondarySSH(SecondaryOpts{SSHMode: SSHModeInitial, SecondaryPass: "x"})
	if err != nil || mode != SSHModeInitial || port != 22 {
		t.Fatalf("explicit initial: %s %d %v", mode, port, err)
	}
}

func TestResolveSecondarySSHResume(t *testing.T) {
	mode, port, key, err := ResolveSecondarySSH(SecondaryOpts{SecondarySSHKey: "/k"})
	if err != nil || mode != SSHModeResume || port != 52222 || key != "/k" {
		t.Fatalf("got %s %d %s %v", mode, port, key, err)
	}
	mode, port, key, err = ResolveSecondarySSH(SecondaryOpts{SSHMode: SSHModeResume, PrimaryKey: "/p"})
	if err != nil || port != 52222 || key != "/p" {
		t.Fatalf("resume falls back to primary key: %s %d %s %v", mode, port, key, err)
	}
	_, _, _, err = ResolveSecondarySSH(SecondaryOpts{SSHMode: SSHModeResume})
	if err == nil {
		t.Fatal("resume without key must fail")
	}
}

func TestResolveSecondarySSHAmbiguous(t *testing.T) {
	_, _, _, err := ResolveSecondarySSH(SecondaryOpts{SecondaryPass: "pw", SecondarySSHKey: "/k"})
	if err == nil {
		t.Fatal("both credentials without mode must fail")
	}
	_, _, _, err = ResolveSecondarySSH(SecondaryOpts{SSHMode: SSHModeInitial, SecondaryPass: "pw", SecondarySSHKey: "/k"})
	if err == nil {
		t.Fatal("initial plus a separate key must fail")
	}
}

func TestResolveSecondarySSHInitialNeedsPassword(t *testing.T) {
	_, _, _, err := ResolveSecondarySSH(SecondaryOpts{SSHMode: SSHModeInitial})
	if err == nil {
		t.Fatal("initial without password must fail")
	}
}

func TestSecondaryDialKeySeparate(t *testing.T) {
	path, pass := secondaryDialKey(SecondaryOpts{
		PrimaryKey: "/primary", PrimaryKeyPassphrase: "pp",
		SecondarySSHKey: "/secondary",
	})
	if path != "/secondary" || pass != "" {
		t.Fatalf("separate key must not use primary passphrase: %s %q", path, pass)
	}
	path, pass = secondaryDialKey(SecondaryOpts{
		PrimaryKey: "/primary", PrimaryKeyPassphrase: "pp",
	})
	if path != "/primary" || pass != "pp" {
		t.Fatalf("fallback: %s %q", path, pass)
	}
}

func TestResolveSecondarySSHInitialRejectsSeparateKey(t *testing.T) {
	_, _, _, err := ResolveSecondarySSH(SecondaryOpts{
		SSHMode: SSHModeInitial, SecondaryPass: "pw",
		PrimaryKey: "/p", SecondarySSHKey: "/other",
	})
	if err == nil {
		t.Fatal("initial with a different secondary key must fail")
	}
}

func TestRequirePublicIPv4(t *testing.T) {
	for _, bad := range []string{
		"10.1.2.3", "192.168.0.1", "127.0.0.1", "169.254.1.1", "host.example",
		"100.64.0.1", "192.0.2.1", "198.51.100.1", "203.0.113.1", "198.18.0.1", "240.0.0.1",
	} {
		if err := requirePublicIPv4(bad); err == nil {
			t.Fatalf("%s should be rejected", bad)
		}
	}
	if err := requirePublicIPv4("92.255.77.253"); err != nil {
		t.Fatal(err)
	}
}

func TestPrimarySSHPort(t *testing.T) {
	p, err := primarySSHPort(SecondaryOpts{})
	if err != nil || p != "52222" {
		t.Fatalf("default: %s %v", p, err)
	}
	p, err = primarySSHPort(SecondaryOpts{PrimarySSHPort: 2200})
	if err != nil || p != "2200" {
		t.Fatalf("custom: %s %v", p, err)
	}
	if _, err := primarySSHPort(SecondaryOpts{PrimarySSHPort: 70000}); err == nil {
		t.Fatal("invalid port accepted")
	}
}
