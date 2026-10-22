package vpscheck

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SSHOpts for running checks on a remote VPS from the operator Mac.
type SSHOpts struct {
	Host          string
	User          string
	Password      string
	KeyPath       string
	KeyPassphrase string
	Port          string // default 22
	Profile       Profile
	PrimaryIP     string
	SkipInstall   bool
}

// RunViaSSH runs scorecard on remote host.
// Prefer `netductor vps-check --json` if binary exists (day-2); else self-contained Python probe.
func RunViaSSH(ctx context.Context, o SSHOpts) (Report, error) {
	if o.User == "" {
		o.User = "root"
	}
	if o.Port == "" {
		o.Port = "22"
	}
	if o.Profile == "" {
		o.Profile = ProfileGeneric
	}
	script := remoteScript(o)
	out, err := sshRun(ctx, o, script)
	if err != nil {
		return Report{}, fmt.Errorf("ssh vps-check: %w\n%s", err, truncate(out, 800))
	}
	start := strings.Index(out, "ND_VPSCHECK_JSON_BEGIN")
	end := strings.Index(out, "ND_VPSCHECK_JSON_END")
	if start >= 0 && end > start {
		raw := strings.TrimSpace(out[start+len("ND_VPSCHECK_JSON_BEGIN") : end])
		var r Report
		if json.Unmarshal([]byte(raw), &r) == nil {
			r.Host = o.Host
			if r.Profile == "" {
				r.Profile = o.Profile
			}
			r.tally()
			return r, nil
		}
	}
	r := Report{At: time.Now().UTC(), Profile: o.Profile, Host: o.Host}
	r.Results = append(r.Results, Result{ID: "remote_raw", TitleEN: "Remote output", TitleRU: "Вывод", Status: StatusInfo, Detail: truncate(out, 2000)})
	r.tally()
	return r, nil
}

