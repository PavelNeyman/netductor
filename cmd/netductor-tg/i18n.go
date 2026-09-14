package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	tokenFile = "/etc/netductor/secrets/telegram_bot_token"
	chatFile  = "/etc/netductor/secrets/telegram_admin_id"
	langFile  = "/etc/netductor/telegram_lang"
	statusSh  = "/opt/netductor/runtime/telegram/status.sh"
)

type update struct {
	UpdateID      int            `json:"update_id"`
	Message       *message       `json:"message"`
	CallbackQuery *callbackQuery `json:"callback_query"`
}

type message struct {
	MessageID    int      `json:"message_id"`
	Text         string   `json:"text"`
	Chat         chat     `json:"chat"`
	From         *user    `json:"from"`
	ForwardFrom  *user    `json:"forward_from"`
	ForwardOrigin *struct {
		SenderUser *user `json:"sender_user"`
	} `json:"forward_origin"`
}

type callbackQuery struct {
	ID      string   `json:"id"`
	From    user     `json:"from"`
	Message *message `json:"message"`
	Data    string   `json:"data"`
}

type chat struct {
	ID int64 `json:"id"`
}

type user struct {
	ID int64 `json:"id"`
}

// --- i18n ---

var dict = map[string]map[string]string{
	"en": {
		"menu_title":    "🛡 <b>Netductor operator panel</b>",
		"menu_hint":     "Use buttons below. Slash commands still work.",
		"status":        "📊 Status",
		"ready":         "📄 READY",
		"vpn_list":      "👥 VPN list",
		"vpn_add":       "➕ Add user",
		"vpn_link":      "🔗 Link / QR",
		"vpn_rename_hint": "Send: <code>old new</code>\nUUID stays the same.",
		"vpn_sub":       "📦 Subscription",
		"vpn_rename":    "✏️ Rename",
		"vpn_disable":   "🚫 Disable",
		"vpn_enable":    "✅ Enable",
		"vpn_revoke":    "🗑 Revoke",
		"session":       "🔑 API session",
		"admin":         "🖥 Admin UI",
		"help":          "❓ Help",
		"routers":       "📡 Routers",
		"routers_title": "📡 <b>Routers</b>",
		"routers_empty": "No edge agents yet.",
		"lang":          "🌐 Language",
		"main_menu":     "🏠 Main menu",
		"status_title":  "📊 <b>Status</b>",
		"help_title":    "❓ <b>Help</b>",
		"help_body":     "Operator-only bot.\n• VPN users: add / link / enable / disable / revoke\n• Session token for Admin UI &amp; Shortcuts\n• Status &amp; READY\n\nCommands: /menu /status /vpn_list /lang /session /admin /help",
		"add_prompt":    "➕ <b>Add VPN user</b>\n\nSend a short <b>name</b> (e.g. <code>alice</code>).",
		"name_prompt":   "✏️ <b>%s</b>\n\nSend the VPN <b>user name</b>:",
		"session_prompt": "🔑 <b>API session</b>\n\nSend lifetime in <b>hours</b> (e.g. <code>72</code>), or /cancel:",
		"admin_body":    "🖥 <b>Admin UI</b>\n\nURL: <code>http://127.0.0.1:8787/admin/</code>\n\n1) SSH tunnel:\n<code>ssh -L 8787:127.0.0.1:8787 root@VPS</code>\n2) Open URL in browser\n3) Paste session token\n\nOr: <code>netductor vpn api-bind detect</code>",
		"unknown":       "Unknown action.",
		"unknown_cmd":   "Unknown. Open the menu:",
		"no_ready":      "⚠️ No READY.txt",
		"ready_title":   "📄 <b>READY</b>",
		"session_token": "🔑 Session token",
		"lang_now":      "🌐 Language: <b>%s</b>\n\nChoose:",
		"lang_en":       "English",
		"lang_ru":       "Русский",
		"lang_set":      "✅ Language set to <b>%s</b>",
		"copy_token":    "📋 Copy token",
		"show_links":    "🔗 Show links again",
		"cancelled":     "Cancelled.",
		"vpn_users":     "👥 <b>VPN users</b>",
		"label_link":    "Link / QR",
		"label_disable": "Disable",
		"label_enable":  "Enable",
		"label_revoke":  "Revoke",
		"sites": "🏠 Sites",
		"sites_list": "📋 List sites",
		"sites_rsc": "📜 MT RSC",
		"sites_title": "🏠 <b>Sites</b> (MikroTik + RPi)",
		"sites_empty": "No sites yet — create in Admin",
		"nodes":         "🗂 Nodes",
		"nodes_title":   "🗂 <b>Nodes</b>",
		"nodes_hint":    "Format: <code>nd-&lt;role&gt;-&lt;marker&gt;</code>\nExamples: <code>nd-core-nl01</code>, <code>nd-edge-de02</code>, <code>nd-lab-01</code>\nRoles: core, relay, edge, lab. Only a-z, 0-9, hyphen.",
		"nodes_empty":   "No nodes in registry.",
		"nodes_pick":    "Node: <b>%s</b>\nSend new hostname (e.g. nd-relay-msk01):",
		"nodes_done":    "✅ desired set: <code>%s</code> → <code>%s</code>",
		"relay": "📡 Relay",
		"relay_title": "📡 <b>RU Relay</b>",
		"relay_export": "📦 Export join",
		"relay_oneline": "🧾 One-liner",
		"relay_list": "📋 Relays",
		"nodes_rename":  "✏️ <b>Rename</b>\nSelect a node button, then send the new name.\nFormat: <code>nd-&lt;role&gt;-&lt;marker&gt;</code>",
		"cat_vpn":       "🔐 VPN",
		"cat_routers":   "📡 Routers",
		"cat_vpn_title": "🔐 <b>VPN</b>\nManage users, links, access.",
		"cat_routers_title": "📡 <b>Routers</b>\nEdge devices, templates, enroll.",
		"pending":       "⏳ Pending",
		"templates":     "📋 Templates",
		"bind_tmpl":     "🔗 Bind template",
		"apply_tmpl":    "⚙️ Apply template",
		"devices":       "📡 Devices",
		"ssh_hosts":     "🔐 SSH hosts",
		"backup":        "💾 Backup",
		"backup_hint":   "Peer SCP + run backup now",
		"backup_run":    "▶ Run now",
		"backup_set":    "Set peer…",
		"ssh_hosts_hint": "TOFU keys. Forget after MT/VPS reinstall, then reconnect once from trusted LAN.",
		"ssh_forget":    "🗑 Forget…",
		"ssh_clear_mt":  "Clear MikroTik",
		"ssh_clear_rel": "Clear Relay",
		"ssh_cleared":   "Cleared",
		"ssh_forgot":    "Forgot: %s",
		"ssh_empty":     "(empty)",
		"nodes_list_btn": "📋 List",
		"nodes_rename_btn": "✏️ Rename",
		"nodes_enroll": "➕ Enroll relay",
		"nodes_sync": "🔄 Sync relays",
		"nodes_exit": "🇷🇺 RU exit",
		"node_journal": "📋 Journal",
		"node_restart_sb": "♻️ Restart sing-box",
		"node_metrics": "📊 Metrics",
		"node_upgrade": "🔄 Upgrade",
		"node_reboot": "♻️ Reboot",
		"relay_exit_on": "🇷🇺 RU exit ON",
		"relay_exit_off": "RU exit OFF",
		"relay_sync": "🔄 Sync",
		"upgrade_queued": "🔄 <b>Upgrade queued</b>",
		"cmd_wait_hint": "Telegram notify when done · or Metrics",
		"reboot_queued": "♻️ <b>Reboot queued</b>",
		"enroll_title": "📡 <b>Enroll relay</b>",
		"enroll_ip": "Send <b>IP</b> (or host) of the new RU VPS:",
		"enroll_user": "SSH login (usually <code>root</code>):",
		"enroll_pass": "Root password (once; then only core SSH key):",
		"enroll_wait": "⏳ Provisioning <code>%s</code>… (1–3 min)",
		"ru_exit_help": "🇷🇺 <b>RU exit</b>\nWhen ON, traffic from core exits via RU IP (RU services from abroad).",
		"back_vpn":      "⬅️ VPN",
		"back_routers":  "⬅️ Routers",
		"pending_title": "⏳ <b>Pending</b>",
		"pending_empty": "No pending devices.",
		"templates_title": "📋 <b>Templates</b>",
		"bind_prompt":   "Device id to bind template:",
		"apply_prompt":  "Device id to enqueue <code>apply_template</code>:",
		"menu_hint2":    "Choose a category — VPN or Routers open a submenu.",
	},
	"ru": {
		"menu_title":    "🛡 <b>Панель оператора Netductor</b>",
		"menu_hint":     "Кнопки ниже. Слеш-команды тоже работают.",
		"status":        "📊 Статус",
		"ready":         "📄 Готово",
		"vpn_list":      "👥 Список VPN",
		"vpn_add":       "➕ Добавить",
		"vpn_link":      "🔗 Ссылка / QR",
		"vpn_rename_hint": "Пришлите: <code>старое новое</code>\nUUID не меняется.",
		"vpn_sub":       "📦 Subscription",
		"vpn_rename":    "✏️ Переименовать",
		"vpn_disable":   "🚫 Отключить",
		"vpn_enable":    "✅ Включить",
		"vpn_revoke":    "🗑 Отозвать",
		"session":       "🔑 API session",
		"admin":         "🖥 Админка",
		"help":          "❓ Справка",
		"routers":       "📡 Роутеры",
		"routers_title": "📡 <b>Роутеры</b>",
		"routers_empty": "Агентов пока нет.",
		"lang":          "🌐 Язык",
		"main_menu":     "🏠 Меню",
		"status_title":  "📊 <b>Статус</b>",
		"help_title":    "❓ <b>Справка</b>",
		"help_body":     "Бот только для оператора.\n• VPN: добавить / ссылка / вкл / выкл / отозвать\n• Session-токен для админки и Shortcuts\n• Статус и READY\n\nКоманды: /menu /status /vpn_list /lang /session /admin /help",
		"add_prompt":    "➕ <b>Новый VPN-пользователь</b>\n\nПришлите короткое <b>имя</b> (напр. <code>alice</code>).",
		"name_prompt":   "✏️ <b>%s</b>\n\nПришлите <b>имя пользователя VPN</b>:",
		"session_prompt": "🔑 <b>API session</b>\n\nПришлите срок в <b>часах</b> (напр. <code>72</code>) или /cancel:",
		"admin_body":    "🖥 <b>Админка</b>\n\nURL: <code>http://127.0.0.1:8787/admin/</code>\n\n1) SSH-туннель:\n<code>ssh -L 8787:127.0.0.1:8787 root@VPS</code>\n2) Откройте URL в браузере\n3) Вставьте session-токен\n\nИли: <code>netductor vpn api-bind detect</code>",
		"unknown":       "Неизвестное действие.",
		"unknown_cmd":   "Неизвестно. Откройте меню:",
		"no_ready":      "⚠️ Нет READY.txt",
		"ready_title":   "📄 <b>Готово</b>",
		"session_token": "🔑 Session-токен",
		"lang_now":      "🌐 Язык: <b>%s</b>\n\nВыберите:",
		"lang_en":       "English",
		"lang_ru":       "Русский",
		"lang_set":      "✅ Язык: <b>%s</b>",
		"copy_token":    "📋 Копировать токен",
		"show_links":    "🔗 Ссылки ещё раз",
		"cancelled":     "Отменено.",
		"vpn_users":     "👥 <b>Пользователи VPN</b>",
		"label_link":    "Ссылка / QR",
		"label_disable": "Отключить",
		"label_enable":  "Включить",
		"label_revoke":  "Отозвать",
		"sites":         "🏠 Площадки",
		"sites_list":    "📋 Список",
		"sites_rsc":     "📜 RSC MikroTik",
		"sites_title":   "🏠 <b>Площадки</b> (MikroTik + RPi)",
		"sites_empty":   "Пока пусто — создайте в Admin",
		"nodes":         "🗂 Ноды",
		"nodes_title":   "🗂 <b>Ноды</b>",
		"nodes_hint":    "Формат: <code>nd-&lt;role&gt;-&lt;marker&gt;</code>\nПримеры: <code>nd-core-nl01</code>, <code>nd-edge-de02</code>, <code>nd-lab-01</code>\nРоли: core, relay, edge, lab. Только a-z, 0-9, дефис.",
		"nodes_empty":   "В реестре нет нод.",
		"nodes_pick":    "Нода: <b>%s</b>\nПришлите новое имя (например nd-relay-msk01):",
		"nodes_done":    "✅ desired: <code>%s</code> → <code>%s</code>",
		"relay": "📡 Relay",
		"relay_title": "📡 <b>RU Relay</b>",
		"relay_export": "📦 Export join",
		"relay_oneline": "🧾 One-liner",
		"relay_list": "📋 Relays",
		"nodes_rename":  "✏️ <b>Переименовать</b>\n\nФормат: <code>nd-&lt;role&gt;-&lt;marker&gt;</code>\nПришлите: <code>id новое-имя</code>\nПример: <code>nd-core-11870 nd-core-nl01</code>",
		"cat_vpn":       "🔐 VPN",
		"cat_routers":   "📡 Роутеры",
		"cat_vpn_title": "🔐 <b>VPN</b>\nПользователи, ссылки, доступ.",
		"cat_routers_title": "📡 <b>Роутеры</b>\nУстройства, шаблоны, enroll.",
		"pending":       "⏳ Ожидают",
		"templates":     "📋 Шаблоны",
		"bind_tmpl":     "🔗 Привязать шаблон",
		"apply_tmpl":    "⚙️ Применить шаблон",
		"devices":       "📡 Устройства",
		"ssh_hosts":     "🔐 SSH hosts",
		"backup":        "💾 Бэкап",
		"backup_hint":   "Peer SCP + запуск бэкапа",
		"backup_run":    "▶ Сейчас",
		"backup_set":    "Задать peer…",
		"ssh_hosts_hint": "TOFU-ключи. После переустановки MT/VPS — Forget, затем один раз из LAN.",
		"ssh_forget":    "🗑 Забыть…",
		"ssh_clear_mt":  "Очистить MikroTik",
		"ssh_clear_rel": "Очистить Relay",
		"ssh_cleared":   "Очищено",
		"ssh_forgot":    "Забыто: %s",
		"ssh_empty":     "(пусто)",
		"nodes_list_btn": "📋 Список",
		"nodes_rename_btn": "✏️ Переименовать",
		"nodes_enroll": "➕ Добавить relay",
		"nodes_sync": "🔄 Sync relay",
		"nodes_exit": "🇷🇺 RU exit",
		"node_journal": "📋 Journal",
		"node_restart_sb": "♻️ Restart sing-box",
		"node_metrics": "📊 Метрики",
		"node_upgrade": "🔄 Обновить",
		"node_reboot": "♻️ Перезагрузка",
		"relay_exit_on": "🇷🇺 RU exit ВКЛ",
		"relay_exit_off": "RU exit ВЫКЛ",
		"relay_sync": "🔄 Sync",
		"upgrade_queued": "🔄 <b>Обновление в очереди</b>",
		"cmd_wait_hint": "уведомление в Telegram по завершении · или Метрики",
		"reboot_queued": "♻️ <b>Перезагрузка в очереди</b>",
		"enroll_title": "📡 <b>Добавить relay</b>",
		"enroll_ip": "Пришлите <b>IP</b> (или host) новой RU VPS:",
		"enroll_user": "Логин SSH (обычно <code>root</code>):",
		"enroll_pass": "Пароль root (один раз; дальше только SSH-ключ core):",
		"enroll_wait": "⏳ Настройка <code>%s</code>… (1–3 мин)",
		"ru_exit_help": "🇷🇺 <b>RU exit</b>\nЕсли ВКЛ — трафик с core выходит через IP РФ (доступ к RU-сервисам из-за границы).",
		"back_vpn":      "⬅️ VPN",
		"back_routers":  "⬅️ Роутеры",
		"pending_title": "⏳ <b>Ожидают</b>",
		"pending_empty": "Нет устройств в ожидании.",
		"templates_title": "📋 <b>Шаблоны</b>",
		"bind_prompt":   "ID устройства для привязки шаблона:",
		"apply_prompt":  "ID устройства для <code>apply_template</code>:",
		"menu_hint2":    "Выберите раздел — VPN или Роутеры откроют подменю.",
	},
}

func getLang() string {
	b, err := os.ReadFile(langFile)
	if err != nil {
		return "ru"
	}
	s := strings.TrimSpace(string(b))
	if s == "en" || s == "ru" {
		return s
	}
	return "ru"
}

func setLang(l string) {
	if l != "en" && l != "ru" {
		return
	}
	_ = os.WriteFile(langFile, []byte(l+"\n"), 0644)
}

func T(key string) string {
	l := getLang()
	if m, ok := dict[l]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := dict["en"][key]; ok {
		return v
	}
	return key
}

func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

