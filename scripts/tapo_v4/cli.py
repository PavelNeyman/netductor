#!/usr/bin/env python3
"""netductor TPAP helper — freeKC protocol (MIT). Usage:
  cli.py login HOST CLOUD_PASSWORD
  cli.py exec HOST CLOUD_PASSWORD METHOD [PARAMS_JSON]
  cli.py multi HOST CLOUD_PASSWORD '[{"method":"...","params":{}}]'
Env: TAPO_CRED_HASH=md5|sha256 (default try both)
"""
from __future__ import annotations
import json, os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from tapo_v4 import TapoV4, TapoV4Error

def client(host, password):
    order = []
    env = os.environ.get("TAPO_CRED_HASH", "").lower()
    if env in ("md5", "sha256"):
        order = [env]
    else:
        order = ["md5", "sha256"]
    last = None
    for h in order:
        try:
            c = TapoV4(host, password, username="admin", credential_hash=h)
            c.login()
            return c
        except Exception as e:
            last = e
    raise last

def main():
    if len(sys.argv) < 4:
        print(json.dumps({"ok": False, "error": "usage: login|exec|multi HOST PASS ..."}))
        sys.exit(2)
    cmd, host, password = sys.argv[1], sys.argv[2], sys.argv[3]
    try:
        c = client(host, password)
        if cmd == "login":
            print(json.dumps({"ok": True, "stok": c.stok, "seq": c.seq}))
            return
        if cmd == "exec":
            method = sys.argv[4] if len(sys.argv) > 4 else "getDeviceInfo"
            params = json.loads(sys.argv[5]) if len(sys.argv) > 5 else {}
            resps = c.multiple([{"method": method, "params": params}])
            print(json.dumps({"ok": True, "responses": resps}))
            return
        if cmd == "multi":
            reqs = json.loads(sys.argv[4])
            resps = c.multiple(reqs)
            print(json.dumps({"ok": True, "responses": resps}))
            return
        print(json.dumps({"ok": False, "error": "unknown cmd"}))
        sys.exit(2)
    except TapoV4Error as e:
        print(json.dumps({"ok": False, "error": str(e), "code": e.code}))
        sys.exit(1)
    except Exception as e:
        print(json.dumps({"ok": False, "error": str(e)}))
        sys.exit(1)

if __name__ == "__main__":
    main()
