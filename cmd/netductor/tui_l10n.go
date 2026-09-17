package main

import (
	"os"
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
	TabWizard, TabTools, TabOps, TabMode string
	HelpWizard, HelpTools, HelpOps, HelpModeBar, HelpOutput []helpChip
}

func l10n(lang tuiLang) locPack { return locFor(lang) }

func detectLang() tuiLang {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(os.Getenv(k))
		if strings.HasPrefix(v, "ru") {
			return langRU
		}
	}
	return langEN
}

func locFor(lang tuiLang) locPack {
	if lang == langRU {
		return locPack{
			App: "netductor",
			TabWizard: "Мастер", TabTools: "Инструменты", TabOps: "Операции", TabMode: "Режим",
			HelpWizard: []helpChip{{"tab", "вкладки"}, {"↑↓", "выбор"}, {"enter", "далее"}, {"esc", "назад"}, {"l", "язык"}, {"^c", "выход"}},
			HelpTools:  []helpChip{{"tab", "вкладки"}, {"↑↓", "пункт"}, {"enter", "запуск"}, {"esc", "назад"}, {"l", "язык"}, {"^c", "выход"}},
			HelpOps:    []helpChip{{"tab", "вкладки"}, {"↑↓", "пункт"}, {"enter", "запуск"}, {"esc", "назад"}, {"l", "язык"}, {"^c", "выход"}},
			HelpModeBar: []helpChip{{"↑↓", "режим"}, {"enter", "выбрать"}, {"l", "язык"}, {"^c", "выход"}},
			HelpOutput: []helpChip{{"esc", "назад"}, {"enter", "назад"}, {"l", "язык"}, {"^c", "выход"}},
		}
	}
	return locPack{
		App: "netductor",
		TabWizard: "Wizard", TabTools: "Tools", TabOps: "Ops", TabMode: "Mode",
		HelpWizard: []helpChip{{"tab", "tabs"}, {"↑↓", "select"}, {"enter", "next"}, {"esc", "back"}, {"l", "lang"}, {"^c", "quit"}},
		HelpTools:  []helpChip{{"tab", "tabs"}, {"↑↓", "item"}, {"enter", "run"}, {"esc", "back"}, {"l", "lang"}, {"^c", "quit"}},
		HelpOps:    []helpChip{{"tab", "tabs"}, {"↑↓", "item"}, {"enter", "run"}, {"esc", "back"}, {"l", "lang"}, {"^c", "quit"}},
		HelpModeBar: []helpChip{{"↑↓", "mode"}, {"enter", "select"}, {"l", "lang"}, {"^c", "quit"}},
		HelpOutput: []helpChip{{"esc", "back"}, {"enter", "back"}, {"l", "lang"}, {"^c", "quit"}},
	}
}

type menuEntry struct {
	ID, Title, Short, Detail string
}

func modeEntries(lang tuiLang) []menuEntry {
	if lang == langRU {
		return []menuEntry{
			{"vps", "VPS (эта машина)", "Install на сервере", "Запуск на самой VPS: install стека, VPN, Blocky, API, bot. Не для Mac."},
			{"workstation", "Workstation", "Mac/PC → remote", "С ноутбука: подключение по SSH к primary/secondary и управление без выхода из TUI."},
			{"openwrt", "OpenWrt / edge", "Роутер", "Provision агента на OpenWrt/RPi по LAN/SSH."},
			{"operator", "Оператор", "День-2", "Локальные операции на уже настроенной ноде (или remote, если задан SSH)."},
		}
	}
	return []menuEntry{
		{"vps", "VPS (this host)", "Server install", "Run on the VPS itself: stack install, VPN, Blocky, API, bot. Not for Mac."},
		{"workstation", "Workstation", "Mac/PC → remote", "From laptop: SSH to primary/secondary and operate without leaving the TUI."},
		{"openwrt", "OpenWrt / edge", "Router", "Provision edge agent on OpenWrt/RPi over LAN/SSH."},
		{"operator", "Operator", "Day-2", "Day-2 ops on a configured node (or via remote SSH target)."},
	}
}

