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
	App                                                                   string
	TabWizard, TabTools, TabOps, TabSettings, TabMode                     string
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
			App:       "netductor",
			TabWizard: "Мастер", TabTools: "Инструменты", TabOps: "Операции", TabSettings: "Настройки", TabMode: "Режим",
			HelpWizard: w, HelpTools: t, HelpOps: o, HelpSettings: s, HelpModeBar: m, HelpOutput: out,
		}
	}
	w, t, o, s, m, out := chips(false)
	return locPack{
		App:       "netductor",
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
			{"secondary-sync", "VPN → secondary", "config_ver", "UUID на RU entry."},
			{"secondary-status", "Secondary", "Agent", "Heartbeat / sing-box."},
			{"nodes-list", "Реестр нод", "nodes", "Hostname, role, IP."},
			{"edge-list", "Edge devices", "OpenWrt", "Список устройств."},
			{"edge-pending", "Edge pending", "approve", "Ожидают approve."},
			{"edge-recovery", "Edge recovery", "код", "One-time код для LAN-страницы."},
			{"edge-register", "Edge register", "device_id", "Pre-declare pending."},
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
		{"secondary-sync", "VPN → secondary", "config_ver", "UUIDs to RU entry."},
		{"secondary-status", "Secondary", "Agent", "Heartbeat / sing-box."},
		{"nodes-list", "Nodes registry", "nodes", "Hostname, role, IP."},
		{"edge-list", "Edge devices", "OpenWrt", "Device list."},
		{"edge-pending", "Edge pending", "approve", "Awaiting approve."},
		{"edge-recovery", "Edge recovery", "code", "One-time LAN recovery code."},
		{"edge-register", "Edge register", "device_id", "Pre-declare pending."},
		{"nvr-status", "NVR status", "cameras", "config, storage, cameras."},
		{"nvr-go2rtc", "NVR go2rtc", "yaml", "Write go2rtc.yaml."},
		{"probe", "Probes", "Connectivity", "API/VPN probes."},
		{"backup-now", "Backup now", "Run", "On node (local/remote)."},
		{"audit-tail", "Audit", "Events", "Recent events."},
		{"disable-legacy", "Disable legacy", "Timers", "Old sync/failover timers."},
		{"change-mode", "Change mode", "Mode", "Work mode picker."},
	}
}

// TT returns en or ru string for the active TUI language.
func TT(lang tuiLang, en, ru string) string {
	if lang == langRU {
		return ru
	}
	return en
}

// formDict common form labels (key → en, ru). Prefer FormT over ad-hoc TT for shared keys.
var formDict = map[string][2]string{
	"ssh_user":           {"SSH user", "SSH пользователь"},
	"ssh_password":       {"SSH password", "SSH пароль"},
	"ssh_password_first": {"SSH password (first login only)", "SSH пароль (только первый вход)"},
	"ssh_password_ifkey": {"SSH password (if no key)", "SSH пароль (если нет ключа)"},
	"ssh_port":           {"SSH port", "SSH порт"},
	"ssh_key_path":       {"SSH private key path", "Путь к SSH private key"},
	"gen_ssh_key":        {"Generate new SSH key?", "Сгенерировать новый SSH-ключ?"},
	"gen_ssh_key_desc":   {"No = use path below", "Нет = путь ниже"},
	"host":               {"Host / IP", "Host / IP"},
	"primary_host":       {"Primary VPS host / IP", "Primary VPS host / IP"},
	"primary_host_desc":  {"Empty = install on THIS machine", "Пусто = install на ЭТОЙ машине"},
	"secondary_host":     {"Secondary (RU) IP / host", "Secondary (RU) IP / host"},
	"router_lan":         {"Router LAN IP", "LAN IP роутера"},
	"password":           {"Password", "Пароль"},
	"device_id":          {"Device ID", "Device ID"},
	"edge_device_id":     {"Edge device id", "Edge device id"},
	"agent_arch":         {"Agent arch", "Arch агента"},
	"primary_mtls":       {"Primary mTLS URL", "Primary mTLS URL"},
	"reality_sni":        {"Reality SNI", "Reality SNI"},
	"tg_token":           {"Telegram bot token", "Токен Telegram-бота"},
	"tg_admin":           {"Telegram admin user id", "Telegram admin user id"},
	"yes":                {"Yes", "Да"},
	"no":                 {"No", "Нет"},
	"cancel":             {"Cancel", "Отмена"},
	"apply":              {"Apply", "Применить"},
	"refresh":            {"Refresh", "Обновить"},
	"push":               {"Push", "Залить"},
	"required":           {"required", "обязательно"},
	"setup_title":        {"Setup wizard — what are we configuring?", "Мастер — что настраиваем?"},
	"setup_desc":         {"Questions first, then fully automatic apply", "Сначала вопросы, затем авто-применение"},
	"opt_primary":        {"Primary VPS (abroad control plane)", "Primary VPS (control plane за рубежом)"},
	"opt_secondary":      {"Secondary VPS (RU VPN entry only)", "Secondary VPS (только RU VPN entry)"},
	"opt_openwrt":        {"OpenWrt router / RPi (edge agent)", "OpenWrt / RPi (edge agent)"},
	"opt_nvr":            {"Cameras / NVR", "Камеры / NVR"},
	"opt_mikrotik":       {"MikroTik (ROS routes / site)", "MikroTik (ROS / сайт)"},
	"nvr_action":         {"NVR action", "Действие NVR"},
	"cam_name":           {"Camera name (add)", "Имя камеры (add)"},
	"cam_ip":             {"Camera IP", "IP камеры"},
	"cam_mac":            {"Camera MAC", "MAC камеры"},
	"cam_pass":           {"Camera password", "Пароль камеры"},
	"run_install_here":   {"Run netductor install here?", "Запустить netductor install здесь?"},
	"host_pass_required": {"host and password required", "нужны host и пароль"},
	"host_id_required":   {"host and device id required", "нужны host и device id"},
	"set_primary_first":  {"set primary remote in Settings / primary wizard first", "сначала primary в Настройках / мастере primary"},
}

// FormT looks up formDict; falls back to TT(lang, key, key) if missing.
func FormT(lang tuiLang, key string) string {
	if v, ok := formDict[key]; ok {
		return TT(lang, v[0], v[1])
	}
	return key
}
