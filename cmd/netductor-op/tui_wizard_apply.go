package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
	"github.com/PavelNeyman/netductor/internal/operator"
)

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func yesish(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "y" || s == "yes" || s == "1" || s == "true" || s == "да"
}

func wizBuildFields(id string, m *model) []wizField {
	s := loadTUISettings()
	lang := detectLang()
	ru := lang == langRU
	ph := func(en, r string) string {
		if ru {
			return r
		}
		return en
	}
	switch id {
	case "primary":
		return []wizField{
			{Key: "host", Label: FormT(lang, "primary_host"), Value: s.RemoteHost, Placeholder: "x.x.x.x",
				Short:  ph("VPS public IP or DNS", "Публичный IP или DNS VPS"),
				Detail: ph("Abroad control-plane host reachable over SSH on port 22 (first login).", "Зарубежный control plane: SSH на порт 22 при первом входе.")},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: "root",
				Short:  ph("SSH user", "Пользователь SSH"),
				Detail: ph("Usually root on a fresh Debian VPS.", "Обычно root на чистом Debian VPS.")},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true,
				Short:  ph("Only for first bootstrap", "Только первый вход"),
				Detail: ph("Used once with sshpass; then password auth is disabled.", "Один раз через sshpass; дальше вход только по ключу.")},
			{Key: "gen_key", Label: FormT(lang, "gen_ssh_key") + " (yes/no)", Value: "yes",
				Short:  ph("yes = create ed25519 key", "yes = создать ключ ed25519"),
				Detail: ph("Creates ~/.ssh/netductor_primary unless path overridden.", "Создаёт ~/.ssh/netductor_primary, если путь не задан иначе.")},
			{Key: "key_path", Label: FormT(lang, "ssh_key_path"), Value: orDefault(s.RemoteKey, "~/.ssh/netductor_primary"),
				Short:  ph("Private key path on Mac", "Путь к private key на Mac"),
				Detail: ph("Saved into TUI settings as remote_key.", "Сохраняется в настройках TUI как remote_key.")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (empty=none)", Secret: true,
				Short:  ph("Optional key encryption", "Опциональная фраза ключа"),
				Detail: ph("Leave empty for no passphrase. Needed later for secondary/edge if set.", "Пусто = без фразы. Если задали — понадобится для secondary/edge.")},
			{Key: "sni", Label: FormT(lang, "reality_sni"), Value: "api.vk.me",
				Short:  ph("Reality TLS SNI", "SNI для Reality"),
				Detail: ph("SNI camouflage for VPN (e.g. api.vk.me). DEFAULT_SNI in conf.", "Маскировка VPN. В conf: DEFAULT_SNI.")},
			{Key: "ssh_port", Label: FormT(lang, "ssh_port"), Value: "52222",
				Short:  ph("SSH after harden", "SSH после harden"),
				Detail: ph("Default 52222. Written to netductor.conf SSH_PORT.", "По умолчанию 52222. В conf: SSH_PORT.")},
			{Key: "redirect_https_port", Label: FormT(lang, "redirect_https_port"), Value: "8443",
				Short:  ph("LE redirect HTTPS port", "Порт HTTPS redirect (LE)"),
				Detail: ph("Default 8443 (443 is Reality). REDIRECT_HTTPS_PORT.", "По умолчанию 8443 (443 = Reality).")},
			{Key: "agent_mtls_port", Label: FormT(lang, "agent_mtls_port"), Value: "8789",
				Short:  ph("Agent mTLS plane", "Плоскость agent mTLS"),
				Detail: ph("Default 8789. AGENT_MTLS_PORT.", "По умолчанию 8789.")},
			{Key: "lampac_port", Label: FormT(lang, "lampac_port"), Value: "9118",
				Short:  ph("Lampac listen (loopback)", "Lampac (localhost)"),
				Detail: ph("Only if Lampac installed. LAMPAC_PORT.", "Если ставите Lampac. LAMPAC_PORT.")},
			{Key: "domain_primary", Label: FormT(lang, "domain_primary"), Value: "",
				Short:  ph("DNS A → primary IP", "DNS A → IP primary"),
				Detail: ph("e.g. p2.nd.example.com. Required for LE. No auto p./i. invent.", "Напр. p2.nd.example.com. Для LE обязательно. Без авто-p./i.")},
			{Key: "domain_vpn", Label: FormT(lang, "domain_vpn"), Value: "",
				Short:  ph("VPN entry host", "Хост входа VPN"),
				Detail: ph("e.g. s.nd.example.com (secondary public name).", "Напр. s.nd.example.com (имя secondary).")},
			{Key: "domain_redirect", Label: FormT(lang, "domain_redirect"), Value: "",
				Short:  ph("REDIRECT_BASE URL", "URL REDIRECT_BASE"),
				Detail: ph("e.g. https://i2.nd.example.com:8443", "Напр. https://i2.nd.example.com:8443")},
			{Key: "domain_base", Label: FormT(lang, "domain_base"), Value: "",
				Short:  ph("Optional DOMAIN= label only", "Опционально только DOMAIN="),
				Detail: ph("Org label in conf only; does not invent hostnames.", "Метка в conf; хосты не выдумывает.")},
			{Key: "le_email", Label: FormT(lang, "le_email"), Value: "",
				Short:  ph("Let's Encrypt email", "Email Let's Encrypt"),
				Detail: ph("Requires domain_primary (+ redirect). Empty = skip LE.", "Нужен domain_primary. Пусто = без LE.")},
			{Key: "cf_proxy", Label: FormT(lang, "cf_proxy"), Value: "no", Toggle: true,
				Short: ph("Cloudflare orange on i.", "CF orange на i."),
				Detail: ph("Usually no. CF free SSL only covers one subdomain level (*.neyman.top), not i.netductor.neyman.top. Keep DNS-only (grey) and :8443.",
					"Обычно no. Бесплатный SSL CF не покрывает i.netductor.… (два уровня). Серое облако + :8443.")},
			{Key: "with_lampac", Label: FormT(lang, "with_lampac_primary"), Value: "no", Toggle: true,
				Short:  ph("Install Lampac add-on", "Поставить Lampac"),
				Detail: ph("Docker Lampac on primary after core install.", "Docker Lampac на primary после ядра.")},
			{Key: "with_git", Label: FormT(lang, "with_git_registry"), Value: "no", Toggle: true,
				Short:  ph("Thin git + registry", "Git + registry"),
				Detail: ph("Optional git repos + local container registry.", "Опционально git и локальный registry.")},
			{Key: "tg_token", Label: FormT(lang, "tg_bot_token"), Value: "", Secret: true,
				Short:  ph("Optional Telegram bot", "Опционально TG-бот"),
				Detail: ph("If set, installs telegram unit on primary.", "Если задан — ставит telegram unit.")},
			{Key: "tg_admin", Label: FormT(lang, "tg_admin_id"), Value: "",
				Short:  ph("Telegram admin user id", "TG admin id"),
				Detail: ph("Numeric Telegram user id for operator.", "Числовой id оператора в Telegram.")},
		}

	case "fleet":
		return []wizField{
			{Key: "do_primary", Label: FormT(lang, "fleet_do_primary"), Value: "yes", Toggle: true,
				Short:  ph("Deploy abroad control plane", "Деплой зарубежного primary"),
				Detail: ph("Runs full primary install + domain/LE + selected add-ons.", "Полный primary: install, domain/LE, add-ons.")},
			{Key: "do_secondary", Label: FormT(lang, "fleet_do_secondary"), Value: "yes", Toggle: true,
				Short:  ph("Deploy RU secondary", "Деплой RU secondary"),
				Detail: ph("After primary: Mac-direct secondary provision.", "После primary: secondary с Mac.")},
			{Key: "host", Label: FormT(lang, "primary_host"), Value: s.RemoteHost, Placeholder: "x.x.x.x",
				Short: ph("Primary public IP", "IP primary"), Detail: ph("Abroad VPS.", "Зарубежный VPS.")},
			{Key: "password", Label: FormT(lang, "ssh_password_first") + " (primary)", Secret: true,
				Short: ph("Primary first-login password", "Пароль primary"), Detail: ph("Once; then key-only.", "Один раз.")},
			{Key: "sec_host", Label: FormT(lang, "secondary_host"), Value: "",
				Short: ph("Secondary public IP", "IP secondary"), Detail: ph("RU VPS.", "РФ VPS.")},
			{Key: "sec_password", Label: FormT(lang, "ssh_password_first") + " (secondary)", Secret: true,
				Short: ph("Secondary first-login password", "Пароль secondary"), Detail: ph("Once; then key-only.", "Один раз.")},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: "root", Short: "SSH user", Detail: "root"},
			{Key: "gen_key", Label: FormT(lang, "gen_ssh_key") + " (yes/no)", Value: "yes",
				Short: ph("Create Mac key", "Ключ на Mac"), Detail: ph("~/.ssh/netductor_primary", "~/.ssh/netductor_primary")},
			{Key: "key_path", Label: FormT(lang, "ssh_key_path"), Value: orDefault(s.RemoteKey, "~/.ssh/netductor_primary"),
				Short: ph("Private key path", "Путь к ключу"), Detail: ph("Shared for primary+secondary collect.", "Общий для collect.")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (empty=none)", Secret: true,
				Short: ph("Optional", "Опционально"), Detail: ph("Key passphrase", "Фраза ключа")},
			{Key: "sni", Label: FormT(lang, "reality_sni"), Value: "api.vk.me",
				Short: "Reality SNI", Detail: ph("Same on both nodes. DEFAULT_SNI.", "На обеих нодах.")},
			{Key: "ssh_port", Label: FormT(lang, "ssh_port"), Value: "52222",
				Short: ph("SSH after harden", "SSH после harden"), Detail: "SSH_PORT"},
			{Key: "redirect_https_port", Label: FormT(lang, "redirect_https_port"), Value: "8443",
				Short: "REDIRECT_HTTPS_PORT", Detail: ph("LE import port", "Порт LE redirect")},
			{Key: "agent_mtls_port", Label: FormT(lang, "agent_mtls_port"), Value: "8789",
				Short: "AGENT_MTLS_PORT", Detail: ph("Agent plane", "Плоскость агента")},
			{Key: "lampac_port", Label: FormT(lang, "lampac_port"), Value: "9118",
				Short: "LAMPAC_PORT", Detail: "Lampac"},
			{Key: "domain_primary", Label: FormT(lang, "domain_primary"), Value: "",
				Short: ph("CORE FQDN explicit", "CORE FQDN явно"), Detail: ph("e.g. p2.nd.example.com — required for LE", "Напр. p2.nd… — для LE")},
			{Key: "domain_vpn", Label: FormT(lang, "domain_vpn"), Value: "",
				Short: ph("VPN entry FQDN", "VPN entry FQDN"), Detail: ph("e.g. s.nd.example.com", "Напр. s.nd…")},
			{Key: "domain_redirect", Label: FormT(lang, "domain_redirect"), Value: "",
				Short: "REDIRECT_BASE URL", Detail: ph("e.g. https://i2.nd.example.com:8443", "https://i2…:8443")},
			{Key: "domain_base", Label: FormT(lang, "domain_base"), Value: "",
				Short: ph("Optional DOMAIN= label", "Опционально DOMAIN="), Detail: ph("Does not invent hostnames", "Хосты не выдумывает")},
			{Key: "le_email", Label: FormT(lang, "le_email"), Value: "",
				Short: ph("LE email", "LE email"), Detail: ph("Needs domain_primary", "Нужен domain_primary")},
			{Key: "cf_proxy", Label: FormT(lang, "cf_proxy"), Value: "no", Toggle: true,
				Short: "CF orange i.", Detail: ph("Usually no for multi-level names.", "Обычно no.")},
			{Key: "with_lampac", Label: FormT(lang, "with_lampac_primary"), Value: "no", Toggle: true,
				Short: "Lampac", Detail: "Docker on primary"},
			{Key: "with_git", Label: FormT(lang, "with_git_registry"), Value: "no", Toggle: true,
				Short: "Git+registry", Detail: "Optional"},
			{Key: "tg_token", Label: FormT(lang, "tg_bot_token"), Value: "", Secret: true,
				Short: "TG bot", Detail: "Optional"},
			{Key: "tg_admin", Label: FormT(lang, "tg_admin_id"), Value: "",
				Short: "TG admin id", Detail: "Optional"},
		}

	case "secondary":
		return []wizField{
			{Key: "host", Label: FormT(lang, "secondary_host"),
				Short:  ph("RU VPS address", "Адрес РФ VPS"),
				Detail: ph("VPN entry only. Primary must already be set in TUI.", "Только VPN entry. Primary уже должен быть в TUI.")},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: "root", Short: "SSH", Detail: ph("First login user.", "Пользователь первого входа.")},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true,
				Short:  ph("First SSH password", "Пароль первого SSH"),
				Detail: ph("Mac pubkey installed; password auth then off.", "Ставится pubkey Mac; пароль отключается.")},
			{Key: "sni", Label: FormT(lang, "reality_sni"), Value: "api.vk.me", Short: "Reality SNI", Detail: ph("Usually same as primary.", "Обычно как на primary.")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (primary key)", Secret: true,
				Short:  ph("If primary key has passphrase", "Если ключ primary с фразой"),
				Detail: ph("Unlocks Mac private key when SSHing to primary during provision.", "Нужна, чтобы с Mac ходить на primary при provision.")},
		}
	case "openwrt":
		srv := "https://PRIMARY:8789"
		if s.RemoteHost != "" {
			srv = "https://" + s.RemoteHost + ":8789"
		}
		return []wizField{
			{Key: "router", Label: FormT(lang, "router_lan_ip"), Value: "192.168.1.1",
				Short:  ph("Current SSH reachability IP", "IP, куда сейчас ходит SSH"),
				Detail: ph("Must be reachable from this Mac over LAN.", "Должен быть доступен с Mac по LAN.")},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true,
				Short:  ph("Current root; empty=factory", "Текущий root; пусто=завод"),
				Detail: ph("Leave empty for factory OpenWrt with no password.", "Пусто = заводской OpenWrt без пароля.")},
			{Key: "new_root_password", Label: "New root password (LuCI)", Secret: true,
				Short:  ph("required for LuCI", "нужен для LuCI"),
				Detail: ph("Sets system root password for web UI. SSH uses key; password SSH is disabled after provision.", "Пароль root для LuCI. SSH — ключ; password SSH отключается.")},
			{Key: "skip_root_pass", Label: "Skip setting root password? (yes/no)", Value: "no",
				Short:  ph("default no = require password", "по умолчанию no = пароль обязателен"),
				Detail: ph("yes = leave root password empty/unchanged (not recommended for LuCI).", "yes = не задавать пароль (для LuCI не рекомендуется).")},
			{Key: "id", Label: FormT(lang, "edge_device_id"), Value: orDefault(s.LastEdgeID, "edge-1"),
				Short:  ph("Stable edge id", "Стабильный id edge"),
				Detail: ph("Used for enroll/approve on primary.", "Для enroll/approve на primary.")},
			{Key: "arch", Label: FormT(lang, "agent_arch"), Value: "auto", Short: "auto | arm64 | arm | mipsle | amd64 | riscv64", Detail: ph("auto = SSH probe uname -m (Cudy TR1200 → mipsle).", "auto = probe uname -m (Cudy TR1200 → mipsle).")},
			{Key: "server", Label: FormT(lang, "primary_mtls"), Value: srv,
				Short:  "https://PRIMARY:8789",
				Detail: ph("Agent plane from router. Prefills from Settings→primary host. SSH to primary uses Settings key (set after Fleet).", "Agent plane с роутера. Prefill из Настройки→primary. SSH на primary — ключ из Настроек (после Fleet).")},
			{Key: "net", Label: "Configure network? (yes/no)", Value: "no",
				Short:  ph("Apply LAN/Wi-Fi/WAN via UCI now", "Применить LAN/Wi-Fi/WAN через UCI"),
				Detail: ph("no = only install agent (enroll). yes = also write LAN/DHCP/Wi-Fi/WAN on router after agent install.", "no = только агент (enroll). yes = ещё UCI LAN/DHCP/Wi-Fi/WAN после установки агента.")},
			{Key: "lan_ip", Label: "LAN IP", Value: "192.168.50.1", Short: ph("New router address", "Новый адрес роутера"),
				Detail: ph("Prefer different subnets per site.", "На разных точках лучше разные подсети.")},
			{Key: "lan_mask", Label: "LAN netmask", Value: "255.255.255.0", Short: "mask", Detail: ph("LAN subnet mask.", "Маска LAN.")},
			{Key: "dhcp_start", Label: "DHCP start", Value: "100", Short: "pool start", Detail: ph("DHCP range start host.", "Начало DHCP pool.")},
			{Key: "dhcp_limit", Label: "DHCP limit", Value: "150", Short: "pool size", Detail: ph("DHCP lease count.", "Число адресов DHCP.")},
			{Key: "wifi_ssid_24", Label: "Wi-Fi SSID 2.4 GHz", Short: "2.4 GHz", Detail: ph("Empty 5 GHz inherits 2.4.", "Пустой 5 GHz = как 2.4.")},
			{Key: "wifi_key_24", Label: "Wi-Fi password 2.4", Secret: true, Short: "PSK 2.4", Detail: ph("WPA2/3 key 2.4 GHz.", "Ключ 2.4 ГГц.")},
			{Key: "wifi_ssid_5", Label: "Wi-Fi SSID 5 GHz (optional)", Short: "5 GHz", Detail: ph("Leave empty to copy from 2.4.", "Пусто = скопировать с 2.4.")},
			{Key: "wifi_key_5", Label: "Wi-Fi password 5 (optional)", Secret: true, Short: "PSK 5", Detail: ph("Empty = same as 2.4.", "Пусто = как 2.4.")},
			{Key: "wan_proto", Label: "WAN (dhcp|static|pppoe)", Value: "dhcp", Short: "WAN type", Detail: ph("Provider uplink type.", "Тип uplink провайдера.")},
			{Key: "wan_ip", Label: "WAN static IP", Short: "static IP", Detail: ph("Only if wan_proto=static.", "Только для static.")},
			{Key: "wan_mask", Label: "WAN netmask", Short: "mask", Detail: ph("Static WAN mask.", "Маска static WAN.")},
			{Key: "wan_gateway", Label: "WAN gateway", Short: "gw", Detail: ph("Static gateway.", "Шлюз static.")},
			{Key: "wan_dns", Label: "WAN DNS", Short: "DNS", Detail: ph("Comma-separated or one IP; empty=ISP/default.", "DNS; пусто = провайдер.")},
			{Key: "pppoe_user", Label: "PPPoE user", Short: "pppoe", Detail: ph("Only if wan_proto=pppoe.", "Только для pppoe.")},
			{Key: "pppoe_pass", Label: "PPPoE password", Secret: true, Short: "pppoe pass", Detail: ph("PPPoE password.", "Пароль PPPoE.")},
			{Key: "guest", Label: "Guest Wi-Fi? (yes/no)", Value: "no", Short: "Guest SSID", Detail: ph("Separate guest SSID: internet only, no LAN, traffic bypasses VPN (ISP direct). Staff PIN/grant flow.", "Отдельный guest SSID: только интернет, без LAN, мимо VPN (провайдер). PIN/grant для продавца.")},
			{Key: "guest_ssid", Label: "Guest SSID", Value: "Guest", Short: "guest name", Detail: ph("Guest network name.", "Имя гостевой сети.")},
			{Key: "reboot", Label: "Reboot after provision? (yes/no)", Value: "yes",
				Short:  ph("Reboot router", "Перезагрузить роутер"),
				Detail: ph("Agent starts on boot and enrolls to primary.", "Агент стартует после boot и делает enroll.")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (primary key)", Secret: true, Short: "Mac key", Detail: ph("If primary key is encrypted.", "Если ключ primary с фразой.")},
		}
	case "luci":
		return []wizField{
			{Key: "router", Label: "Router LAN IP", Value: "192.168.1.1", Short: "SSH host", Detail: ph("Same LAN as this Mac.", "Та же LAN, что Mac.")},
			{Key: "user", Label: "SSH user", Value: "root", Short: "user", Detail: "root"},
			{Key: "password", Label: "Router password", Secret: true, Short: "empty=key", Detail: ph("Or leave empty if key auth works.", "Пусто если уже ключ.")},
			{Key: "key", Label: "SSH key path", Value: orDefault(s.RemoteKey, "~/.ssh/netductor_primary"), Short: "key", Detail: ph("Operator key on Mac.", "Ключ на Mac.")},
			{Key: "action", Label: "Action (enable|disable|extend|status)", Value: "enable", Short: "luci", Detail: ph("Default enable 1h TTL.", "Вкл на 1ч по умолчанию.")},
			{Key: "hours", Label: "Hours", Value: "1", Short: "1/4/24/72", Detail: ph("TTL or extend amount.", "TTL или продление.")},
		}
	case "nvr":
		return []wizField{
			{Key: "action", Label: "Action (leases|add|probe|rec-start|rec-stop|status)", Value: "status",
				Short:  ph("NVR operation", "Операция NVR"),
				Detail: ph("Runs against primary (remote in settings).", "Через primary из настроек.")},
			{Key: "device_id", Label: FormT(lang, "edge_device_id"), Value: s.LastEdgeID, Short: "edge id", Detail: ph("For DHCP leases on edge.", "Для DHCP leases на edge.")},
			{Key: "cam_name", Label: FormT(lang, "cam_name"), Short: "camera id/name", Detail: ph("Used by add/probe/record.", "Для add/probe/record.")},
			{Key: "cam_ip", Label: FormT(lang, "cam_ip"), Short: "LAN IP", Detail: ph("Camera address on site LAN.", "IP камеры в LAN.")},
			{Key: "cam_pass", Label: FormT(lang, "cam_pass"), Secret: true, Short: "cam password", Detail: ph("Tapo/account password as configured.", "Пароль камеры/аккаунта.")},
		}
	case "addons":
		hostDef := s.RemoteHost
		keyDef := s.RemoteKey
		if keyDef == "" {
			keyDef = "~/.ssh/netductor_primary"
		}
		return []wizField{
			{Key: "host", Label: FormT(lang, "ssh_host") + " / IP", Value: hostDef,
				Short:  ph("Any VPS with netductor (often primary)", "Любая VPS с netductor (часто primary)"),
				Detail: ph("SSH target for selected add-ons. Defaults to saved primary in Settings.", "Куда ставить выбранные дополнения. По умолчанию primary из Настроек.")},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: orDefault(s.RemoteUser, "root"),
				Short: "SSH user", Detail: "root"},
			{Key: "key_path", Label: FormT(lang, "ssh_key_path"), Value: keyDef,
				Short:  ph("Operator private key on this Mac", "Приватный ключ оператора на Mac"),
				Detail: ph("Key that already has access (pubkey on the VPS).", "Ключ, к которому уже есть доступ (pubkey на VPS).")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase"), Secret: true,
				Short:  ph("If key is encrypted", "Если ключ с фразой"),
				Detail: ph("Leave empty if none.", "Пусто, если нет.")},
			{Key: "lampac", Label: "Lampac", Value: "no", Toggle: true,
				Short:  ph("Docker · 127.0.0.1:9118", "Docker · 127.0.0.1:9118"),
				Detail: ph("Requires Docker on target. Binds localhost only.", "Нужен Docker на цели. Только localhost.")},
			{Key: "telegram", Label: "Telegram bot", Value: "no", Toggle: true,
				Short:  ph("netductor-tg unit + secrets", "Юнит netductor-tg + секреты"),
				Detail: ph("After toggle ON: fill token + admin id below (or leave if already on VPS).", "После ON: токен и admin id ниже (или уже лежат на VPS).")},
			{Key: "tg_token", Label: FormT(lang, "tg_token"), Secret: true,
				Short:  ph("Required if installing Telegram", "Нужен при установке Telegram"),
				Detail: ph("BotFather token written to /etc/netductor/secrets on target.", "Пишется в secrets на целевой VPS.")},
			{Key: "tg_admin", Label: FormT(lang, "tg_admin"),
				Short:  ph("Telegram numeric user id", "Числовой Telegram user id"),
				Detail: ph("Admin allowed to control the bot.", "Админ бота.")},
			{Key: "go2rtc", Label: "go2rtc", Value: "no", Toggle: true,
				Short:  ph("NVR helper (soon)", "NVR helper (скоро)"),
				Detail: ph("Placeholder until hardware e2e.", "Заглушка до hardware e2e.")},
			{Key: "git_registry", Label: "Git + Registry", Value: "no", Toggle: true,
				Short:  ph("Thin git + local registry", "Thin git + локальный registry"),
				Detail: ph("Optional. Not part of core primary install.", "Опционально. Не входит в ядро primary.")},
		}
	case "mikrotik":
		return []wizField{
			{Key: "host", Label: "MikroTik host", Short: "ROS IP", Detail: ph("RouterOS management address.", "Адрес управления ROS.")},
			{Key: "user", Label: "User", Value: "admin", Short: "API/SSH user", Detail: "admin"},
			{Key: "password", Label: "Password", Secret: true, Short: "ROS password", Detail: ph("First connection.", "Первое подключение.")},
			{Key: "name", Label: "Site name", Short: "logical site", Detail: ph("Stored site id.", "Имя сайта.")},
		}
	default:
		return nil
	}
}

