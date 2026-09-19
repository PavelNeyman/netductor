package main

import (
	"os"
	"os/exec"
	"strings"
)

type tuiLang int

const (
	langEN tuiLang = iota
	langRU
)

type helpChip struct{ Key, Label string }

type locPack struct {
	App string
	TabWizard, TabTools, TabOps, TabSettings, TabMode string
	HelpWizard, HelpTools, HelpOps, HelpSettings, HelpModeBar, HelpOutput []helpChip
}

func detectLang() tuiLang {
	// Explicit override for TUI / CI
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("NETDUCTOR_LANG"))); v != "" {
		if strings.HasPrefix(v, "ru") {
			return langRU
		}
		return langEN
	}
	// macOS UI language list is more trustworthy than LANG in modern terminals
	// (Ghostty/iTerm often export LANG=en_US.UTF-8 while system is Russian).
	if b, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
		v := strings.ToLower(string(b))
		// first preferred language roughly appears first in the plist dump
		if idxRu, idxEn := strings.Index(v, "ru"), strings.Index(v, "en"); idxRu >= 0 && (idxEn < 0 || idxRu < idxEn) {
			return langRU
		}
		if strings.Contains(v, `"ru"`) || strings.Contains(v, "ru-") || strings.Contains(v, "ru_") {
			return langRU
		}
	}
	if b, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output(); err == nil {
		v := strings.ToLower(strings.TrimSpace(string(b)))
		if strings.HasPrefix(v, "ru") {
			return langRU
		}
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
		if v == "" || v == "c" || v == "posix" {
			continue
		}
		if strings.HasPrefix(v, "ru") || strings.Contains(v, ".ru_") || strings.Contains(v, "_ru") || strings.Contains(v, "ru_") {
			return langRU
		}
	}
	return langEN
}

func l10n(lang tuiLang) locPack { return locFor(lang) }

func locFor(lang tuiLang) locPack {
	chips := func(ru bool) (wiz, tools, ops, set, mode, out []helpChip) {
		if ru {
			c := []helpChip{{"tab", "вкладки"}, {"↑↓", "пункт"}, {"enter", "выбор"}, {"esc", "назад"}, {"l", "язык"}, {"^c", "выход"}}
			return c, c, c, c, []helpChip{{"↑↓", "режим"}, {"enter", "выбрать"}, {"l", "язык"}, {"^c", "выход"}},
				[]helpChip{{"esc", "назад"}, {"enter", "назад"}, {"l", "язык"}, {"^c", "выход"}}
		}
		c := []helpChip{{"tab", "tabs"}, {"↑↓", "item"}, {"enter", "select"}, {"esc", "back"}, {"l", "lang"}, {"^c", "quit"}}
		return c, c, c, c, []helpChip{{"↑↓", "mode"}, {"enter", "select"}, {"l", "lang"}, {"^c", "quit"}},
			[]helpChip{{"esc", "back"}, {"enter", "back"}, {"l", "lang"}, {"^c", "quit"}}
	}
	if lang == langRU {
		w, t, o, s, m, out := chips(true)
		return locPack{
			App: "netductor",
			TabWizard: "Мастер", TabTools: "Инструменты", TabOps: "Операции", TabSettings: "Настройки", TabMode: "Режим",
			HelpWizard: w, HelpTools: t, HelpOps: o, HelpSettings: s, HelpModeBar: m, HelpOutput: out,
		}
	}
	w, t, o, s, m, out := chips(false)
	return locPack{
		App: "netductor",
		TabWizard: "Wizard", TabTools: "Tools", TabOps: "Ops", TabSettings: "Settings", TabMode: "Mode",
		HelpWizard: w, HelpTools: t, HelpOps: o, HelpSettings: s, HelpModeBar: m, HelpOutput: out,
	}
}

type menuEntry struct {
	ID, Title, Short, Detail string
}

func modeEntries(lang tuiLang) []menuEntry {
	if lang == langRU {
		return []menuEntry{
			{"vps", "VPS (эта машина)", "Install на сервере", "Запуск на самой VPS: установка стека. Не с Mac."},
			{"workstation", "Workstation", "Mac/PC", "С ноутбука: remote SSH к ноде + локальная сборка/edge."},
			{"openwrt", "OpenWrt / edge", "Роутер", "Provision агента на OpenWrt/RPi."},
			{"operator", "Управление", "Уже установлено", "Операции на работающей системе (local или через remote в Настройках)."},
		}
	}
	return []menuEntry{
		{"vps", "VPS (this host)", "Server install", "Run on the VPS itself: stack install. Not from Mac."},
		{"workstation", "Workstation", "Mac/PC", "From laptop: remote SSH + local build/edge."},
		{"openwrt", "OpenWrt / edge", "Router", "Provision edge agent on OpenWrt/RPi."},
		{"operator", "Manage", "Already installed", "Operate a running node (local or remote from Settings)."},
	}
}

