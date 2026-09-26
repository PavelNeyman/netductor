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
			{"vps", "VPS (эта машина)", "Установка на сервере", "Вы на Debian VPS под root: install стека (sing-box, API, bot…). Не выбирайте с Mac."},
			{"workstation", "Workstation (Mac/PC)", "Удалённый деплой", "С ноутбука: master/wizard → primary/secondary/edge по SSH. Remote в Настройках."},
			{"openwrt", "OpenWrt / edge", "Роутер", "Настройка агента на OpenWrt/Cudy/RPi (локально или через SSH)."},
			{"operator", "Оператор", "Уже развёрнуто", "Управление живой системой: VPN, fleet, backup, doctor. Local или Remote."},
		}
	}
	return []menuEntry{
		{"vps", "VPS (this host)", "Server-side install", "You are on the Debian VPS as root: install stack. Do not pick this on a Mac."},
		{"workstation", "Workstation (Mac/PC)", "Remote deploy", "From laptop: wizard deploys primary/secondary/edge over SSH. Set Remote in Settings."},
		{"openwrt", "OpenWrt / edge", "Router", "Provision edge agent on OpenWrt/Cudy/RPi (local or SSH)."},
		{"operator", "Operator", "Already installed", "Operate a live system: VPN, fleet, backup, doctor. Local or Remote."},
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
			{"dns-list", "DNS списки", "Blocky", "list id + on/off."},
			{"dns-on", "DNS on", "id", "Включить список."},
			{"dns-off", "DNS off", "id", "Выключить список."},
			{"dns-reload", "DNS reload", "Blocky", "Применить списки."},
			{"backup-now", "Бэкап", "Сейчас", "На ноде (local/remote)."},
			{"backup-list", "Бэкапы", "Файлы", "Список файлов."},
			{"backup-schedule", "Расписание бэкапа", "GET", "Текущее расписание."},
			{"git-repos", "Git", "Репозитории", "list repos."},
			{"git-pipelines", "Git pipelines", "CI", "list pipelines."},
			{"registry-status", "Registry", "status", "Docker registry."},
			{"domain-show", "Domain", "show", "base / LE status."},
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
		{"dns-list", "DNS lists", "Blocky", "list id + on/off."},
		{"dns-on", "DNS on", "id", "Enable list."},
		{"dns-off", "DNS off", "id", "Disable list."},
		{"dns-reload", "DNS reload", "Blocky", "Apply lists."},
		{"backup-now", "Backup now", "Run", "On node (local/remote)."},
		{"backup-list", "Backups", "Files", "List backup files."},
		{"backup-schedule", "Backup schedule", "GET", "Current schedule."},
		{"git-repos", "Git", "Repos", "list repos."},
		{"git-pipelines", "Git pipelines", "CI", "list pipelines."},
		{"registry-status", "Registry", "status", "Docker registry."},
		{"domain-show", "Domain", "show", "base / LE status."},
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
	"backup.run": {"Run backup now", "Сделать бэкап сейчас"},
	"backup.list": {"List backups", "Список бэкапов"},
	"recover.from_secondary": {"Recover from secondary", "Восстановить с secondary"},

	"ssh_user":           {"SSH user", "SSH пользователь"},
	"ssh_password":       {"SSH password", "SSH пароль"},
	"ssh_password_first": {"SSH password (first login only)", "SSH пароль (только первый вход)"},
	"ssh_password_ifkey": {"SSH password (if no key)", "SSH пароль (если нет ключа)"},
	"ssh_port":           {"SSH port", "SSH порт"},
	"ssh_key_path":       {"SSH private key path", "Путь к SSH private key"},
	"gen_ssh_key":        {"Generate new SSH key?", "Сгенерировать новый SSH-ключ?"},
	"gen_ssh_key_desc":   {"No = use path below", "Нет = путь ниже"},
	"key_use_passphrase": {"Protect key with passphrase?", "Защитить ключ passphrase?"},
	"key_use_passphrase_desc": {"Yes = enter phrase next; needed for ssh-agent later", "Да = дальше фраза; потом удобно ssh-add"},
	"key_passphrase":     {"Key passphrase", "Passphrase ключа"},
	"key_passphrase_confirm": {"Confirm passphrase", "Подтвердите passphrase"},
	"key_passphrase_mismatch": {"Passphrases do not match", "Passphrase не совпадают"},
	"host":               {"Host / IP", "Host / IP"},
	"primary_host":       {"Primary VPS host / IP", "Primary VPS host / IP"},
	"primary_host_desc":  {"Empty = install on THIS machine", "Пусто = install на ЭТОЙ машине"},
	"secondary_host":     {"Secondary (RU) IP / host", "Secondary (RU) IP / host"},
	"router_lan":         {"Router LAN IP", "LAN IP роутера"},
	"password":           {"Password", "Пароль"},
	"device_id":          {"Device ID", "Device ID"},
	"edge_device_id":     {"Edge device id", "Edge device id"},
	"router_lan_ip":     {"Router LAN IP", "LAN IP роутера"},
	"agent_arch":         {"Agent arch", "Arch агента"},
	"primary_mtls":       {"Primary mTLS URL", "Primary mTLS URL"},
	"reality_sni":        {"Reality SNI", "Reality SNI"},
	"with_git_registry": {"Git + registry add-on", "Git + registry"},
	"tg_bot_token": {"Telegram bot token", "Токен TG-бота"},
	"tg_admin_id": {"Telegram admin id", "TG admin id"},
	"fleet_do_primary": {"Deploy primary", "Деплоить primary"},
	"fleet_do_secondary": {"Deploy secondary", "Деплоить secondary"},
	"domain_base":        {"Domain base (optional)", "Домен base (опц.)"},
	"cf_proxy":          {"CF proxy i. (orange)", "CF proxy i. (orange)"},
	"le_email":           {"LE email (if domain)", "LE email (если домен)"},
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
	"opt_addons":         {"Add-ons (Lampac, …)", "Дополнения (Lampac, …)"},
	"addons_title":       {"Optional add-ons on primary", "Опциональные дополнения на primary"},
	"addons_desc":        {"Runs on primary over SSH (key from TUI settings)", "Ставится на primary по SSH (ключ из настроек TUI)"},
	"addon_lampac":       {"Install Lampac (Docker, localhost:9118)", "Установить Lampac (Docker, localhost:9118)"},
	"addon_lampac_desc":  {"Needs Docker; not exposed on WAN", "Нужен Docker; наружу не открывается"},
	"with_lampac_primary":{"Also install Lampac after primary?", "Сразу поставить Lampac после primary?"},

	"net_configure":      {"Configure LAN/Wi‑Fi/WAN now?", "Настроить LAN/Wi‑Fi/WAN сейчас?"},
	"net_configure_desc": {"LAN/DHCP/Wi‑Fi/WAN (dhcp|static|pppoe). Wi‑Fi: fill one band → both; or set 2.4 and 5 separately", "LAN/DHCP/Wi‑Fi/WAN (dhcp|static|pppoe). Wi‑Fi: одна полоса → обе; или 2.4 и 5 раздельно"},
	"lan_ip":             {"LAN IP (router)", "LAN IP (роутер)"},
	"lan_mask":           {"LAN netmask", "Маска LAN"},
	"dhcp_start":         {"DHCP pool start (host part)", "DHCP start (последний октет)"},
	"dhcp_limit":         {"DHCP pool size", "Размер пула DHCP"},
	"wifi_ssid_24":       {"Wi‑Fi SSID 2.4 GHz", "Wi‑Fi SSID 2.4 ГГц"},
	"wifi_ssid_24_desc":  {"Empty 5 GHz fields → same as 2.4; empty 2.4 → same as 5", "Пустые поля 5 ГГц → как 2.4; пустые 2.4 → как 5"},
	"wifi_key_24":        {"Wi‑Fi password 2.4 GHz", "Пароль Wi‑Fi 2.4 ГГц"},
	"wifi_ssid_5":        {"Wi‑Fi SSID 5 GHz (optional)", "Wi‑Fi SSID 5 ГГц (опционально)"},
	"wifi_ssid_5_desc":   {"Leave empty to copy SSID/password from 2.4 GHz", "Пусто = скопировать SSID/пароль с 2.4 ГГц"},
	"wifi_key_5":         {"Wi‑Fi password 5 GHz (optional)", "Пароль Wi‑Fi 5 ГГц (опционально)"},
	"wifi_key":           {"Wi‑Fi password", "Пароль Wi‑Fi"},
	"wan_proto":          {"WAN mode", "Режим WAN"},
	"wan_proto_desc":     {"dhcp | static | pppoe", "dhcp | static | pppoe"},
	"pppoe_user":         {"PPPoE username", "PPPoE логин"},
	"pppoe_pass":         {"PPPoE password", "PPPoE пароль"},
	"pppoe_service":      {"PPPoE service (optional)", "PPPoE service (опционально)"},
	"pppoe_ac":           {"PPPoE AC (optional)", "PPPoE AC (опционально)"},
	"wan_ip":             {"WAN static IP", "WAN static IP"},
	"wan_mask":           {"WAN netmask", "Маска WAN"},
	"wan_gateway":        {"WAN gateway", "Шлюз WAN"},
	"wan_dns":            {"WAN DNS", "DNS WAN"},
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
