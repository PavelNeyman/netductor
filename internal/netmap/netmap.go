// Package netmap is the canonical CIDR ownership map for operator UI (Web/TUI/TG)
// and field hints (edge shop LAN, service-net, etc.).
package netmap

import (
	"fmt"
	"strings"
)

// Row is one network in the ownership table.
type Row struct {
	ID        string // home | work | svc_paths | service_net | shops
	NetworkEN string
	NetworkRU string
	CIDR      string
	PurposeEN string
	PurposeRU string
	HintEN    string // short field hint
	HintRU    string
}

// Rows is the fixed product map (keep in sync with SR profiles / docs).
func Rows() []Row {
	return []Row{
		{
			ID: "home", NetworkEN: "Home", NetworkRU: "Дом",
			CIDR: "10.9.8.0/24",
			PurposeEN: "Mac en0; SR → DIRECT",
			PurposeRU: "Mac en0; в SR — DIRECT",
			HintEN: "Home LAN — never use for shop/site overlays",
			HintRU: "Домашняя сеть — не для магазинов/site",
		},
		{
			ID: "work", NetworkEN: "Work", NetworkRU: "Work",
			CIDR: "10.1.253/24, 10.2.27/24, 172.28.*, 172.30.*, 192.168.20/40/70…",
			PurposeEN: "OpenConnect split; SR → DIRECT",
			PurposeRU: "OpenConnect split; в SR — DIRECT",
			HintEN: "Corporate prefixes from operator SR work list",
			HintRU: "Корпоративные префиксы из work-списка SR",
		},
		{
			ID: "svc_paths", NetworkEN: "Nodes / svc-paths", NetworkRU: "Узлы / svc-paths",
			CIDR: "10.87.10.0/30 … 10.87.13.0/30",
			PurposeEN: "SP/PS, build PB/BP tunnels",
			PurposeRU: "SP/PS, build PB/BP туннели",
			HintEN: "Internal node links — not for client LAN",
			HintRU: "Служебные линки нод — не LAN клиентов",
		},
		{
			ID: "service_net", NetworkEN: "Service-net (VIP)", NetworkRU: "Service-net (VIP)",
			CIDR: "10.88.0.0/24",
			PurposeEN: "lampac/git/registry VIP on primary; SR → PROXY",
			PurposeRU: "VIP lampac/git/registry на primary; SR → PROXY",
			HintEN: "Reach via VPN (198.18-style path on client → 10.88 VIP)",
			HintRU: "Доступ через VPN (на клиенте PROXY → VIP 10.88)",
		},
		{
			ID: "shops", NetworkEN: "Shops / site overlays", NetworkRU: "Магазины / site overlays",
			CIDR: "10.120.0.0/16",
			PurposeEN: "OpenWrt site LAN; shop N → 10.120.N.0/24; SR → PROXY",
			PurposeRU: "LAN OpenWrt-сайтов; магазин N → 10.120.N.0/24; SR → PROXY",
			HintEN: "Prefer 10.120.<site>.0/24 for edge LAN (not 192.168.1.0/24 if avoidable)",
			HintRU: "Для edge LAN лучше 10.120.<сайт>.0/24 (не 192.168.1.0/24, если можно)",
		},
	}
}

func (r Row) Network(ru bool) string {
	if ru {
		return r.NetworkRU
	}
	return r.NetworkEN
}

func (r Row) Purpose(ru bool) string {
	if ru {
		return r.PurposeRU
	}
	return r.PurposeEN
}

func (r Row) Hint(ru bool) string {
	if ru {
		return r.HintRU
	}
	return r.HintEN
}

// Text mono table for TG / logs.
func Text(ru bool) string {
	var b strings.Builder
	if ru {
		b.WriteString("Карта сетей netductor\n")
	} else {
		b.WriteString("netductor network map\n")
	}
	for _, r := range Rows() {
		b.WriteString(fmt.Sprintf("• %s  %s\n  %s\n", r.Network(ru), r.CIDR, r.Purpose(ru)))
	}
	return b.String()
}

// HTMLTable for operator Web (escape-safe: we control strings).
func HTMLTable(ru bool) string {
	var b strings.Builder
	b.WriteString(`<table class="netmap"><thead><tr>`)
	if ru {
		b.WriteString(`<th>Сеть</th><th>CIDR</th><th>Назначение</th>`)
	} else {
		b.WriteString(`<th>Network</th><th>CIDR</th><th>Purpose</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, r := range Rows() {
		b.WriteString("<tr><td>")
		b.WriteString(r.Network(ru))
		b.WriteString("</td><td><code>")
		b.WriteString(r.CIDR)
		b.WriteString("</code></td><td>")
		b.WriteString(r.Purpose(ru))
		b.WriteString("</td></tr>")
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

// HintByID returns field hint or empty.
func HintByID(id string, ru bool) string {
	for _, r := range Rows() {
		if r.ID == id {
			return r.Hint(ru)
		}
	}
	return ""
}

// JSON-friendly export for /v1/netmap
func Export(ru bool) []map[string]string {
	var out []map[string]string
	for _, r := range Rows() {
		out = append(out, map[string]string{
			"id": r.ID, "network": r.Network(ru), "cidr": r.CIDR,
			"purpose": r.Purpose(ru), "hint": r.Hint(ru),
		})
	}
	return out
}
