package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type tuiLang int

const (
	langEN tuiLang = iota
	langRU
)

func localeLooksRU(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	return strings.HasPrefix(s, "ru") || strings.Contains(s, "ru_ru") || strings.Contains(s, "russian")
}

func detectTUILang() tuiLang {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			continue
		}
		for _, part := range strings.Split(v, ":") {
			if localeLooksRU(part) {
				return langRU
			}
		}
		if localeLooksRU(v) {
			return langRU
		}
	}
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output(); err == nil && localeLooksRU(string(out)) {
			return langRU
		}
		if out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
			if strings.Contains(strings.ToLower(string(out)), "ru") {
				return langRU
			}
		}
	}
	return langEN
}

type helpChip struct{ Key, Label string }

type tuiL10n struct {
	App, TabWizard, TabTools, TabOps, TabMode string
	Suggested, SelectMode, HelpMode, Output, Detail string
	ChipUp, ChipEnter, ChipEsc, ChipNum, ChipLang, ChipQuit, ChipTab string
	HelpWizard, HelpTools, HelpOps, HelpModeBar, HelpOutput []helpChip
}

func l10n(lang tuiLang) tuiL10n {
	if lang == langRU {
		return tuiL10n{
			App: "Netductor", TabWizard: "Мастер", TabTools: "Инструменты", TabOps: "Операции", TabMode: "Режим",
			Suggested: "Рекомендуется", SelectMode: "Выбор режима",
			HelpMode: "Выберите режим. Установка начнётся только после подтверждения в мастере.",
			Output: "Вывод", Detail: "Описание",
			ChipUp: "↑↓", ChipEnter: "↵", ChipEsc: "Esc", ChipNum: "1–9", ChipLang: "L", ChipQuit: "^C", ChipTab: "Tab",
			HelpWizard: []helpChip{{"Tab", "вкладки"}, {"↑↓", "поле"}, {"Esc", "отмена"}, {"L", "язык"}, {"^C", "выход"}},
			HelpTools:  []helpChip{{"Tab", "вкладки"}, {"↑↓", "пункт"}, {"↵", "запуск"}, {"Esc", "назад"}, {"1–9", "быстро"}, {"L", "язык"}, {"^C", "выход"}},
			HelpOps:    []helpChip{{"Tab", "вкладки"}, {"↑↓", "пункт"}, {"↵", "запуск"}, {"Esc", "назад"}, {"L", "язык"}, {"^C", "выход"}},
			HelpModeBar: []helpChip{{"↑↓", "режим"}, {"↵", "выбрать"}, {"1–9", "быстро"}, {"L", "язык"}, {"^C", "выход"}},
			HelpOutput: []helpChip{{"Esc", "назад"}, {"↵", "назад"}, {"L", "язык"}, {"^C", "выход"}},
		}
	}
	return tuiL10n{
		App: "Netductor", TabWizard: "Wizard", TabTools: "Tools", TabOps: "Ops", TabMode: "Mode",
		Suggested: "Suggested", SelectMode: "Select mode",
		HelpMode: "Pick a mode. Nothing installs until you confirm in the wizard.",
		Output: "Output", Detail: "Details",
		ChipUp: "↑↓", ChipEnter: "↵", ChipEsc: "Esc", ChipNum: "1–9", ChipLang: "L", ChipQuit: "^C", ChipTab: "Tab",
		HelpWizard: []helpChip{{"Tab", "tabs"}, {"↑↓", "field"}, {"Esc", "cancel"}, {"L", "lang"}, {"^C", "quit"}},
		HelpTools:  []helpChip{{"Tab", "tabs"}, {"↑↓", "item"}, {"↵", "run"}, {"Esc", "back"}, {"1–9", "jump"}, {"L", "lang"}, {"^C", "quit"}},
		HelpOps:    []helpChip{{"Tab", "tabs"}, {"↑↓", "item"}, {"↵", "run"}, {"Esc", "back"}, {"L", "lang"}, {"^C", "quit"}},
		HelpModeBar: []helpChip{{"↑↓", "mode"}, {"↵", "select"}, {"1–9", "jump"}, {"L", "lang"}, {"^C", "quit"}},
		HelpOutput: []helpChip{{"Esc", "back"}, {"↵", "back"}, {"L", "lang"}, {"^C", "quit"}},
	}
}

// menuEntry is one selectable row with short + long help (BIOS-style).
type menuEntry struct {
	ID, Title, Short, Detail string
}

