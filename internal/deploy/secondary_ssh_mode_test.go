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
	for _, bad := range []string{"10.1.2.3", "192.168.0.1", "127.0.0.1", "169.254.1.1", "host.example"} {
		if err := requirePublicIPv4(bad); err == nil {
			t.Fatalf("%s should be rejected", bad)
		}
	}
	if err := requirePublicIPv4("92.255.77.253"); err != nil {
		t.Fatal(err)
	}
}
