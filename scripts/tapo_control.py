#!/usr/bin/env python3
"""Minimal Tapo C200 control via pytapo (same stack as HA Tapo-Control).

Requires: pip install pytapo
Tapo app: Me → Tapo Lab → Third-Party Compatibility → On
Camera Account: Advanced → Camera Account (not cloud login)

Usage:
  tapo_control.py <host> <user> <pass> move left|right|up|down [step]
  tapo_control.py <host> <user> <pass> night on|off|auto
  tapo_control.py <host> <user> <pass> privacy on|off
  tapo_control.py <host> <user> <pass> info
"""
from __future__ import annotations

import sys


def main() -> int:
    if len(sys.argv) < 5:
        print(__doc__.strip(), file=sys.stderr)
        return 2
    host, user, password, action = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
    try:
        from pytapo import Tapo
    except ImportError:
        print("error:pytapo not installed (pip install pytapo)", file=sys.stderr)
        return 1
    try:
        cam = Tapo(host, user, password)
    except Exception as e:
        print(f"error:login:{e}", file=sys.stderr)
        return 1
    try:
        if action == "info":
            print(cam.getBasicInfo())
            return 0
        if action == "move":
            if len(sys.argv) < 6:
                print("error:need direction", file=sys.stderr)
                return 2
            direction = sys.argv[5].lower()
            step = 10
            if len(sys.argv) >= 7:
                step = int(sys.argv[6])
            # same convention as Shinobi/pytapo examples for C200
            mapping = {
                "left": (-step, 0),
                "right": (step, 0),
                "up": (0, step),
                "down": (0, -step),
            }
            if direction not in mapping:
                print("error:dir left|right|up|down", file=sys.stderr)
                return 2
            x, y = mapping[direction]
            r = cam.moveMotor(x, y)
            print(r)
            return 0
        if action == "night":
            mode = sys.argv[5] if len(sys.argv) > 5 else "auto"
            r = cam.setDayNightMode(mode)
            print(r)
            return 0
        if action == "privacy":
            on = (sys.argv[5] if len(sys.argv) > 5 else "on").lower() in ("1", "on", "true")
            r = cam.setPrivacyMode(on)
            print(r)
            return 0
        print("error:action move|night|privacy|info", file=sys.stderr)
        return 2
    except Exception as e:
        print(f"error:{e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
