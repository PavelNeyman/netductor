# qwen_tools_v3.py - unified toolkit
import urllib.request, urllib.error, json, base64, os, io, tarfile, time, re

TOKEN = os.environ.get("QWEN_GH_TOKEN", "")
REPO = "PavelNeyman/netductor"
API = "https://api.github.com/repos/" + REPO

def _req(method, path, payload=None):
    data = json.dumps(payload).encode() if payload is not None else None
    hdrs = {"Authorization": "Bearer " + TOKEN, "Accept": "application/vnd.github+json"}
    if data is not None: hdrs["Content-Type"] = "application/json"
    backoff = 1.0
    for attempt in range(6):
        try:
            r = urllib.request.Request(API + path, data=data, method=method, headers=hdrs)
            with urllib.request.urlopen(r, timeout=20) as resp:
                body = resp.read()
                return json.loads(body) if body else {}
        except urllib.error.HTTPError as e:
            if e.code in (403, 429) and attempt < 5:
                time.sleep(backoff); backoff *= 2; continue
            raise
        except (urllib.error.URLError, OSError):
            if attempt < 5: time.sleep(backoff); backoff *= 2; continue
            raise

def encode(text): return base64.b64encode(text.encode("utf-8")).decode("ascii")
def decode(b64): return base64.b64decode(b64).decode("utf-8")

def get_sha(path): return _req("GET", "/contents/" + path)["sha"]
def get_file(path): return decode(_req("GET", "/contents/" + path)["content"])
def put_file(path, text, msg):
    return _req("PUT", "/contents/" + path, {"message": msg, "content": encode(text)})
def update_file(path, text, msg):
    return _req("PUT", "/contents/" + path, {"message": msg, "content": encode(text), "sha": get_sha(path)})
def patch_file(path, old, new, msg):
    text = get_file(path)
    if old not in text: raise ValueError("pattern not found in " + path)
    return update_file(path, text.replace(old, new, 1), msg)
def append_file(path, text, msg): return update_file(path, get_file(path) + text, msg)
def delete_file(path, msg): return _req("DELETE", "/contents/" + path, {"message": msg, "sha": get_sha(path)})

def write_file(dst, data):
    for _ in range(6):
        fd = open(dst, "wb")
        if fd is not None: fd.write(data); fd.close(); return True
        time.sleep(0.05)
    return False

def fetch_source(src_dir="/home/workspace/src", patterns=(".go",)):
    os.makedirs(src_dir, exist_ok=True)
    url = "https://codeload.github.com/" + REPO + "/tar.gz/refs/heads/main"
    data = urllib.request.urlopen(url, timeout=20).read()
    tf = tarfile.open(fileobj=io.BytesIO(data), mode="r:gz")
    n = 0; skipped = []
    for m in tf.getmembers():
        if not m.isfile(): continue
        rel = m.name.split("/", 1)[1] if "/" in m.name else ""
        if rel.endswith(patterns):
            dst = os.path.join(src_dir, rel); d = os.path.dirname(dst)
            if d and not os.path.exists(d): os.makedirs(d, exist_ok=True)
            f = tf.extractfile(m); c = f.read(); f.close()
            if write_file(dst, c): n += 1
            else: skipped.append(rel)
    return n, skipped

_CACHE = {}
def code_example(path, start=None, end=None):
    if path not in _CACHE: _CACHE[path] = get_file(path).split("\n")
    lines = _CACHE[path]
    return "\n".join(lines) if start is None else "\n".join(lines[start-1:end])
def code_example_between(path, start_pat, end_pat):
    out = []; on = False
    for ln in code_example(path).split("\n"):
        if not on and re.search(start_pat, ln): on = True
        if on: out.append(ln)
        if on and re.search(end_pat, ln): break
    return "\n".join(out)

def fence(code, lang="go"):
    n = 3
    while ("`" * n) in code: n += 1
    f = "`" * n
    return f + lang + "\n" + code + "\n" + f
def add_code_block(doc, title, code, lang="go"):
    return doc + "\n\n### " + title + "\n\n" + fence(code, lang) + "\n"
def render_template(tpl, varsmap=None):
    out = tpl
    for k, v in (varsmap or {}).items():
        out = out.replace("{{" + k + "}}", v)
    out = re.sub(r"\{\{FILE:([^}]+)\}\}", lambda m: get_file(m.group(1)), out)
    out = re.sub(r"\{\{CODEFILE:([^:}]+)(?::([^}]+))?\}\}",
                 lambda m: fence(get_file(m.group(1)), m.group(2) or ""), out)
    return out