func modeEntries(lang tuiLang) []menuEntry {
	if lang == langRU {
		return []menuEntry{
			{"vps", "Настройка VPS", "Стек на сервере", "Установка и настройка control plane на Debian/VPS: sing-box (VLESS/HY2), Blocky, API, Telegram-бот, бэкапы, hardening. Идемпотентно — повторный запуск безопасен."},
			{"openwrt", "OpenWrt / edge", "Агент на роутере", "Вынос агента на OpenWrt/RPi: outbound enroll на primary, approve, шаблоны UCI (LAN/Wi‑Fi). Интернет на роутере для enroll может появиться позже — агент ждёт с backoff."},
			{"workstation", "Рабочая станция", "Mac/PC оператора", "Сборка бинарников, brew/tap, bootstrap one-liner для VPS, MikroTik manage, SSH TOFU. Не ставит серверные сервисы на этот Mac."},
			{"operator", "Панель оператора", "День-2 операции", "Пользователи VPN, сессии, edge devices, doctor, probes, secondary/relay, audit — без полного install."},
		}
	}
	return []menuEntry{
		{"vps", "VPS setup", "Server stack", "Install and configure the control plane on Debian/VPS: sing-box (VLESS/HY2), Blocky, API, Telegram bot, backups, hardening. Idempotent — safe to re-run."},
		{"openwrt", "OpenWrt / edge", "Router agent", "Deploy netductor-agent on OpenWrt/RPi: outbound enroll to primary, approve, UCI templates (LAN/Wi‑Fi). WAN can come later — agent retries enroll with backoff."},
		{"workstation", "Workstation", "Operator Mac/PC", "Build binaries, brew/tap, VPS bootstrap one-liner, MikroTik manage, SSH TOFU. Does not install server services on this machine."},
		{"operator", "Operator panel", "Day-2 ops", "VPN users, sessions, edge devices, doctor, probes, secondary status, audit — without full install."},
	}
}