func (m model) runWizardApplyInTUI() string {
	lang := detectLang()
	s := loadTUISettings()
	switch m.wizTarget {
	case wizPrimary:
		spec := operator.PrimaryFromFields(m.fieldVal)
		if spec.Host == "" {
			return FormT(lang, "primary_host") + " required"
		}
		spec.Version = deploy.Release
		if err := operator.DeployPrimary(spec); err != nil {
			return err.Error()
		}
		s.RemoteHost = spec.Host
		s.RemoteUser = orDefault(spec.User, "root")
		s.RemoteKey = spec.SSHPrivateKey
		_ = saveTUISettings(s)
		return TT(lang, "Primary deploy finished: "+spec.Host, "Primary готов: "+spec.Host)

	case wizFleet:
		f := operator.FleetFromFields(m.fieldVal)
		f.Primary.Version = deploy.Release
		rep := func(st operator.Step) {
			if st.Err != "" {
				fmt.Fprintf(os.Stderr, "step %s ERROR: %s\n", st.ID, st.Err)
				return
			}
			if st.Done {
				fmt.Fprintf(os.Stderr, "step %s done: %s\n", st.ID, st.Message)
				return
			}
			fmt.Fprintf(os.Stderr, "step %s: %s\n", st.ID, st.Message)
		}
		if err := operator.FleetDeployWithReport(f, rep); err != nil {
			return err.Error()
		}
		if f.DoPrimary {
			s.RemoteHost = f.Primary.Host
			s.RemoteUser = orDefault(f.Primary.User, "root")
			s.RemoteKey = f.Primary.SSHPrivateKey
			_ = saveTUISettings(s)
		}
		return TT(lang, "Fleet deploy finished", "Fleet готов")

	case wizSecondary:
		if s.RemoteHost == "" || s.RemoteKey == "" {
			return FormT(lang, "set_primary_first")
		}
		sec := operator.SecondaryFromFields(m.fieldVal)
		sec.PrimaryHost = s.RemoteHost
		sec.PrimaryUser = orDefault(s.RemoteUser, "root")
		sec.PrimaryKey = s.RemoteKey
		if err := operator.DeploySecondary(sec); err != nil {
			return err.Error()
		}
		return TT(lang, "Secondary deploy finished", "Secondary готов")
	case wizOpenWrt:
		if s.RemoteHost == "" || s.RemoteKey == "" {
			return FormT(lang, "set_primary_first")
		}
		edge := operator.EdgeFromFields(m.fieldVal)
		edge.PrimaryHost = s.RemoteHost
		edge.PrimaryUser = orDefault(s.RemoteUser, "root")
		edge.PrimaryKey = s.RemoteKey
		edge.Version = deploy.Release
		if edge.RouterUser == "" {
			edge.RouterUser = "root"
		}
		if err := operator.DeployEdge(edge); err != nil {
			return err.Error()
		}
		if edge.DeviceID != "" {
			s.LastEdgeID = edge.DeviceID
			_ = saveTUISettings(s)
		}
		return TT(lang, "Edge provisioned: "+edge.DeviceID, "Edge: "+edge.DeviceID)

	case wizLuci:
		hours := 1.0
		fmt.Sscanf(m.fieldVal("hours"), "%f", &hours)
		out, err := deploy.LuciSSH(m.fieldVal("password"), expandHome(m.fieldVal("key")), orDefault(m.fieldVal("user"), "root"), m.fieldVal("router"), m.fieldVal("action"), hours, "")
		if err != nil {
			return err.Error()+"\n"+out
		}
		return out
	case wizNVR:
		mm := &model{remoteHost: s.RemoteHost, remoteUser: s.RemoteUser, remoteKey: s.RemoteKey}
		switch m.fieldVal("action") {
		case "leases":
			return mm.runNetductor("nvr", "leases", m.fieldVal("device_id"), "--wait")
		case "add":
			return mm.runNetductor("nvr", "cameras", "add", "name="+m.fieldVal("cam_name"), "ip="+m.fieldVal("cam_ip"), "password="+m.fieldVal("cam_pass"))
		case "probe":
			return mm.runNetductor("nvr", "probe", m.fieldVal("cam_name"))
		case "rec-start":
			return mm.runNetductor("nvr", "record", "start", m.fieldVal("cam_name"))
		case "rec-stop":
			return mm.runNetductor("nvr", "record", "stop", m.fieldVal("cam_name"))
		default:
			return mm.runNetductor("nvr", "status")
		}
	case "addons":
		host := orDefault(m.fieldVal("host"), s.RemoteHost)
		user := orDefault(m.fieldVal("user"), orDefault(s.RemoteUser, "root"))
		keyPath := expandHome(orDefault(m.fieldVal("key_path"), s.RemoteKey))
		pass := m.fieldVal("key_pass")
		if host == "" || keyPath == "" {
			return TT(lang, "Host and SSH key path required (or set primary in Settings).", "Нужны host и путь к SSH-ключу (или primary в Настройках).")
		}
		var parts []string
		any := false
		if yesish(m.fieldVal("lampac")) {
			any = true
			out, err := operator.RunRemote(user, host, keyPath, pass, "netductor install lampac")
			parts = append(parts, "=== lampac @ "+host+" ===", out)
			if err != nil {
				parts = append(parts, err.Error())
			}
		}
		if yesish(m.fieldVal("telegram")) {
			any = true
			tok := m.fieldVal("tg_token")
			adm := m.fieldVal("tg_admin")
			script := "set -e; mkdir -p /etc/netductor/secrets; chmod 700 /etc/netductor/secrets; "
			if tok != "" {
				script += fmt.Sprintf("printf '%%s\n' %s > /etc/netductor/secrets/telegram_bot_token; ", strconv.Quote(tok))
			}
			if adm != "" {
				script += fmt.Sprintf("printf '%%s\n' %s > /etc/netductor/secrets/telegram_admin_id; ", strconv.Quote(adm))
			}
			script += "chmod 600 /etc/netductor/secrets/* 2>/dev/null || true; "
			script += "netductor install telegram; systemctl restart netductor-telegram-bot || true; systemctl is-active netductor-telegram-bot || true"
			out, err := operator.RunRemote(user, host, keyPath, pass, script)
			parts = append(parts, "=== telegram @ "+host+" ===", out)
			if err != nil {
				parts = append(parts, err.Error())
			}
			if tok == "" {
				parts = append(parts, TT(lang, "(no token in form — used secrets already on VPS if present)", "(токен в форме пуст — если на VPS уже есть secrets, использованы они)"))
			}
		}
		if yesish(m.fieldVal("go2rtc")) {
			any = true
			parts = append(parts, "=== go2rtc @ "+host+" ===",
				TT(lang, "go2rtc addon not fully wired yet (planned after hardware e2e).", "go2rtc пока в плане после hardware e2e."))
		}
		if yesish(m.fieldVal("git_registry")) {
			any = true
			cmd := "command -v git >/dev/null || apt-get install -y -qq git; " +
				"netductor registry ensure; netductor registry crane; " +
				"netductor git init netductor 2>/dev/null || true; netductor git pipelines; netductor registry status"
			out, err := operator.RunRemote(user, host, keyPath, pass, cmd)
			parts = append(parts, "=== git+registry @ "+host+" ===", out)
			if err != nil {
				parts = append(parts, err.Error())
			}
		}
		if !any {
			return TT(lang, "Nothing selected (all off).", "Ничего не выбрано (всё выкл).")
		}
		return strings.Join(parts, "\n")
	case wizMikroTik:
		out, err := operator.DeploySite(operator.SiteSpec{
			SiteID:          orDefault(m.fieldVal("name"), "site"),
			Name:            m.fieldVal("name"),
			MTHost:          m.fieldVal("host"),
			MTUser:          orDefault(m.fieldVal("user"), "admin"),
			MTPass:          m.fieldVal("password"),
			MTPort:          22,
			DoPush:          true,
			OperatorKeyPath: s.RemoteKey,
			MikroTikID:      m.fieldVal("name"),
		})
		if err != nil {
			return out + "\n" + err.Error()
		}
		return out
	default:
		return "unknown target"
	}
}
