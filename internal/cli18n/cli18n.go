package cli18n

import (
	"fmt"
	"os"
	"strings"
)

// Lang is "en" or "ru".
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

var dict = map[string]map[string]string{
	"en": {
		"mtls.help": `netductor mtls ensure                 — generate CA/server/client certs for agent plane
netductor mtls issue-client <id>     — per-node client cert under secrets/mtls/clients/<id>
netductor mtls list                  — plane + per-node certs (serial, expiry, pending rotate)
netductor mtls revoke <serial|node>  — revoke by serial hex or node id
netductor mtls rotate <node-id>      — re-issue cert, enqueue mtls_refresh (grace before revoke)
netductor mtls revoked               — list revoke entries
netductor mtls rollover start|status|issue <id>|finish|abort — dual-CA rollover
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
		"mtls.rotated":            "rotated node serial days_left",
		"mtls.material":           "material:",
		"mtls.enqueued_edge":      "enqueued edge mtls_refresh id=",
		"mtls.enqueued_secondary": "enqueued secondary mtls_refresh",
		"mtls.push_manual":        "could not enqueue push — re-provision or wait for manual copy",
		"mtls.grace_note":         "old serial stays valid for grace hours:",
		"mtls.empty":              "(empty)",
		"mtls.rollover_started":   "CA dual-trust started (ca-new). Issue clients, then finish.",
		"mtls.rollover_issued":    "issued from ca-new + push queued for",
		"mtls.rollover_done":      "CA rollover finished; ca-new is now primary",
		"mtls.rollover_abort":     "CA rollover aborted; ca-new removed",
		"mtls.unknown":            "unknown mtls subcommand",
		"doctor.header":           "netductor doctor",
	},
	"ru": {
		"mtls.help": `netductor mtls ensure                 — создать CA/server/client для agent plane
netductor mtls issue-client <id>     — client-сертификат узла в secrets/mtls/clients/<id>
netductor mtls list                  — plane + узлы (serial, срок, pending rotate)
netductor mtls revoke <serial|node>  — отозвать serial или узел
netductor mtls rotate <node-id>      — перевыпуск, очередь mtls_refresh (grace до revoke)
netductor mtls revoked               — список отзывов
netductor mtls rollover start|status|issue <id>|finish|abort — смена CA (dual-trust)
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
		"mtls.rotated":            "перевыпущен узел serial days_left",
		"mtls.material":           "материал:",
		"mtls.enqueued_edge":      "в очередь edge mtls_refresh id=",
		"mtls.enqueued_secondary": "в очередь secondary mtls_refresh",
		"mtls.push_manual":        "не удалось поставить push — re-provision или копирование вручную",
		"mtls.grace_note":         "старый serial действует (часы grace):",
		"mtls.empty":              "(пусто)",
		"mtls.rollover_started":   "Dual-trust CA начат (ca-new). Выдайте клиентам, затем finish.",
		"mtls.rollover_issued":    "выпущен от ca-new + push для",
		"mtls.rollover_done":      "Rollover CA завершён; ca-new — основная",
		"mtls.rollover_abort":     "Rollover отменён; ca-new удалён",
		"mtls.unknown":            "неизвестная подкоманда mtls",
		"doctor.header":           "netductor doctor",
	},
}

// T returns localized string; missing keys fall back to key or EN.
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

// Register adds or overrides a key for tests / expansion.
func Register(lang, key, value string) {
	if dict[lang] == nil {
		dict[lang] = map[string]string{}
	}
	dict[lang][key] = value
}
