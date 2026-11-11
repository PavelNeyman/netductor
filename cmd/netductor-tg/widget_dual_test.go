package main

import (
	"os"
	"testing"
)

func TestDualPairLifecycleState(t *testing.T) {
	const chat int64 = 424242
	forgetDual(chat)
	if _, ok := getDual(chat); ok {
		t.Fatal("expected empty pair")
	}
	setDual(chat, dualPair{InfoID: 11, NavID: 22, InfoShell: shellSpark, NavShell: shellHub})
	p, ok := getDual(chat)
	if !ok || p.InfoID != 11 || p.NavID != 22 {
		t.Fatalf("getDual: %+v ok=%v", p, ok)
	}
	forgetDual(chat)
	if _, ok := getDual(chat); ok {
		t.Fatal("forgetDual left state")
	}
}

func TestInfoShellForScreen(t *testing.T) {
	if infoShellForScreen("home") != shellSpark {
		t.Fatal("home should spark")
	}
	if infoShellForScreen("status") != shellSpark {
		t.Fatal("status should spark")
	}
	if infoShellForScreen("users") != shellHub {
		t.Fatal("users should hub")
	}
	if infoShellForScreen("fleet") != shellHub {
		t.Fatal("fleet should hub")
	}
}

func TestBuildInfoDataScreens(t *testing.T) {
	screens := []string{"home", "status", "fleet", "users", "operator", "tools", "updates", "ncard", "ucard", "git"}
	for _, s := range screens {
		cd, err := buildInfoData(s)
		if err != nil {
			t.Errorf("%s: %v", s, err)
			continue
		}
		if len(cd) > 1024 {
			t.Errorf("%s chat_data %d > 1024", s, len(cd))
		}
	}
}

func TestDualEnabledRequiresBothFlags(t *testing.T) {
	oldW := os.Getenv("NETDUCTOR_TG_WIDGETS")
	oldD := os.Getenv("NETDUCTOR_TG_DUAL")
	defer func() {
		_ = os.Setenv("NETDUCTOR_TG_WIDGETS", oldW)
		_ = os.Setenv("NETDUCTOR_TG_DUAL", oldD)
	}()
	_ = os.Setenv("NETDUCTOR_TG_WIDGETS", "")
	_ = os.Setenv("NETDUCTOR_TG_DUAL", "1")
	if dualEnabled() {
		t.Fatal("dual must require widgets flag")
	}
	_ = os.Setenv("NETDUCTOR_TG_WIDGETS", "1")
	_ = os.Setenv("NETDUCTOR_TG_DUAL", "")
	if dualEnabled() {
		t.Fatal("dual must require DUAL=1")
	}
	_ = os.Setenv("NETDUCTOR_TG_DUAL", "1")
	if !dualEnabled() {
		t.Fatal("expected dual on")
	}
}

func TestOpenAppScreenDualRequiresFlag(t *testing.T) {
	// dual off → openAppScreen must not call dual (smoke: home build only)
	oldW := os.Getenv("NETDUCTOR_TG_WIDGETS")
	oldD := os.Getenv("NETDUCTOR_TG_DUAL")
	defer func() {
		_ = os.Setenv("NETDUCTOR_TG_WIDGETS", oldW)
		_ = os.Setenv("NETDUCTOR_TG_DUAL", oldD)
	}()
	_ = os.Setenv("NETDUCTOR_TG_WIDGETS", "1")
	_ = os.Setenv("NETDUCTOR_TG_DUAL", "")
	if dualEnabled() {
		t.Fatal("dual should be off")
	}
	cd, err := buildAppScreen("home")
	if err != nil || len(cd) > 1024 {
		t.Fatalf("home %v len=%d", err, len(cd))
	}
}

func TestBuildInfoDataAllParityScreens(t *testing.T) {
	screens := []string{
		"home", "status", "users", "ulist", "fleet", "tools", "operator", "help",
		"channel", "netmap", "actions", "sessions", "digest", "incident", "reboot",
		"nodes", "secondary", "addons", "seclist", "edge", "routers", "sites", "pending",
		"media", "lampac", "canary", "nvr", "data", "backup", "logs", "dns",
		"adv", "probes", "stack", "doctor", "certs", "fw", "cleanup",
		"versions", "updates", "ssh", "git", "session", "alerts", "admin", "audit", "svcpaths",
		"devices", "presets", "mtls", "secops", "ucard", "ncard",
	}
	for _, s := range screens {
		cd, err := buildInfoData(s)
		if err != nil {
			t.Errorf("info %s: %v", s, err)
			continue
		}
		if len(cd) > 1024 {
			t.Errorf("info %s len %d", s, len(cd))
		}
	}
}