func remoteScript(o SSHOpts) string {
	prof := string(o.Profile)
	prim := o.PrimaryIP
	skip := "0"
	if o.SkipInstall {
		skip = "1"
	}
	return fmt.Sprintf(`
set +e
export DEBIAN_FRONTEND=noninteractive
SKIP_INSTALL=%s
export PROFILE=%s
export PRIMARY_IP=%s

# Day-2 path: full Go scorecard if netductor is installed
if command -v netductor >/dev/null 2>&1; then
  EXTRA=""
  [ -n "$PRIMARY_IP" ] && EXTRA="--primary $PRIMARY_IP"
  [ "$SKIP_INSTALL" = "1" ] && EXTRA="$EXTRA --skip-install"
  OUT=$(netductor vps-check --profile "$PROFILE" $EXTRA --json 2>/tmp/nd-vpscheck.err)
  if echo "$OUT" | grep -q '"results"'; then
    echo ND_VPSCHECK_JSON_BEGIN
    echo "$OUT"
    echo ND_VPSCHECK_JSON_END
    exit 0
  fi
fi

if [ "$SKIP_INSTALL" != "1" ]; then
  command -v apt-get >/dev/null 2>&1 && apt-get install -y -qq curl iperf3 sysbench dnsutils 2>/dev/null
fi

python3 - <<'PY'
import json,socket,ssl,urllib.request,subprocess,time,os
from datetime import datetime,timezone
profile=os.environ.get("PROFILE","generic")
primary=os.environ.get("PRIMARY_IP","")
results=[]
def add(id,en,ru,st,detail,ms=0):
    results.append({"id":id,"title_en":en,"title_ru":ru,"status":st,"detail":str(detail)[:500],"duration_ms":ms})

add("tools","Install tools","Пакеты","ok","curl/iperf3/sysbench/dnsutils attempted")

ip=region=""
try:
    t0=time.time()
    with urllib.request.urlopen("http://ip-api.com/json/?fields=status,country,countryCode,regionName,city,query,isp,org,as,proxy,hosting,mobile,reverse", timeout=12) as r:
        m=json.load(r)
    ms=int((time.time()-t0)*1000)
    if m.get("status")=="success":
        ip=m.get("query","") or ""
        region=f"{m.get('countryCode')}/{m.get('regionName')} {m.get('city')}"
        det=f"{region} isp={m.get('isp')} org={m.get('org')} as={m.get('as')} proxy={m.get('proxy')} hosting={m.get('hosting')}"
        st="fail" if m.get("proxy") else ("info" if m.get("hosting") else "ok")
        add("ip_region","IP region","Регион IP",st,det,ms)
        add("ipq_proxy","Proxy/VPN flags","Флаги proxy/VPN","fail" if m.get("proxy") else "info" if m.get("hosting") else "ok",
            f"proxy={m.get('proxy')} hosting={m.get('hosting')} mobile={m.get('mobile')}",ms)
        add("ipq_org","ASN / org","ASN / org","ok",f"as={m.get('as')} org={m.get('org')} isp={m.get('isp')} rdns={m.get('reverse')}",ms)
    else:
        add("ip_region","IP region","Регион IP","warn",str(m),ms)
except Exception as e:
    add("ip_region","IP region","Регион IP","warn",str(e))

# DNSBL (subset)
def rev_ip(ip):
    p=ip.split(".")
    if len(p)!=4: return None
    return ".".join(reversed(p))
def dnsbl(ip):
    rev=rev_ip(ip)
    if not rev: return "skip","not v4"
    hits=[]; checked=[]
    for zone,name in [("zen.spamhaus.org","zen"),("cbl.abuseat.org","cbl"),("b.barracudacentral.org","barracuda")]:
        checked.append(name)
        try:
            socket.getaddrinfo(f"{rev}.{zone}",None)
            # if resolves to something - may be listed (simplified: any answer = listed for spamhaus style)
            hits.append(name)
        except socket.gaierror:
            pass
    if hits:
        return "fail","LISTED: "+",".join(hits)
    return "ok","clean on "+",".join(checked)
if ip:
    st,det=dnsbl(ip)
    add("ipq_dnsbl","DNSBL","DNSBL",st,det)

try:
    t0=time.time()
    out=subprocess.check_output(["sysbench","cpu","run","--threads=1"],stderr=subprocess.STDOUT,timeout=120,text=True)
    ms=int((time.time()-t0)*1000)
    eps=[ln.strip() for ln in out.splitlines() if "events per second" in ln]
    add("sysbench_cpu","CPU sysbench","CPU sysbench","ok",eps[0] if eps else out[-200:],ms)
except Exception as e:
    add("sysbench_cpu","CPU sysbench","CPU sysbench","skip",str(e))

try:
    t0=time.time()
    out=subprocess.check_output(["sysbench","memory","run","--threads=1"],stderr=subprocess.STDOUT,timeout=120,text=True)
    ms=int((time.time()-t0)*1000)
    line=next((ln.strip() for ln in out.splitlines() if "transferred" in ln), out[-150:])
    add("sysbench_mem","Memory sysbench","Память sysbench","ok",line,ms)
except Exception as e:
    add("sysbench_mem","Memory sysbench","Память sysbench","skip",str(e))

try:
    t0=time.time(); socket.getaddrinfo("example.com",80); ms=int((time.time()-t0)*1000)
    add("dns","DNS resolve","DNS","ok",f"example.com {ms}ms",ms)
except Exception as e:
    add("dns","DNS resolve","DNS","fail",str(e))

def tcp(addr,id):
    host,port=addr.rsplit(":",1)
    t0=time.time()
    try:
        s=socket.create_connection((host,int(port)),timeout=5); s.close()
        ms=int((time.time()-t0)*1000)
        add("latency_"+id,"TCP "+addr,"TCP "+addr,"ok",f"{ms}ms",ms)
    except Exception as e:
        ms=int((time.time()-t0)*1000)
        add("latency_"+id,"TCP "+addr,"TCP "+addr,"warn",str(e),ms)
tcp("1.1.1.1:443","cloudflare")
tcp("8.8.8.8:443","google_dns")
if primary:
    tcp(primary+":443","primary_443")
    tcp(primary+":52222","primary_ssh")

def http(url,id,en=None,ru=None):
    en=en or ("HTTP "+id); ru=ru or en
    t0=time.time()
    try:
        req=urllib.request.Request(url,headers={"User-Agent":"Mozilla/5.0 netductor-vpscheck/1"})
        with urllib.request.urlopen(req,timeout=15) as r:
            r.read(65536); code=getattr(r,"status",200)
        ms=int((time.time()-t0)*1000)
        st="ok" if 200<=code<400 else ("warn" if code in (403,451) else "warn")
        add(id,en,ru,st,f"{code} {ms}ms",ms)
    except Exception as e:
        add(id,en,ru,"warn",str(e))

# geoblock matrix
for uid,en,ru,url in [
    ("geo_youtube","YouTube","YouTube","https://www.youtube.com/"),
    ("geo_twitter","X/Twitter","X/Twitter","https://x.com/"),
    ("geo_discord","Discord","Discord","https://discord.com/"),
    ("geo_github","GitHub","GitHub","https://github.com/"),
    ("geo_yandex","Yandex","Яндекс","https://ya.ru/"),
    ("geo_vk","VK","VK","https://vk.com/"),
]:
    http(url,uid,en,ru)

# DPI TLS SNI
def tls_sni(addr,sni,id):
    host,port=addr.rsplit(":",1)
    t0=time.time()
    try:
        raw=socket.create_connection((host,int(port)),timeout=8)
        ctx=ssl.create_default_context(); ctx.check_hostname=False; ctx.verify_mode=ssl.CERT_NONE
        ss=ctx.wrap_socket(raw,server_hostname=sni)
        ms=int((time.time()-t0)*1000)
        add(id,"TLS SNI "+sni,"TLS SNI "+sni,"ok",f"ok {ms}ms",ms)
        ss.close()
    except Exception as e:
        ms=int((time.time()-t0)*1000)
        add(id,"TLS SNI "+sni,"TLS SNI "+sni,"warn",str(e),ms)
tls_sni("api.vk.me:443","api.vk.me","dpi_tls_api_vk_me")
tls_sni("www.cloudflare.com:443","www.cloudflare.com","dpi_tls_cloudflare")
tls_sni("www.google.com:443","www.google.com","dpi_tls_google")

# DNS pollution
for name in ["www.google.com","twitter.com","youtube.com"]:
    t0=time.time()
    try:
        ips=socket.getaddrinfo(name,443,socket.AF_INET)
        addrs=[x[4][0] for x in ips]
        ms=int((time.time()-t0)*1000)
        bad=any(a.startswith("127.") or a.startswith("0.") or a.startswith("10.") for a in addrs)
        add("dpi_dns_"+name.replace(".","_"),"DNS "+name,"DNS "+name,"fail" if bad else "ok",",".join(addrs),ms)
    except Exception as e:
        add("dpi_dns_"+name.replace(".","_"),"DNS "+name,"DNS "+name,"warn",str(e))

def iperf(hosts,id,en,ru):
    last="none"
    for h in hosts:
        try:
            t0=time.time()
            out=subprocess.check_output(["iperf3","-c",h,"-p","5201","-t","5","-J"],stderr=subprocess.STDOUT,timeout=30,text=True)
            ms=int((time.time()-t0)*1000)
            j=json.loads(out)
            bps=(j.get("end",{}) or {}).get("sum_received",{}) or {}
            bps=bps.get("bits_per_second") or (j.get("end",{}) or {}).get("sum",{}).get("bits_per_second")
            if bps:
                add(id,en,ru,"ok",f"{h} {bps/1e6:.1f} Mbit/s",ms); return
            add(id,en,ru,"ok",h+" ok",ms); return
        except Exception as e:
            last=str(e)
    add(id,en,ru,"warn","no server: "+last)

if profile=="secondary":
    iperf(["iperf.msk1.wnet.su","iperf.spb1.wnet.su"],"iperf_ru","iperf3 RU","iperf3 РФ")
elif profile=="primary":
    iperf(["iperf3.moji.online","bouygues.iperf.fr"],"iperf_pub","iperf3 public","iperf3 public")
else:
    iperf(["iperf3.moji.online"],"iperf_pub","iperf3 public","iperf3 public")

http("https://api.ipify.org","http_ipify")
ok=sum(1 for x in results if x["status"]=="ok"); warn=sum(1 for x in results if x["status"]=="warn"); fail=sum(1 for x in results if x["status"]=="fail")
rep={"at":datetime.now(timezone.utc).isoformat().replace("+00:00","Z"),"profile":profile,"public_ip":ip,"region":region,"results":results,"ok":ok,"warn":warn,"fail":fail,
 "summary_en":"FAIL" if fail else ("WARN" if warn else "OK"),"summary_ru":"FAIL" if fail else ("WARN" if warn else "OK")}
print("ND_VPSCHECK_JSON_BEGIN")
print(json.dumps(rep,ensure_ascii=False))
print("ND_VPSCHECK_JSON_END")
PY
true
`, skip, shellQuote(prof), shellQuote(prim))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func sshRun(ctx context.Context, o SSHOpts, script string) (string, error) {
	args := []string{"-p", o.Port, "-o", "StrictHostKeyChecking=accept-new", "-o", "ConnectTimeout=15"}
	if o.KeyPath != "" {
		args = append(args, "-i", o.KeyPath, "-o", "IdentitiesOnly=yes")
	}
	if o.Password != "" {
		if _, err := exec.LookPath("sshpass"); err == nil {
			full := append([]string{"-p", o.Password, "ssh"}, append(args, o.User+"@"+o.Host, "bash", "-s")...)
			cmd := exec.CommandContext(ctx, "sshpass", full...)
			cmd.Stdin = strings.NewReader(script)
			b, err := cmd.CombinedOutput()
			return string(b), err
		}
	}
	args = append(args, o.User+"@"+o.Host, "bash", "-s")
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = strings.NewReader(script)
	if o.KeyPassphrase != "" {
		cmd.Env = append(os.Environ(), "SSH_ASKPASS_REQUIRE=force")
	}
	b, err := cmd.CombinedOutput()
	return string(b), err
}
