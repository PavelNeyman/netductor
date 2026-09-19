#!/usr/bin/env python3
"""Tapo C200 control via pytapo (parity helper for environments with Python).

Prefer native Go: internal/tapo (agent camera_ptz).

  tapo_control.py <host> <user> <pass> move left|right|up|down [step]
  tapo_control.py <host> <user> <pass> night on|off|auto
  tapo_control.py <host> <user> <pass> privacy on|off
  tapo_control.py <host> <user> <pass> info
  tapo_control.py <host> <user> <pass> presets
  tapo_control.py <host> <user> <pass> preset_goto <id>
  tapo_control.py <host> <user> <pass> calibrate
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
        cam = Tapo(host, user, password)
    except Exception as e:
        print(f"error:{e}", file=sys.stderr)
        return 1
    try:
        if action == "info":
            print(cam.getBasicInfo()); return 0
        if action == "presets":
            print(cam.getPresets()); return 0
        if action == "calibrate":
            print(cam.calibrateMotor()); return 0
        if action == "preset_goto" and len(sys.argv) > 5:
            print(cam.setPreset(sys.argv[5])); return 0
        if action == "move" and len(sys.argv) > 5:
            d = sys.argv[5].lower(); step = int(sys.argv[6]) if len(sys.argv) > 6 else 10
            m = {"left": (-step, 0), "right": (step, 0), "up": (0, step), "down": (0, -step)}
            print(cam.moveMotor(*m[d])); return 0
        if action == "night" and len(sys.argv) > 5:
            print(cam.setDayNightMode(sys.argv[5])); return 0
        if action == "privacy" and len(sys.argv) > 5:
            print(cam.setPrivacyMode(sys.argv[5].lower() in ("on","1","true"))); return 0
        print("error:action", file=sys.stderr); return 2
    except Exception as e:
        print(f"error:{e}", file=sys.stderr); return 1

if __name__ == "__main__":
    raise SystemExit(main())
