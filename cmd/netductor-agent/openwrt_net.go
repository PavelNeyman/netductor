package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func dhcpLeasesJSON() string {
	b, err := os.ReadFile("/tmp/dhcp.leases")
	if err != nil {
		// try dnsmasq alternate
		b, err = os.ReadFile("/var/dhcp.leases")
		if err != nil {
			return "error:" + err.Error()
		}
	}
	type lease struct {
		Expiry   int64  `json:"expiry"`
		MAC      string `json:"mac"`
		IP       string `json:"ip"`
		Hostname string `json:"hostname,omitempty"`
		ClientID string `json:"client_id,omitempty"`
	}
	var out []lease
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		exp, _ := strconv.ParseInt(f[0], 10, 64)
		l := lease{Expiry: exp, MAC: strings.ToLower(strings.ReplaceAll(f[1], "-", ":")), IP: f[2]}
		if len(f) >= 4 && f[3] != "*" {
			l.Hostname = f[3]
		}
		if len(f) >= 5 && f[4] != "*" {
			l.ClientID = f[4]
		}
		out = append(out, l)
	}
	raw, _ := json.Marshal(map[string]any{"leases": out, "count": len(out)})
	return string(raw)
}

func wifiClientsJSON() string {
	// Best-effort: iwinfo <if> assoclist for each radio iface
	ifaces := []string{}
	if out, err := exec.Command("iwinfo").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// "wlan0     ESSID: ..."
			parts := strings.Fields(line)
			if len(parts) > 0 && !strings.Contains(parts[0], ":") {
				ifaces = append(ifaces, parts[0])
			}
		}
	}
	type client struct {
		Iface string `json:"iface"`
		MAC   string `json:"mac"`
		Raw   string `json:"raw,omitempty"`
	}
	var clients []client
	seen := map[string]bool{}
	for _, ifc := range ifaces {
		if seen[ifc] {
			continue
		}
		seen[ifc] = true
		out, err := exec.Command("iwinfo", ifc, "assoclist").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			mac := strings.ToLower(strings.ReplaceAll(fields[0], "-", ":"))
			if strings.Count(mac, ":") != 5 {
				continue
			}
			clients = append(clients, client{Iface: ifc, MAC: mac, Raw: truncate(line, 160)})
		}
	}
	raw, _ := json.Marshal(map[string]any{"clients": clients, "count": len(clients)})
	return string(raw)
}

// dhcpStaticHost arg: mac=aa:bb:..|ip=192.168.1.50|name=tapo1
func dhcpStaticHost(arg string) string {
	if strings.ContainsAny(arg, "\r\n") {
		return "error:invalid arg"
	}
	kv := map[string]string{}
	for _, p := range strings.Split(arg, "|") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		i := strings.IndexByte(p, '=')
		if i <= 0 {
			continue
		}
		kv[strings.ToLower(p[:i])] = p[i+1:]
	}
	mac, err := validMAC(kv["mac"])
	if err != nil {
		return "error:invalid mac"
	}
	ip, err := validIPv4(kv["ip"])
	if err != nil {
		return "error:invalid ip"
	}
	name := kv["name"]
	if name == "" {
		hex := strings.ReplaceAll(mac, ":", "")
		if len(hex) >= 4 {
			name = "cam-" + hex[len(hex)-4:]
		} else {
			name = "cam-host"
		}
	}
	name, err = validDHCPName(name)
	if err != nil {
		return "error:invalid name"
	}
	show, _ := exec.Command("uci", "-q", "show", "dhcp").CombinedOutput()
	section := ""
	for _, line := range strings.Split(string(show), "\n") {
		if strings.Contains(line, ".mac=") && strings.Contains(strings.ToLower(line), mac) {
			left := strings.SplitN(line, ".mac=", 2)[0]
			section = left
			break
		}
	}
	var lines []string
	if section == "" {
		lines = append(lines, "add dhcp host")
		section = "dhcp.@host[-1]"
	}
	lines = append(lines,
		"set "+section+".mac="+mac,
		"set "+section+".ip="+ip,
		"set "+section+".name="+name,
		"set "+section+".dns=1",
		"commit dhcp",
	)
	res := uciBatch(strings.Join(lines, "\n"))
	out, _ := exec.Command("/etc/init.d/dnsmasq", "reload").CombinedOutput()
	return res + "; dnsmasq:" + truncate(string(out), 120)
}

func uciBatch(arg string) string {
	var ok, fail int
	var logs []string
	for _, line := range strings.Split(arg, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "network_reload" {
			out, err := exec.Command("/etc/init.d/network", "reload").CombinedOutput()
			if err != nil {
				logs = append(logs, "fail:network_reload:"+truncate(string(out), 200))
				fail++
			} else {
				logs = append(logs, "network_reload:"+truncate(string(out), 200))
				ok++
			}
			continue
		}
		if line == "wifi_reload" {
			out, err := exec.Command("wifi", "reload").CombinedOutput()
			if err != nil {
				logs = append(logs, "fail:wifi_reload:"+truncate(string(out), 200))
				fail++
			} else {
				logs = append(logs, "wifi_reload:"+truncate(string(out), 200))
				ok++
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		op := fields[0]
		var out []byte
		var err error
		switch op {
		case "set":
			// "set network.lan.ipaddr=192.168.1.1" or path without spaces after set
			rest := strings.TrimSpace(strings.TrimPrefix(line, "set"))
			if rest == "" || !strings.Contains(rest, "=") {
				logs = append(logs, "bad:"+line)
				fail++
				continue
			}
			out, err = exec.Command("uci", "set", rest).CombinedOutput()
		case "delete", "del":
			rest := strings.TrimSpace(line[len(fields[0]):])
			if rest == "" {
				logs = append(logs, "bad:"+line)
				fail++
				continue
			}
			out, err = exec.Command("uci", "delete", rest).CombinedOutput()
		case "add":
			// add <config> <type>
			if len(fields) < 3 {
				logs = append(logs, "bad:"+line)
				fail++
				continue
			}
			out, err = exec.Command("uci", "add", fields[1], fields[2]).CombinedOutput()
		case "commit":
			if len(fields) == 1 {
				out, err = exec.Command("uci", "commit").CombinedOutput()
			} else {
				out, err = exec.Command("uci", "commit", fields[1]).CombinedOutput()
			}
		default:
			// Legacy: bare path=value → uci set
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				logs = append(logs, "bad:"+line)
				fail++
				continue
			}
			out, err = exec.Command("uci", "set", parts[0]+"="+parts[1]).CombinedOutput()
		}
		if err != nil {
			logs = append(logs, "fail:"+op+":"+truncate(string(out), 100))
			fail++
		} else {
			ok++
		}
	}
	return fmt.Sprintf("ok=%d fail=%d\n%s", ok, fail, strings.Join(logs, "\n"))
}

// uciBatchFailed reports whether a uciBatch result string recorded any failures.
func uciBatchFailed(res string) bool {
	for _, line := range strings.Split(res, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ok=") {
			continue
		}
		// ok=N fail=M
		parts := strings.Fields(line)
		for _, p := range parts {
			if strings.HasPrefix(p, "fail=") {
				n := strings.TrimPrefix(p, "fail=")
				return n != "" && n != "0"
			}
		}
	}
	return strings.Contains(res, "\nfail:") || strings.HasPrefix(res, "fail:")
}
