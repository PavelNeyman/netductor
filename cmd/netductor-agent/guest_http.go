package main

import (
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/guest"
)

func guestConfigPath() string {
	return agentDir() + "/guest.json"
}

func guestStoreDir() string {
	return agentDir() + "/guest"
}

func startGuestHTTP(cfg *config) {
	gc, err := guest.LoadConfig(guestConfigPath())
	if err != nil || !gc.Enabled {
		return
	}
	store, err := guest.OpenStore(guestStoreDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "guest store: %v\n", err)
		return
	}
	// Captive on all interfaces (guest clients need it) — only shows code, no grant.
	go func() {
		addr := fmt.Sprintf(":%d", gc.CaptivePort)
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			mac := clientMAC(r)
			if mac == "" {
				mac = strings.TrimSpace(r.URL.Query().Get("mac"))
			}
			if mac == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprint(w, captiveHTML("", "", "Не удалось определить устройство. Откройте повторно или обратитесь к персоналу."))
				return
			}
			if store.IsAllowed(mac) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprint(w, captiveHTML(mac, "", "Доступ в интернет уже выдан. Можно закрыть страницу."))
				return
			}
			sess, err := store.EnsureSession(mac)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			grantURL := fmt.Sprintf("http://%s:%d/grant?t=%s", deskHostHint(), gc.DeskPort, sess.Token)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, captiveHTML(mac, sess.Code, grantURL))
		})
		fmt.Fprintf(os.Stderr, "guest captive on %s\n", addr)
		_ = http.ListenAndServe(addr, mux)
	}()
	// Desk: LAN only
	go func() {
		addr := pickPrivateListenAddr(fmt.Sprintf("%d", gc.DeskPort))
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if !allowRecoveryClient(r) {
				http.Error(w, "forbidden: private LAN only", http.StatusForbidden)
				return
			}
			deskHandler(w, r, &gc, store)
		})
		mux.HandleFunc("/grant", func(w http.ResponseWriter, r *http.Request) {
			if !allowRecoveryClient(r) {
				http.Error(w, "forbidden: private LAN only", http.StatusForbidden)
				return
			}
			deskGrantQR(w, r, &gc, store)
		})
		mux.HandleFunc("/api/grant", func(w http.ResponseWriter, r *http.Request) {
			if !allowRecoveryClient(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			deskAPIGrant(w, r, &gc, store)
		})
		fmt.Fprintf(os.Stderr, "guest desk on %s\n", addr)
		_ = http.ListenAndServe(addr, mux)
	}()
	_ = cfg
}

func deskHostHint() string {
	// Best-effort LAN IP for QR targeting staff on private Wi‑Fi
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}
			ip := ipnet.IP.String()
			if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
				return ip
			}
		}
	}
	return "192.168.1.1"
}

func clientMAC(r *http.Request) string {
	// OpenWrt may pass via header from redirector; also try ARP later.
	if m := r.Header.Get("X-Client-MAC"); m != "" {
		return guest.NormalizeMAC(m)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return arpMAC(host)
}

func arpMAC(ip string) string {
	b, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if fields[0] == ip && fields[3] != "00:00:00:00:00:00" {
			return guest.NormalizeMAC(fields[3])
		}
	}
	return ""
}

func captiveHTML(mac, code, extra string) string {
	codeBlock := ""
	if code != "" {
		codeBlock = fmt.Sprintf(`<p style="font-size:3rem;font-weight:700;letter-spacing:.15em">%s</p>
<p>Назовите этот код продавцу или покажите QR ниже.</p>`, html.EscapeString(code))
		if strings.HasPrefix(extra, "http") {
			codeBlock += fmt.Sprintf(`<p class="muted">Grant QR (для кассира, в своей Wi‑Fi):</p>
<p style="word-break:break-all;font-size:.85rem"><code>%s</code></p>
<p class="muted">На кассе откройте ссылку телефоном в <b>своей</b> сети магазина.</p>`, html.EscapeString(extra))
		}
	} else if extra != "" {
		codeBlock = `<p>` + html.EscapeString(extra) + `</p>`
	}
	return fmt.Sprintf(`<!DOCTYPE html><html lang="ru"><head><meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>Guest Wi‑Fi</title>
<style>
body{font-family:system-ui,sans-serif;max-width:26rem;margin:2rem auto;padding:0 1rem;text-align:center}
.muted{color:#666;font-size:.9rem}
</style></head><body>
<h1>Гостевой доступ</h1>
<p class="muted">Вы в гостевой сети. Интернет откроет сотрудник.</p>
%s
<p class="muted">%s</p>
</body></html>`, codeBlock, html.EscapeString(mac))
}

