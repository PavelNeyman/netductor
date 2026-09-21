package cli18n

import (
	"fmt"
	"os"
	"strings"
)

func Lang() string {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("NETDUCTOR_LANG"))); v != "" {
		if strings.HasPrefix(v, "ru") {
			return "ru"
		}
		return "en"
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
		if v == "" || v == "c" || v == "posix" {
			continue
		}
		if strings.HasPrefix(v, "ru") {
			return "ru"
		}
	}
	return "en"
}

var dict = map[string]map[string]string{}

func init() {
	dict["en"] = map[string]string{
		"help.main": `netductor — network control plane

  tui|menu [--mode vps|openwrt|workstation|operator] …
  deploy primary|secondary|edge
  backup | restore | recover | fleet | audit | self-install | update
  version | doctor | status | vpn | sites | ssh-hosts | secondary | addons | edge | nvr | mtls | serve | install | probe | collect | help

  (no args on a TTY → interactive menu)

Workstation: docs/DEPLOY-WORKSTATION.md
`,
		"doctor.header":        "netductor doctor role=%s host=%s",
		"doctor.ok":            "OK  ",
		"doctor.warn":          "WARN",
		"doctor.fail":          "FAIL",
		"edge.usage":           "usage: netductor edge list|pending|approve|deny|revoke|register|recovery|export|import|set-site|cmd|provision …",
		"edge.approved":        "approved",
		"edge.denied":          "denied",
		"edge.revoked":         "revoked",
		"edge.registered":      "registered pending",
		"edge.provisioned":     "provisioned",
		"edge.imported":        "imported",
		"edge.ok":              "ok",
		"vpn.usage":            "usage: netductor vpn add|list|rename|link|refresh-links|note|disable|enable|revoke|apply|set-sni|session …",
		"vpn.refreshed":        "refreshed %d",
		"secondary.usage":      "usage: netductor secondary export|join|links|status|sync|exit|provision …",
		"secondary.pull_hint":  "secondaries will pull on next heartbeat (~30s)",
		"install.usage":        "usage: netductor install [flags]",
		"nvr.usage":            "usage: netductor nvr …",
		"mtls.help": `netductor mtls ensure | issue-client | list | revoke | rotate | revoked
netductor mtls rollover start|status|issue <id>|finish|abort
`,
		"mtls.ready":              "mtls ready:",
		"mtls.usage.issue":        "usage: netductor mtls issue-client <node-id>",
		"mtls.client_material":    "client material:",
		"mtls.clients_none":       "clients: (none)",
		"mtls.pending_header":     "pending rotates (grace):",
		"mtls.usage.revoke":       "usage: netductor mtls revoke <serial-hex|node-id>",
		"mtls.revoked_node":       "revoked node",
		"mtls.revoked_serial":     "revoked serial",
		"mtls.usage.rotate":       "usage: netductor mtls rotate <node-id>",
		"mtls.rotated":            "rotated",
		"mtls.material":           "material:",
		"mtls.enqueued_edge":      "enqueued edge mtls_refresh id=",
		"mtls.enqueued_secondary": "enqueued secondary mtls_refresh",
		"mtls.push_manual":        "could not enqueue push — re-provision or copy manually",
		"mtls.grace_note":         "old serial valid for grace hours:",
		"mtls.empty":              "(empty)",
		"mtls.rollover_started":   "CA dual-trust started (ca-new)",
		"mtls.rollover_issued":    "issued from ca-new + push queued for",
		"mtls.rollover_done":      "CA rollover finished",
		"mtls.rollover_abort":     "CA rollover aborted",
		"mtls.unknown":            "unknown mtls subcommand",
		"sites.usage":             "usage: netductor sites …",
		"nodes.usage":             "usage: netductor nodes …",
		"fleet.usage":             "usage: netductor fleet …",
		"backup.usage":            "usage: netductor backup …",
		"status.header":           "netductor status",
	}
	dict["ru"] = map[string]string{
		"help.main": `netductor — плоскость управления сетью

  tui|menu [--mode vps|openwrt|workstation|operator] …
  deploy primary|secondary|edge
  backup | restore | recover | fleet | audit | self-install | update
  version | doctor | status | vpn | sites | ssh-hosts | secondary | addons | edge | nvr | mtls | serve | install | probe | collect | help

  (без аргументов в TTY → интерактивное меню)

Workstation: docs/DEPLOY-WORKSTATION.md
`,
		"doctor.header":           "netductor doctor role=%s host=%s",
		"doctor.ok":               "OK  ",
		"doctor.warn":             "WARN",
		"doctor.fail":             "FAIL",
		"edge.usage":              "использование: netductor edge list|pending|approve|deny|revoke|register|recovery|export|import|set-site|cmd|provision …",
		"edge.approved":           "одобрен",
		"edge.denied":             "отклонён",
		"edge.revoked":            "отозван",
		"edge.registered":         "зарегистрирован (pending)",
		"edge.provisioned":        "provision выполнен",
		"edge.imported":           "импортировано",
		"edge.ok":                 "ok",
		"vpn.usage":               "использование: netductor vpn add|list|rename|link|refresh-links|note|disable|enable|revoke|apply|set-sni|session …",
		"vpn.refreshed":           "обновлено %d",
		"secondary.usage":         "использование: netductor secondary export|join|links|status|sync|exit|provision …",
		"secondary.pull_hint":     "secondary подтянут на следующем heartbeat (~30с)",
		"install.usage":           "использование: netductor install [флаги]",
		"nvr.usage":               "использование: netductor nvr …",
		"mtls.help": `netductor mtls ensure | issue-client | list | revoke | rotate | revoked
netductor mtls rollover start|status|issue <id>|finish|abort
`,
		"mtls.ready":              "mtls готов:",
		"mtls.usage.issue":        "использование: netductor mtls issue-client <node-id>",
		"mtls.client_material":    "материал клиента:",
		"mtls.clients_none":       "клиенты: (нет)",
		"mtls.pending_header":     "ожидают rotate (grace):",
		"mtls.usage.revoke":       "использование: netductor mtls revoke <serial|node-id>",
		"mtls.revoked_node":       "отозван узел",
		"mtls.revoked_serial":     "отозван serial",
		"mtls.usage.rotate":       "использование: netductor mtls rotate <node-id>",
		"mtls.rotated":            "перевыпущен",
		"mtls.material":           "материал:",
		"mtls.enqueued_edge":      "в очередь edge mtls_refresh id=",
		"mtls.enqueued_secondary": "в очередь secondary mtls_refresh",
		"mtls.push_manual":        "не удалось поставить push — re-provision вручную",
		"mtls.grace_note":         "старый serial действует (часы grace):",
		"mtls.empty":              "(пусто)",
		"mtls.rollover_started":   "Dual-trust CA начат (ca-new)",
		"mtls.rollover_issued":    "выпущен от ca-new + push для",
		"mtls.rollover_done":      "Rollover CA завершён",
		"mtls.rollover_abort":     "Rollover отменён",
		"mtls.unknown":            "неизвестная подкоманда mtls",
		"sites.usage":             "использование: netductor sites …",
		"nodes.usage":             "использование: netductor nodes …",
		"fleet.usage":             "использование: netductor fleet …",
		"backup.usage":            "использование: netductor backup …",
		"status.header":           "netductor status",
	}
}

func T(key string, args ...any) string {
	lang := Lang()
	s, ok := dict[lang][key]
	if !ok {
		s, ok = dict["en"][key]
	}
	if !ok {
		s = key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

func Register(lang, key, value string) {
	if dict[lang] == nil {
		dict[lang] = map[string]string{}
	}
	dict[lang][key] = value
}
