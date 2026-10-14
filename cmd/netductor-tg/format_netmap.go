package main

import (
	"strings"

	"github.com/PavelNeyman/netductor/internal/netmap"
)

func formatNetmapHTML() string {
	ru := getLang() != "en"
	title := T("netmap_title")
	var b strings.Builder
	b.WriteString("🗺 <b>" + title + "</b>\n")
	b.WriteString("<pre>")
	b.WriteString(esc(netmap.Text(ru)))
	b.WriteString("</pre>")
	return b.String()
}
