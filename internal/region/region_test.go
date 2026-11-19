package region

import "testing"

func TestValid(t *testing.T) {
	for _, s := range []string{"", "ru", "eu", "ru-msk", "fi-hel", "us-nyc"} {
		if !Valid(s) {
			t.Fatalf("want valid %q", s)
		}
	}
	for _, s := range []string{"R", "toolongname", "ru_msk", "r1-msk", "-msk"} {
		if Valid(s) {
			t.Fatalf("want invalid %q", s)
		}
	}
}

func TestFlagEmoji(t *testing.T) {
	if FlagEmoji("ru-msk") != "🇷🇺" {
		t.Fatalf("ru flag: %q", FlagEmoji("ru-msk"))
	}
	if FlagEmoji("fi-hel") != "🇫🇮" {
		t.Fatalf("fi flag: %q", FlagEmoji("fi-hel"))
	}
}

func TestProfileFragment(t *testing.T) {
	got := ProfileFragment("ru-msk", "1.2.3.4", 0)
	if got != "🇷🇺 ru-msk" {
		t.Fatalf("got %q", got)
	}
	got = ProfileFragment("", "203.0.113.11", 0)
	if got != "nd-secondary-203-0-113-11" {
		t.Fatalf("legacy got %q", got)
	}
}

func TestFromHostname(t *testing.T) {
	if FromHostname("nd-egress-ru-msk") != "ru-msk" {
		t.Fatal(FromHostname("nd-egress-ru-msk"))
	}
	if FromHostname("nd-secondary-fi-hel.example") != "fi-hel" {
		t.Fatal(FromHostname("nd-secondary-fi-hel.example"))
	}
	if FromHostname("nd-primary") != "" {
		t.Fatal("no region")
	}
}

func TestHostname(t *testing.T) {
	if Hostname("secondary", "ru-msk") != "nd-egress-ru-msk" {
		t.Fatal(Hostname("secondary", "ru-msk"))
	}
	if !MatchesHostname("nd-egress-ru-msk", "secondary", "ru-msk") {
		t.Fatal("match")
	}
	if !MatchesHostname("nd-secondary-ru-msk", "secondary", "ru-msk") {
		t.Fatal("legacy match")
	}
}
