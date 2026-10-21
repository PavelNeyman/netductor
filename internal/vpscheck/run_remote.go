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
	Host           string
	User           string
	Password       string
	KeyPath        string
	KeyPassphrase  string
	Port           string // default 22
	Profile        Profile
	PrimaryIP      string
	SkipInstall    bool
}

// RunViaSSH uploads nothing: runs a self-contained remote shell that installs tools and
// prints JSON lines we parse — actually we run netductor if present, else inline Go is not available.
// Strategy: SSH a bash snippet that installs curl/iperf3/sysbench and runs lightweight checks,
// then Mac-side RunLocal is preferred when netductor is already on the node.
//
// For fresh VPS: we execute remote bash probes that mirror RunLocal key checks (no third-party pipe).
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
	// Prefer JSON block between markers
	start := strings.Index(out, "ND_VPSCHECK_JSON_BEGIN")
	end := strings.Index(out, "ND_VPSCHECK_JSON_END")
	if start >= 0 && end > start {
		raw := strings.TrimSpace(out[start+len("ND_VPSCHECK_JSON_BEGIN") : end])
		var r Report
		if json.Unmarshal([]byte(raw), &r) == nil {
			r.Host = o.Host
			r.tally()
			return r, nil
		}
	}
	// Fallback: wrap stdout as info
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
	// ash/bash compatible; no curl|bash third parties
	return fmt.Sprintf(`
set +e
export DEBIAN_FRONTEND=noninteractive
SKIP_INSTALL=%s
export PROFILE=%s
export PRIMARY_IP=%s
if [ "$SKIP_INSTALL" != "1" ]; then
  command -v apt-get >/dev/null 2>&1 && apt-get install -y -qq curl iperf3 sysbench 2>/dev/null
fi
# collect JSON via python3 if present, else minimal shell
python3 - <<'PY' 2>/dev/null || true
import json,socket,ssl,urllib.request,subprocess,time,os,re
from datetime import datetime,timezone
profile=os.environ.get("PROFILE","generic")
primary=os.environ.get("PRIMARY_IP","")
results=[]
def add(id,en,ru,st,detail,ms=0):
    results.append({"id":id,"title_en":en,"title_ru":ru,"status":st,"detail":detail,"duration_ms":ms})
# tools
add("tools","Install tools","Пакеты","ok","curl/iperf3/sysbench attempted")
# ip region
ip=region=""
try:
    t0=time.time()
    with urllib.request.urlopen("http://ip-api.com/json/?fields=status,country,countryCode,regionName,city,query,isp,org,as,proxy,hosting", timeout=12) as r:
        m=json.load(r)
    ms=int((time.time()-t0)*1000)
    if m.get("status")=="success":
        ip=m.get("query",""); region=f"{m.get('countryCode')}/{m.get('regionName')} {m.get('city')}"
        det=f"{region} isp={m.get('isp')} org={m.get('org')} as={m.get('as')} proxy={m.get('proxy')} hosting={m.get('hosting')}"
        st="warn" if m.get("proxy") else "info" if m.get("hosting") else "ok"
        add("ip_region","IP region","Регион IP",st,det,ms)
    else:
        add("ip_region","IP region","Регион IP","warn",str(m),ms)
except Exception as e:
    add("ip_region","IP region","Регион IP","warn",str(e))
# sysbench
try:
    t0=time.time()
    out=subprocess.check_output(["sysbench","cpu","run","--threads=1"],stderr=subprocess.STDOUT,timeout=120,text=True)
    ms=int((time.time()-t0)*1000)
    eps=[ln.strip() for ln in out.splitlines() if "events per second" in ln]
    add("sysbench_cpu","CPU sysbench","CPU sysbench","ok",eps[0] if eps else out[-200:],ms)
except Exception as e:
    add("sysbench_cpu","CPU sysbench","CPU sysbench","skip",str(e))
# dns
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
if primary:
    tcp(primary+":443","primary_443")
    tcp(primary+":52222","primary_ssh")
def http(url,id):
    t0=time.time()
    try:
        req=urllib.request.Request(url,headers={"User-Agent":"netductor-vpscheck/1"})
        with urllib.request.urlopen(req,timeout=12) as r:
            r.read(65536); code=r.status
        ms=int((time.time()-t0)*1000)
        st="ok" if 200<=code<400 else "warn"
        add("http_"+id,"HTTP "+id,"HTTP "+id,st,f"{code} {ms}ms",ms)
    except Exception as e:
        add("http_"+id,"HTTP "+id,"HTTP "+id,"warn",str(e))
if profile in ("secondary","generic"):
    http("https://ya.ru/","ya_ru"); http("https://vk.com/","vk")
if profile in ("primary","generic"):
    http("https://www.cloudflare.com/cdn-cgi/trace","cf_trace"); http("https://github.com/","github")
http("https://api.ipify.org","ipify")
# iperf optional
def iperf(hosts,id,en,ru):
    for h in hosts:
        try:
            t0=time.time()
            out=subprocess.check_output(["iperf3","-c",h,"-p","5201","-t","5","-J"],stderr=subprocess.STDOUT,timeout=30,text=True)
            ms=int((time.time()-t0)*1000)
            j=json.loads(out)
            bps=j.get("end",{}).get("sum_received",{}).get("bits_per_second") or j.get("end",{}).get("sum",{}).get("bits_per_second")
            if bps:
                add(id,en,ru,"ok",f"{h} {bps/1e6:.1f} Mbit/s",ms); return
            add(id,en,ru,"ok",h+" ok",ms); return
        except Exception as e:
            last=str(e)
    add(id,en,ru,"warn","no server: "+last if 'last' in dir() else "no server")
if profile=="secondary":
    iperf(["iperf.msk1.wnet.su","iperf.spb1.wnet.su"],"iperf_ru","iperf3 RU","iperf3 РФ")
elif profile=="primary":
    iperf(["iperf3.moji.online"],"iperf_pub","iperf3 public","iperf3 public")
ok=sum(1 for x in results if x["status"]=="ok"); warn=sum(1 for x in results if x["status"]=="warn"); fail=sum(1 for x in results if x["status"]=="fail")
rep={"at":datetime.now(timezone.utc).isoformat(),"profile":profile,"public_ip":ip,"region":region,"results":results,"ok":ok,"warn":warn,"fail":fail,
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
		// sshpass if available
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
