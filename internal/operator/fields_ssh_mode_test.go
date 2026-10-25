package operator

import "testing"

func TestSecondaryFromFieldsMode(t *testing.T) {
	get := func(m map[string]string) FieldGetter {
		return func(k string) string { return m[k] }
	}
	s := SecondaryFromFields(get(map[string]string{"ssh_mode": "", "password": "pw"}))
	if s.SSHMode != "initial" {
		t.Fatalf("password only: %s", s.SSHMode)
	}
	s = SecondaryFromFields(get(map[string]string{"ssh_mode": "", "secondary_ssh_key": "/k"}))
	if s.SSHMode != "resume" {
		t.Fatalf("key only: %s", s.SSHMode)
	}
	s = SecondaryFromFields(get(map[string]string{"ssh_mode": "", "password": "pw", "secondary_ssh_key": "/k"}))
	if s.SSHMode != "" {
		t.Fatalf("both without mode must stay empty, got %q", s.SSHMode)
	}
	s = SecondaryFromFields(get(map[string]string{"ssh_mode": "initial", "password": "pw", "secondary_ssh_key": "/k"}))
	if s.SSHMode != "initial" {
		t.Fatalf("explicit initial: %s", s.SSHMode)
	}
}