func toolsEntries(mode runMode, lang tuiLang) []menuEntry {
	ru := lang == langRU
	var modeExtra []menuEntry
	switch mode {
	case modeVPS:
		if ru {
			modeExtra = []menuEntry{
				{"install", "Install / upgrade", "Стек на VPS", "Идемпотентная установка компонентов (dirs, hardening, sing-box, blocky, vpn, api, bot, backup)."},
				{"apply-lampac", "Lampac", "Addon", "Поставить/обновить Lampac (Docker) на предпочитаемой ноде флота."},
				{"hostname", "Hostname", "nd-…", "Задать hostname формата nd-<role>-<marker> и зарегистрировать в реестре нод."},
			}
		} else {
			modeExtra = []menuEntry{
				{"install", "Install / upgrade", "VPS stack", "Idempotent install of components (dirs, hardening, sing-box, blocky, vpn, api, bot, backup)."},
				{"apply-lampac", "Lampac", "Addon", "Install/update Lampac (Docker) on preferred fleet node."},
				{"hostname", "Hostname", "nd-…", "Set hostname nd-<role>-<marker> and register in node registry."},
			}
		}
	case modeOpenWRT:
		if ru {
			modeExtra = []menuEntry{
				{"agent-help", "Агент — инструкция", "Установка", "Как положить netductor-agent на роутер, config SERVER/TOKEN/DEVICE_ID, enroll."},
				{"owrt-install", "install-openwrt.sh", "Скрипт", "Запуск site-скрипта, если он есть на устройстве."},
				{"agent-cfg", "Конфиг агента", "Проверка", "Есть ли /etc/netductor-agent/config."},
			}
		} else {
			modeExtra = []menuEntry{
				{"agent-help", "Agent instructions", "Install", "How to place netductor-agent on the router, SERVER/TOKEN/DEVICE_ID, enroll."},
				{"owrt-install", "install-openwrt.sh", "Script", "Run site script if present on the device."},
				{"agent-cfg", "Agent config", "Check", "Whether /etc/netductor-agent/config exists."},
			}
		}
	case modeWorkstation:
		if ru {
			modeExtra = []menuEntry{
				{"bootstrap", "Bootstrap one-liner", "Для VPS", "Команда curl|install бинаря netductor на чистый VPS."},
				{"build", "Сборка Go", "Локально", "go build ./cmd/netductor в текущем дереве."},
				{"mt-manage", "MikroTik", "Manage", "Identity/routes/push через SSH (TOFU)."},
			}
		} else {
			modeExtra = []menuEntry{
				{"bootstrap", "Bootstrap one-liner", "For VPS", "curl|install netductor binary on a clean VPS."},
				{"build", "Go build", "Local", "go build ./cmd/netductor in the current tree."},
				{"mt-manage", "MikroTik", "Manage", "Identity/routes/push over SSH (TOFU)."},
			}
		}
	default:
		if ru {
			modeExtra = []menuEntry{
				{"vpn-add", "VPN — добавить", "Пользователь", "Создать VPN user, UUID, применить sing-box, QR."},
				{"vpn-sub", "VPN — ссылка/QR", "Access", "Показать ссылки/QR для пользователя."},
				{"session", "Session", "Временный доступ", "Создать временную session-ссылку."},
				{"sni-live", "Live SNI", "Метрика", "Наблюдение SNI/handshake (если включено)."},
				{"edge-list", "Edge devices", "Агенты", "Список edge/OpenWrt устройств и pending enroll."},
			}
		} else {
			modeExtra = []menuEntry{
				{"vpn-add", "VPN — add user", "Create", "Create VPN user, UUID, apply sing-box, QR."},
				{"vpn-sub", "VPN — link/QR", "Access", "Show links/QR for a user."},
				{"session", "Session", "Temporary", "Create a temporary session link."},
				{"sni-live", "Live SNI", "Metric", "SNI/handshake observation if enabled."},
				{"edge-list", "Edge devices", "Agents", "Edge/OpenWrt devices and pending enroll."},
			}
		}
	}
	var base []menuEntry
	if ru {
		base = []menuEntry{
			{"wizard", "★ Мастер настройки", "Пошаговый setup", "Проводит через primary / secondary / OpenWrt / MikroTik: вопросы (IP, SSH, роль), затем автоматическое применение. Esc — отмена формы."},
			{"doctor", "Doctor", "Проверки здоровья", "Проверяет unit’ы, порты, Blocky, sing-box, API, secondary. Печатает OK/WARN без изменения системы."},
			{"status", "Status", "systemd", "Состояние netductor-api, telegram-bot, sing-box, blocky и связанных unit’ов."},
			{"fleet-status", "Статус флота", "primary/secondary", "Сводка нод primary и secondary: online, роли, IP."},
			{"nodes-list", "Реестр нод", "Список", "Локальный реестр hostname/role/IP (nd-core-…, nd-secondary-…)."},
			{"relay-status", "Secondary", "Агент online", "Статус secondary (legacy relay): heartbeat, mismatch counters, sing-box."},
			{"backup-now", "Бэкап сейчас", "Шифрованный архив", "Пишет .ndenc (etc/netductor, blocky, state, lampac data…) + COMPONENTS. Ротация Keep N."},
			{"fleet-sync", "Синхронизация", "→ secondary", "Репликация данных primary → secondary для тёплых сервисов."},
			{"vpn-list", "VPN пользователи", "Список", "Имя, on/off, UUID, заметка."},
			{"vpn-refresh", "Обновить ссылки", "Prefer secondary", "Пересобрать client links с предпочтением secondary entry."},
			{"probe", "Probes", "Связность", "Локальные TCP/UDP probes (API, VLESS, HY2, Blocky, secondary)."},
			{"ssh-hosts", "SSH known hosts", "TOFU", "Просмотр/сброс доверенных SSH ключей нод."},
			{"audit-tail", "Audit", "События", "Последние события аудита (actor/action/target)."},
			{"change-mode", "Сменить режим", "Mode picker", "Вернуться к выбору VPS / OpenWrt / Workstation / Operator."},
			{"quit", "Выход", "Quit", "Закрыть TUI."},
		}
	} else {
		base = []menuEntry{
			{"wizard", "★ Setup wizard", "Guided setup", "Walks through primary / secondary / OpenWrt / MikroTik: questions (IP, SSH, role), then automatic apply. Esc cancels the form."},
			{"doctor", "Doctor", "Health checks", "Checks units, ports, Blocky, sing-box, API, secondary. Prints OK/WARN without changing the system."},
			{"status", "Status", "systemd", "State of netductor-api, telegram-bot, sing-box, blocky and related units."},
			{"fleet-status", "Fleet status", "primary/secondary", "Node summary: online, roles, IPs."},
			{"nodes-list", "Nodes registry", "List", "Local hostname/role/IP registry (nd-core-…, nd-secondary-…)."},
			{"relay-status", "Secondary", "Agent online", "Secondary status (legacy relay): heartbeat, mismatch counters, sing-box."},
			{"backup-now", "Backup now", "Encrypted archive", "Writes .ndenc (etc/netductor, blocky, state, lampac data…) + COMPONENTS. Rotation Keep N."},
			{"fleet-sync", "Fleet sync", "→ secondary", "Replicate primary data to secondary for VPN entry."},
			{"vpn-list", "VPN users", "List", "Name, on/off, UUID, note."},
			{"vpn-refresh", "Refresh links", "Prefer secondary", "Rebuild client links preferring secondary entry."},
			{"probe", "Probes", "Connectivity", "Local TCP/UDP probes (API, VLESS, HY2, Blocky, secondary)."},
			{"ssh-hosts", "SSH known hosts", "TOFU", "View/reset trusted SSH host keys for nodes."},
			{"audit-tail", "Audit", "Events", "Recent audit events (actor/action/target)."},
			{"change-mode", "Change mode", "Mode picker", "Return to VPS / OpenWrt / Workstation / Operator selection."},
			{"quit", "Quit", "Exit", "Close the TUI."},
		}
	}
	return append(modeExtra, base...)
}