func deskHandler(w http.ResponseWriter, r *http.Request, gc *guest.Config, store *guest.Store) {
	msg := ""
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		pin := r.Form.Get("pin")
		if !gc.PINOK(pin) {
			msg = `<p class="err">Неверный PIN</p>`
		} else {
			code := strings.TrimSpace(r.Form.Get("code"))
			mins := 10
			fmt.Sscanf(r.Form.Get("minutes"), "%d", &mins)
			if mins < 1 {
				mins = 10
			}
			if mins > 24*60 {
				mins = 24 * 60
			}
			if code != "" {
				sess, ok := store.LookupCode(code)
				if !ok {
					msg = `<p class="err">Код не найден или истёк</p>`
				} else {
					e, err := store.Grant(sess.MAC, time.Duration(mins)*time.Minute, "desk:"+code)
					if err != nil {
						msg = `<p class="err">` + html.EscapeString(err.Error()) + `</p>`
					} else {
						_ = applyGuestFirewallAllow(store)
						msg = fmt.Sprintf(`<p class="ok">Доступ выдан до %s (%d мин)</p>`,
							e.ExpiresAt.Local().Format("15:04"), mins)
					}
				}
			}
		}
	}
	allow, pending := store.Snapshot()
	var pb, ab strings.Builder
	for _, p := range pending {
		pb.WriteString(fmt.Sprintf("<li><b>%s</b> · %s · до %s</li>", html.EscapeString(p.Code), html.EscapeString(p.MAC), p.ExpiresAt.Local().Format("15:04:05")))
	}
	for _, a := range allow {
		ab.WriteString(fmt.Sprintf("<li>%s · до %s</li>", html.EscapeString(a.MAC), a.ExpiresAt.Local().Format("15:04")))
	}
	if pb.Len() == 0 {
		pb.WriteString("<li class=muted>нет ожидающих</li>")
	}
	if ab.Len() == 0 {
		ab.WriteString("<li class=muted>пусто</li>")
	}
	join := guest.JoinQR(gc.SSID, gc.PSK, gc.Hidden)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="ru"><head><meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>Guest Desk</title>
<style>
body{font-family:system-ui,sans-serif;max-width:28rem;margin:1.5rem auto;padding:0 1rem}
input,select,button{width:100%%;padding:.55rem;margin:.35rem 0;font-size:1rem;box-sizing:border-box}
button{background:#1a7;color:#fff;border:0;border-radius:6px}
.err{color:#a00}.ok{color:#070}.muted{color:#666}
code{font-size:.8rem;word-break:break-all}
</style></head><body>
<h1>Guest Desk</h1>
%s
<form method="POST">
<label>PIN сотрудника<br/><input type="password" name="pin" required autocomplete="current-password"/></label>
<label>Код с экрана гостя<br/><input name="code" placeholder="7K2" autocomplete="off"/></label>
<label>Время доступа<br/>
<select name="minutes">
<option value="10" selected>10 минут</option>
<option value="30">30 минут</option>
<option value="60">1 час</option>
<option value="120">2 часа</option>
<option value="360">6 часов</option>
<option value="1440">24 часа (макс)</option>
</select></label>
<button type="submit">Выдать доступ</button>
</form>
<h2>Ожидают</h2><ul>%s</ul>
<h2>С доступом</h2><ul>%s</ul>
<h2>Join QR (покупатель)</h2>
<p><code>%s</code></p>
<p class="muted">SSID: %s · скрытая: %v</p>
</body></html>`, msg, pb.String(), ab.String(), html.EscapeString(join), html.EscapeString(gc.SSID), gc.Hidden)
}

func deskGrantQR(w http.ResponseWriter, r *http.Request, gc *guest.Config, store *guest.Store) {
	tok := r.URL.Query().Get("t")
	sess, ok := store.LookupToken(tok)
	if !ok {
		http.Error(w, "токен недействителен или истёк", http.StatusBadRequest)
		return
	}
	// Cookie session pin optional; require pin query/form once
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html><html lang="ru"><head><meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/><title>Grant</title></head><body>
<h1>Выдать интернет</h1>
<p>Устройство <code>%s</code></p>
<form method="POST">
<label>PIN сотрудника<br/><input type="password" name="pin" required/></label>
<input type="hidden" name="t" value="%s"/>
<p>Время: <b>10 минут</b> (по QR)</p>
<button type="submit">Подтвердить</button>
</form></body></html>`, html.EscapeString(sess.MAC), html.EscapeString(tok))
		return
	}
	_ = r.ParseForm()
	if !gc.PINOK(r.Form.Get("pin")) {
		http.Error(w, "неверный PIN", http.StatusForbidden)
		return
	}
	e, err := store.Grant(sess.MAC, guest.DefaultGrant, "qr")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = applyGuestFirewallAllow(store)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="ru"><head><meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/></head><body>
<h1 class="ok">Устройству предоставлен доступ в интернет</h1>
<p>До %s (10 минут). Можно закрыть страницу.</p>
</body></html>`, e.ExpiresAt.Local().Format("15:04:05"))
}

func deskAPIGrant(w http.ResponseWriter, r *http.Request, gc *guest.Config, store *guest.Store) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST", 405)
		return
	}
	_ = r.ParseForm()
	if !gc.PINOK(r.Form.Get("pin")) {
		http.Error(w, "pin", 403)
		return
	}
	mac := r.Form.Get("mac")
	mins := 10
	fmt.Sscanf(r.Form.Get("minutes"), "%d", &mins)
	e, err := store.Grant(mac, time.Duration(mins)*time.Minute, "api")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = applyGuestFirewallAllow(store)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok":true,"mac":%q,"expires_at":%q}`, e.MAC, e.ExpiresAt.Format(time.RFC3339))
}