func toolsEntries(mode runMode, lang tuiLang) []menuEntry {
	ru := lang == langRU
	var e []menuEntry

	// Connection (workstation first)
	if mode == modeWorkstation {
		if ru {
			e = append(e,
				menuEntry{"remote-set", "Подключить VPS", "SSH target", "Задать user@host (ключ в ssh-agent). Все команды Ops/Tools ниже пойдут на эту ноду."},
				menuEntry{"remote-clear", "Сбросить remote", "Local only", "Дальше команды только локально на этом Mac."},
			)
		} else {
			e = append(e,
				menuEntry{"remote-set", "Connect VPS", "SSH target", "Set user@host (key in ssh-agent). Tools/Ops commands run on that node."},
				menuEntry{"remote-clear", "Clear remote", "Local only", "Commands run only on this Mac again."},
			)
		}
	}

	// Install only on VPS mode
	if mode == modeVPS {
		if ru {
			e = append(e,
				menuEntry{"install", "Install стека", "На этой VPS", "Полный install primary (sing-box, blocky, api…). Только когда TUI запущен на сервере."},
				menuEntry{"prepare", "Подготовка", "apt/docker/go", "Подготовка системы перед install."},
				menuEntry{"apply-lampac", "Lampac", "Primary only", "Docker Lampac на этой машине."},
			)
		} else {
			e = append(e,
				menuEntry{"install", "Install stack", "On this VPS", "Full primary install. Only when TUI runs on the server."},
				menuEntry{"prepare", "Prepare", "apt/docker/go", "System prep before install."},
				menuEntry{"apply-lampac", "Lampac", "Primary only", "Docker Lampac on this host."},
			)
		}
	}

	if mode == modeOpenWRT {
		if ru {
			e = append(e,
				menuEntry{"owrt-install", "Edge provision", "OpenWrt SSH", "Установка агента на роутер (нужен доступ по SSH с этой машины)."},
			)
		} else {
			e = append(e,
				menuEntry{"owrt-install", "Edge provision", "OpenWrt SSH", "Install agent on router (SSH from this machine)."},
			)
		}
	}

	if mode == modeWorkstation {
		if ru {
			e = append(e,
				menuEntry{"build", "Сборка бинарей", "Локально", "go build под linux/darwin (на Mac)."},
				menuEntry{"owrt-install", "Edge provision", "LAN/SSH", "Агент на OpenWrt с этой машины."},
			)
		} else {
			e = append(e,
				menuEntry{"build", "Build binaries", "Local", "go build for linux/darwin on this Mac."},
				menuEntry{"owrt-install", "Edge provision", "LAN/SSH", "OpenWrt agent from this machine."},
			)
		}
	}

	// Common navigation
	if ru {
		e = append(e,
			menuEntry{"change-mode", "Сменить режим", "Mode", "VPS / Workstation / OpenWrt / Operator."},
			menuEntry{"quit", "Выход", "Quit", "Закрыть TUI."},
		)
	} else {
		e = append(e,
			menuEntry{"change-mode", "Change mode", "Mode", "VPS / Workstation / OpenWrt / Operator."},
			menuEntry{"quit", "Quit", "Exit", "Close the TUI."},
		)
	}
	return e
}

func opsEntries(mode runMode, lang tuiLang) []menuEntry {
	ru := lang == langRU
	// Day-2 ops: work locally or via remote SSH target
	if ru {
		return []menuEntry{
			{"status", "Статус сервисов", "systemd", "sing-box, api, bot, blocky (local или remote)."},
			{"doctor", "Doctor", "Проверки", "Health checks без изменений."},
			{"fleet-status", "Флот", "primary/secondary", "Роли нод, online, policy."},
			{"vpn-list", "VPN пользователи", "Список", "Имя, UUID, on/off."},
			{"relay-sync", "VPN → secondary", "config_ver", "Протолкнуть UUID на RU entry."},
			{"relay-status", "Secondary status", "Agent", "Heartbeat / sing-box на secondary."},
			{"nodes-list", "Реестр нод", "nodes", "Hostname, role, IP."},
			{"edge-list", "Edge devices", "OpenWrt", "Список edge / pending approve."},
			{"probe", "Probes", "Связность", "Локальные probes API/VPN."},
			{"backup-now", "Бэкап", "Сейчас", "Шифрованный backup на ноде."},
			{"audit-tail", "Audit", "События", "Последние audit events."},
			{"disable-legacy", "Disable legacy", "Timers", "Выключить старые fleet-sync/bot-failover."},
			{"change-mode", "Сменить режим", "Mode", "Вернуться к выбору режима."},
			{"quit", "Выход", "Quit", "Закрыть TUI."},
		}
	}
	return []menuEntry{
		{"status", "Service status", "systemd", "sing-box, api, bot, blocky (local or remote)."},
		{"doctor", "Doctor", "Health", "Health checks, no changes."},
		{"fleet-status", "Fleet", "primary/secondary", "Node roles, online, policy."},
		{"vpn-list", "VPN users", "List", "Name, UUID, on/off."},
		{"relay-sync", "VPN → secondary", "config_ver", "Push user UUIDs to RU entry."},
		{"relay-status", "Secondary status", "Agent", "Heartbeat / sing-box on secondary."},
		{"nodes-list", "Nodes registry", "nodes", "Hostname, role, IP."},
		{"edge-list", "Edge devices", "OpenWrt", "Edge list / pending approve."},
		{"probe", "Probes", "Connectivity", "API/VPN probes."},
		{"backup-now", "Backup now", "Run", "Encrypted backup on the node."},
		{"audit-tail", "Audit", "Events", "Recent audit events."},
		{"disable-legacy", "Disable legacy", "Timers", "Stop old fleet-sync/bot-failover units."},
		{"change-mode", "Change mode", "Mode", "Back to mode picker."},
		{"quit", "Quit", "Exit", "Close the TUI."},
	}
}
