package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
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
				Short: ph("VPS public IP or DNS", "Публичный IP или DNS VPS"),
				Detail: ph("Abroad control-plane host reachable over SSH on port 22 (first login).", "Зарубежный control plane: SSH на порт 22 при первом входе.")},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: "root",
				Short: ph("SSH user", "Пользователь SSH"),
				Detail: ph("Usually root on a fresh Debian VPS.", "Обычно root на чистом Debian VPS.")},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true,
				Short: ph("Only for first bootstrap", "Только первый вход"),
				Detail: ph("Used once with sshpass; then password auth is disabled.", "Один раз через sshpass; дальше вход только по ключу.")},
			{Key: "gen_key", Label: FormT(lang, "gen_ssh_key") + " (yes/no)", Value: "yes",
				Short: ph("yes = create ed25519 key", "yes = создать ключ ed25519"),
				Detail: ph("Creates ~/.ssh/netductor_primary unless path overridden.", "Создаёт ~/.ssh/netductor_primary, если путь не задан иначе.")},
			{Key: "key_path", Label: FormT(lang, "ssh_key_path"), Value: orDefault(s.RemoteKey, "~/.ssh/netductor_primary"),
				Short: ph("Private key path on Mac", "Путь к private key на Mac"),
				Detail: ph("Saved into TUI settings as remote_key.", "Сохраняется в настройках TUI как remote_key.")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (empty=none)", Secret: true,
				Short: ph("Optional key encryption", "Опциональная фраза ключа"),
				Detail: ph("Leave empty for no passphrase. Needed later for secondary/edge if set.", "Пусто = без фразы. Если задали — понадобится для secondary/edge.")},
			{Key: "sni", Label: FormT(lang, "reality_sni"), Value: "api.vk.me",
				Short: ph("Reality TLS SNI", "SNI для Reality"),
				Detail: ph("SNI camouflage for VPN (e.g. api.vk.me).", "Маскировка VPN, напр. api.vk.me.")},
			{Key: "domain_base", Label: FormT(lang, "domain_base"), Value: "",
				Short: ph("Optional DNS base", "Опционально DNS base"),
				Detail: ph("e.g. netductor.neyman.top → primary./vpn./i. hosts + REDIRECT_BASE. Empty = skip.", "Напр. netductor.neyman.top. Пусто = пропуск.")},
			{Key: "le_email", Label: FormT(lang, "le_email"), Value: "",
				Short: ph("Let's Encrypt registration email", "Email для Let's Encrypt"),
				Detail: ph("With domain_base: LE for primary.+i. Empty = skip LE (http domain only).", "С domain_base: LE. Пусто = без LE.")},
		}
	case "secondary":
		return []wizField{
			{Key: "host", Label: FormT(lang, "secondary_host"),
				Short: ph("RU VPS address", "Адрес РФ VPS"),
				Detail: ph("VPN entry only. Primary must already be set in TUI.", "Только VPN entry. Primary уже должен быть в TUI.")},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: "root", Short: "SSH", Detail: ph("First login user.", "Пользователь первого входа.")},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true,
				Short: ph("First SSH password", "Пароль первого SSH"),
				Detail: ph("Mac pubkey installed; password auth then off.", "Ставится pubkey Mac; пароль отключается.")},
			{Key: "sni", Label: FormT(lang, "reality_sni"), Value: "api.vk.me", Short: "Reality SNI", Detail: ph("Usually same as primary.", "Обычно как на primary.")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (primary key)", Secret: true,
				Short: ph("If primary key has passphrase", "Если ключ primary с фразой"),
				Detail: ph("Unlocks Mac private key when SSHing to primary during provision.", "Нужна, чтобы с Mac ходить на primary при provision.")},
		}
	case "openwrt":
		srv := "https://PRIMARY:8789"
		if s.RemoteHost != "" {
			srv = "https://" + s.RemoteHost + ":8789"
		}
		return []wizField{
			{Key: "router", Label: FormT(lang, "router_lan_ip"), Value: "192.168.1.1",
				Short: ph("Current SSH reachability IP", "IP, куда сейчас ходит SSH"),
				Detail: ph("Must be reachable from this Mac over LAN.", "Должен быть доступен с Mac по LAN.")},
			{Key: "password", Label: FormT(lang, "ssh_password_first"), Secret: true, Short: "root password", Detail: ph("OpenWrt root password.", "Пароль root OpenWrt.")},
			{Key: "id", Label: FormT(lang, "edge_device_id"), Value: orDefault(s.LastEdgeID, "edge-1"),
				Short: ph("Stable edge id", "Стабильный id edge"),
				Detail: ph("Used for enroll/approve on primary.", "Для enroll/approve на primary.")},
			{Key: "arch", Label: FormT(lang, "agent_arch"), Value: "arm64", Short: "arm64 / armv7 / …", Detail: ph("Match router CPU.", "Под CPU роутера.")},
			{Key: "server", Label: FormT(lang, "primary_mtls"), Value: srv,
				Short: "https://PRIMARY:8789",
				Detail: ph("Agent plane. Must be reachable from the router.", "Agent plane. Должен быть доступен с роутера.")},
			{Key: "net", Label: "Configure network? (yes/no)", Value: "no",
				Short: ph("Apply LAN/Wi-Fi/WAN now", "Применить LAN/Wi-Fi/WAN сейчас"),
				Detail: ph("If yes, fill LAN/Wi-Fi/WAN fields below.", "Если yes — заполните поля сети ниже.")},
			{Key: "lan_ip", Label: "LAN IP", Value: "192.168.50.1", Short: ph("New router address", "Новый адрес роутера"),
				Detail: ph("Prefer different subnets per site.", "На разных точках лучше разные подсети.")},
			{Key: "wifi_ssid", Label: "Wi-Fi SSID 2.4", Short: "2.4 GHz", Detail: ph("Empty 5 GHz inherits 2.4.", "Пустой 5 GHz = как 2.4.")},
			{Key: "wifi_key", Label: "Wi-Fi password", Secret: true, Short: "PSK", Detail: ph("WPA2/3 key.", "Ключ WPA.")},
			{Key: "wan_proto", Label: "WAN (dhcp|static|pppoe)", Value: "dhcp", Short: "WAN type", Detail: ph("Provider uplink type.", "Тип uplink провайдера.")},
			{Key: "guest", Label: "Guest Wi-Fi? (yes/no)", Value: "no", Short: "Guest SSID", Detail: ph("Optional captive guest network.", "Опциональная гостевая сеть.")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase") + " (primary key)", Secret: true, Short: "Mac key", Detail: ph("If primary key is encrypted.", "Если ключ primary с фразой.")},
		}
	case "nvr":
		return []wizField{
			{Key: "action", Label: "Action (leases|add|probe|rec-start|rec-stop|status)", Value: "status",
				Short: ph("NVR operation", "Операция NVR"),
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
				Short: ph("Any VPS with netductor (often primary)", "Любая VPS с netductor (часто primary)"),
				Detail: ph("SSH target for selected add-ons. Defaults to saved primary in Settings.", "Куда ставить выбранные дополнения. По умолчанию primary из Настроек.")},
			{Key: "user", Label: FormT(lang, "ssh_user"), Value: orDefault(s.RemoteUser, "root"),
				Short: "SSH user", Detail: "root"},
			{Key: "key_path", Label: FormT(lang, "ssh_key_path"), Value: keyDef,
				Short: ph("Operator private key on this Mac", "Приватный ключ оператора на Mac"),
				Detail: ph("Key that already has access (pubkey on the VPS).", "Ключ, к которому уже есть доступ (pubkey на VPS).")},
			{Key: "key_pass", Label: FormT(lang, "key_passphrase"), Secret: true,
				Short: ph("If key is encrypted", "Если ключ с фразой"),
				Detail: ph("Leave empty if none.", "Пусто, если нет.")},
			{Key: "lampac", Label: "Lampac", Value: "no", Toggle: true,
				Short: ph("Docker · 127.0.0.1:9118", "Docker · 127.0.0.1:9118"),
				Detail: ph("Requires Docker on target. Binds localhost only.", "Нужен Docker на цели. Только localhost.")},
			{Key: "telegram", Label: "Telegram bot", Value: "no", Toggle: true,
				Short: ph("netductor-tg unit + secrets", "Юнит netductor-tg + секреты"),
				Detail: ph("After toggle ON: fill token + admin id below (or leave if already on VPS).", "После ON: токен и admin id ниже (или уже лежат на VPS).")},
			{Key: "tg_token", Label: FormT(lang, "tg_token"), Secret: true,
				Short: ph("Required if installing Telegram", "Нужен при установке Telegram"),
				Detail: ph("BotFather token written to /etc/netductor/secrets on target.", "Пишется в secrets на целевой VPS.")},
			{Key: "tg_admin", Label: FormT(lang, "tg_admin"),
				Short: ph("Telegram numeric user id", "Числовой Telegram user id"),
				Detail: ph("Admin allowed to control the bot.", "Админ бота.")},
			{Key: "go2rtc", Label: "go2rtc", Value: "no", Toggle: true,
				Short: ph("NVR helper (soon)", "NVR helper (скоро)"),
				Detail: ph("Placeholder until hardware e2e.", "Заглушка до hardware e2e.")},
			{Key: "git_registry", Label: "Git + Registry", Value: "no", Toggle: true,
				Short: ph("Thin git + local registry", "Thin git + локальный registry"),
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
		host := m.fieldVal("host")
		if host == "" {
			return FormT(lang, "primary_host") + " required"
		}
		keyPath := expandHome(orDefault(m.fieldVal("key_path"), "~/.ssh/netductor_primary"))
		err := deploy.DeployPrimary(deploy.PrimaryOpts{
			Host: host, User: orDefault(m.fieldVal("user"), "root"), Password: m.fieldVal("password"),
			SSHPrivateKey: keyPath, GenerateKey: yesish(m.fieldVal("gen_key")),
			KeyPassphrase: m.fieldVal("key_pass"), WithLampac: false, // add-ons via Wizard → Дополнения
			Version: deploy.Release, // TG bot via Wizard → Add-ons (token/admin there)
			TelegramToken: "", TelegramAdminID: "",
			SNI: orDefault(m.fieldVal("sni"), "api.vk.me"),
			DomainLE: strings.TrimSpace(m.fieldVal("domain_base")) != "" && strings.TrimSpace(m.fieldVal("le_email")) != "",
			DomainEmail: m.fieldVal("le_email"),
			DomainHTTP: strings.TrimSpace(m.fieldVal("domain_base")) != "" && strings.TrimSpace(m.fieldVal("le_email")) == "",
			DomainBase: strings.TrimSpace(m.fieldVal("domain_base")),
		})
		if err != nil {
			return err.Error()
		}
		s.RemoteHost = host
		s.RemoteUser = orDefault(m.fieldVal("user"), "root")
		s.RemoteKey = keyPath
		_ = saveTUISettings(s)
		return TT(lang, "Primary deploy finished: "+host, "Primary готов: "+host)
	case wizSecondary:
		if s.RemoteHost == "" || s.RemoteKey == "" {
			return FormT(lang, "set_primary_first")
		}
		err := deploy.DeploySecondary(deploy.SecondaryOpts{
			PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
			PrimaryKeyPassphrase: m.fieldVal("key_pass"),
			SecondaryHost: m.fieldVal("host"), SecondaryUser: orDefault(m.fieldVal("user"), "root"),
			SecondaryPass: m.fieldVal("password"), SNI: orDefault(m.fieldVal("sni"), "api.vk.me"),
		})
		if err != nil {
			return err.Error()
		}
		return TT(lang, "Secondary deploy finished", "Secondary готов")
	case wizOpenWrt:
		id := m.fieldVal("id")
		err := deploy.DeployEdge(deploy.EdgeOpts{
			PrimaryHost: s.RemoteHost, PrimaryUser: orDefault(s.RemoteUser, "root"), PrimaryKey: s.RemoteKey,
			PrimaryKeyPassphrase: m.fieldVal("key_pass"),
			RouterHost: m.fieldVal("router"), RouterUser: "root", RouterPass: m.fieldVal("password"),
			DeviceID: id, ServerURL: m.fieldVal("server"), AgentArch: orDefault(m.fieldVal("arch"), "arm64"),
			Version: deploy.Release, NetConfigure: yesish(m.fieldVal("net")),
			LANIP: m.fieldVal("lan_ip"), WiFiSSID24: m.fieldVal("wifi_ssid"), WiFiKey24: m.fieldVal("wifi_key"),
			WANProto: m.fieldVal("wan_proto"), GuestEnable: yesish(m.fieldVal("guest")),
		})
		if err != nil {
			return err.Error()
		}
		s.LastEdgeID = id
		_ = saveTUISettings(s)
		return TT(lang, "Edge provisioned: "+id, "Edge: "+id)
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
			out, err := deploy.RunOnPrimary(host, user, keyPath, pass, "netductor install lampac")
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
			out, err := deploy.RunOnPrimary(host, user, keyPath, pass, script)
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
			out, err := deploy.RunOnPrimary(host, user, keyPath, pass, cmd)
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
		return TT(lang, "MikroTik: use Ops/CLI for full site push", "MikroTik: Ops/CLI")
	default:
		return "unknown target"
	}
}
