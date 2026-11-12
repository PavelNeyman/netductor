#!/usr/bin/env python3
"""Qwen tools for Netductor - workaround for sandbox limitations"""
import urllib.request, json, base64, os, io, tarfile

TOKEN = ""
REPO = "PavelNeyman/netductor"

def get_file(path):
    """Get file content from repo via API"""
    r = urllib.request.Request(
        f"https://api.github.com/repos/{REPO}/contents/{path}",
        headers={"Authorization": f"Bearer {TOKEN}", "Accept": "application/vnd.github+json"}
    )
    with urllib.request.urlopen(r, timeout=15) as resp:
        data = json.load(resp)
    return base64.b64decode(data['content']).decode('utf-8')

def put_file(path, content, message):
    """Create/update file in repo via API"""
    b64 = base64.b64encode(content.encode('utf-8')).decode('utf-8')
    payload = json.dumps({"message": message, "content": b64}).encode('utf-8')
    r = urllib.request.Request(
        f"https://api.github.com/repos/{REPO}/contents/{path}",
        data=payload, method='PUT',
        headers={"Authorization": f"Bearer {TOKEN}", "Content-Type": "application/json"}
    )
    with urllib.request.urlopen(r, timeout=15) as resp:
        return json.load(resp)

def fetch_source(src_dir="/home/workspace/src"):
    """Fetch source via tarball (works when git/subprocess blocked)"""
    os.makedirs(src_dir, exist_ok=True)
    url = f"https://codeload.github.com/{REPO}/tar.gz/refs/heads/main"
    data = urllib.request.urlopen(url, timeout=15).read()
    tf = tarfile.open(fileobj=io.BytesIO(data), mode="r:gz")
    n = 0
    for m in tf.getmembers():
        if m.isfile() and "/" in m.name:
            rel = m.name.split("/", 1)[1]
            if rel.endswith(".go"):
                dst = os.path.join(src_dir, rel)
                os.makedirs(os.path.dirname(dst), exist_ok=True)
                with open(dst, "wb") as f:
                    f.write(tf.extractfile(m).read())
                n += 1
    return n

if __name__ == "__main__":
    import sys
    if len(sys.argv) < 2:
        print("Usage: fetch | get <path> | put <path> <content> <message>")
        sys.exit(1)
    
    cmd = sys.argv[1]
    if cmd == "fetch":
        print(f"Fetched {fetch_source()} .go files")
    elif cmd == "get" and len(sys.argv) > 2:
        print(get_file(sys.argv[2]))
    elif cmd == "put" and len(sys.argv) > 4:
        result = put_file(sys.argv[2], sys.argv[3], sys.argv[4])
        print(f"Created: {result['content']['path']}")