func settingsEntries(lang tuiLang) []menuEntry {
	if lang == langRU {
		return []menuEntry{
			{"cfg-remote", "Подключение к VPS", "SSH target", "Адрес, пользователь, ключ и/или пароль. Сохраняется в ~/.config/netductor/tui.yaml."},
			{"cfg-lang", "Язык интерфейса", "auto → ru/en", "По умолчанию язык системы (LANG/AppleLocale). Клавиша l или этот пункт — явно ru/en."},
			{"cfg-save", "Сохранить настройки", "tui.yaml", "Записать текущие параметры на диск."},
			{"cfg-clear-remote", "Сбросить remote", "Local only", "Убрать SSH target — команды только локально."},
			{"change-mode", "Сменить режим", "Mode", "VPS / Workstation / OpenWrt / Управление."},
		}
	}
	return []menuEntry{
		{"cfg-remote", "VPS connection", "SSH target", "Host, user, key and/or password. Saved to ~/.config/netductor/tui.yaml."},
		{"cfg-lang", "UI language", "auto → ru/en", "Default: system locale (LANG/AppleLocale). Key l or this item forces ru/en."},
		{"cfg-save", "Save settings", "tui.yaml", "Write current settings to disk."},
		{"cfg-clear-remote", "Clear remote", "Local only", "Drop SSH target — local commands only."},
		{"change-mode", "Change mode", "Mode", "VPS / Workstation / OpenWrt / Manage."},
	}
}

func toolsEntries(mode runMode, lang tuiLang) []menuEntry {
	ru := lang == langRU
	var e []menuEntry
	if mode == modeVPS {
		if ru {
			e = append(e,
				menuEntry{"install", "Install стека", "На этой VPS", "Полный install primary. Запускается внутри TUI, без выхода."},
				menuEntry{"prepare", "Подготовка", "apt/docker", "Подготовка системы (если поддерживается CLI)."},
				menuEntry{"apply-lampac", "Lampac", "Primary", "Docker Lampac на этой машине."},
			)
		} else {
			e = append(e,
				menuEntry{"install", "Install stack", "This VPS", "Full primary install. Runs inside TUI, no exit."},
				menuEntry{"prepare", "Prepare", "apt/docker", "System prep if CLI supports it."},
				menuEntry{"apply-lampac", "Lampac", "Primary", "Docker Lampac on this host."},
			)
		}
	}
	if mode == modeOpenWRT || mode == modeWorkstation {
		if ru {
			e = append(e, menuEntry{"owrt-install", "Edge provision", "OpenWrt SSH", "Агент на роутер: host, user, password, id, server — форма внутри TUI."})
		} else {
			e = append(e, menuEntry{"owrt-install", "Edge provision", "OpenWrt SSH", "Router agent: host, user, password, id, server — in-TUI form."})
		}
	}
	if mode == modeWorkstation {
		if ru {
			e = append(e, menuEntry{"build", "Сборка бинарей", "Локально", "GOOS/GOARCH — форма в TUI, без выхода."})
		} else {
			e = append(e, menuEntry{"build", "Build binaries", "Local", "GOOS/GOARCH — in-TUI form."})
		}
	}
	if ru {
		e = append(e,
			menuEntry{"change-mode", "Сменить режим", "Mode", "VPS / Workstation / OpenWrt / Управление."},
		)
	} else {
		e = append(e,
			menuEntry{"change-mode", "Change mode", "Mode", "VPS / Workstation / OpenWrt / Manage."},
		)
	}
	return e
}

func opsEntries(mode runMode, lang tuiLang) []menuEntry {
	if lang == langRU {
		return []menuEntry{
			{"status", "Статус сервисов", "systemd", "local или remote из Настроек."},
			{"doctor", "Doctor", "Проверки", "Health checks."},
			{"fleet-status", "Флот", "primary/secondary", "Роли нод."},
			{"vpn-list", "VPN пользователи", "Список", "Имя, UUID."},
			{"vpn-add", "Добавить VPN user", "Форма", "Имя (+ заметка) — внутри TUI."},
			{"relay-sync", "VPN → secondary", "config_ver", "UUID на RU entry."},
			{"relay-status", "Secondary", "Agent", "Heartbeat / sing-box."},
			{"nodes-list", "Реестр нод", "nodes", "Hostname, role, IP."},
			{"edge-list", "Edge devices", "OpenWrt", "Список / pending."},
			{"nvr-status", "NVR статус", "камеры", "config, storage, cameras."},
			{"nvr-go2rtc", "NVR go2rtc", "yaml", "Сгенерировать go2rtc.yaml."},
			{"probe", "Probes", "Связность", "API/VPN probes."},
			{"backup-now", "Бэкап", "Сейчас", "На ноде (local/remote)."},
			{"audit-tail", "Audit", "События", "Последние события."},
			{"disable-legacy", "Disable legacy", "Timers", "Старые sync/failover timers."},
			{"change-mode", "Сменить режим", "Mode", "Выбор режима работы."},
		}
	}
	return []menuEntry{
		{"status", "Service status", "systemd", "local or remote from Settings."},
		{"doctor", "Doctor", "Health", "Health checks."},
		{"fleet-status", "Fleet", "primary/secondary", "Node roles."},
		{"vpn-list", "VPN users", "List", "Name, UUID."},
		{"vpn-add", "Add VPN user", "Form", "Name (+ note) — in-TUI."},
		{"relay-sync", "VPN → secondary", "config_ver", "UUIDs to RU entry."},
		{"relay-status", "Secondary", "Agent", "Heartbeat / sing-box."},
		{"nodes-list", "Nodes registry", "nodes", "Hostname, role, IP."},
		{"edge-list", "Edge devices", "OpenWrt", "List / pending."},
		{"nvr-status", "NVR status", "cameras", "config, storage, cameras."},
		{"nvr-go2rtc", "NVR go2rtc", "yaml", "Write go2rtc.yaml."},
		{"probe", "Probes", "Connectivity", "API/VPN probes."},
		{"backup-now", "Backup now", "Run", "On node (local/remote)."},
		{"audit-tail", "Audit", "Events", "Recent events."},
		{"disable-legacy", "Disable legacy", "Timers", "Old sync/failover timers."},
		{"change-mode", "Change mode", "Mode", "Work mode picker."},
	}
}
