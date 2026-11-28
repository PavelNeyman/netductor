package deploy

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// EdgeStorageOpts controls USB/extroot/pin steps (PLAN-EDGE-USB-STORAGE).
type EdgeStorageOpts struct {
	UseUSB               bool
	Format               bool // destructive; requires UseUSB
	Extroot              bool // pivot /overlay to USB
	PinFallback          bool // install sync script to pin internal
	ExpectUSB            bool // write flag for agent alerts
	DeviceID             string
	RouterUser           string
	RouterHost           string
	RouterPass           string
	PrimaryKey           string
	PrimaryKeyPassphrase string
}

// applyEdgeStorage runs mount/format/extroot/pin after thin stack, before vpn_client when possible.
func applyEdgeStorage(o EdgeOpts, s EdgeStorageOpts) (needReboot bool, err error) {
	if !s.UseUSB && !s.Extroot && !s.Format {
		return false, nil
	}
	pass := o.RouterPass
	user := o.RouterUser
	host := o.RouterHost
	port := factorySSHPort()
	key := o.PrimaryKey
	kp := o.PrimaryKeyPassphrase

	if s.ExpectUSB || s.Extroot {
		_, _ = runSSHOnPort(port, pass, key, user, host,
			"mkdir -p /etc/netductor-agent; echo 1 > /etc/netductor-agent/expect_usb", kp)
	}

	out, err := runSSHOnPort(port, pass, key, user, host, `
set +e
# Prefer removable (USB) disks only — never first sd* (may be system SATA on x86).
for d in /sys/block/sd*; do
  [ -d "$d" ] || continue
  rem=$(cat "$d/removable" 2>/dev/null || echo 0)
  [ "$rem" = "1" ] || continue
  echo DEV=$(basename "$d")
  echo REMOVABLE=1
  break
done
`, kp)
	if err != nil {
		return false, fmt.Errorf("storage probe: %w (%s)", err, out)
	}
	dev := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DEV=") {
			dev = strings.TrimPrefix(line, "DEV=")
		}
	}
	if dev == "" {
		return false, fmt.Errorf("storage: no USB disk found (sd*)")
	}
	part := "/dev/" + dev + "1"
	o.logLine(fmt.Sprintf("==> storage: disk=%s part=%s format=%v extroot=%v", dev, part, s.Format, s.Extroot))

	if s.Format {
		o.logLine("==> storage: FORMAT USB (destructive)")
		// Long mkfs on 4–64GB sticks can break Dropbear/SSH mid-write ("Broken pipe").
		// Detach mkfs from the SSH session (setsid), then poll with short SSH sessions.
		start := fmt.Sprintf(`
set +e
DISK=/dev/%s
umount "$DISK" 2>/dev/null || true
umount ${DISK}1 2>/dev/null || true
umount ${DISK}2 2>/dev/null || true
PART=""
if command -v parted >/dev/null 2>&1; then
  parted -s $DISK -- mklabel gpt mkpart primary ext4 1MiB 100%%
  sleep 2
  PART=${DISK}1
elif command -v fdisk >/dev/null 2>&1; then
  echo -e 'o\nn\np\n1\n\n\nw\n' | fdisk $DISK
  sleep 2
  PART=${DISK}1
else
  echo "FORMAT_WHOLE_DISK no parted/fdisk"
  PART=$DISK
fi
if [ -n "$PART" ] && [ "$PART" != "$DISK" ] && [ ! -b "$PART" ]; then
  echo "FORMAT_PART_MISSING $PART — falling back to whole disk"
  PART=$DISK
fi
[ -b "$PART" ] || { echo "FORMAT_NO_BLOCK part=$PART"; ls -l $DISK* 2>/dev/null; exit 1; }
MKFS=""
for c in /usr/sbin/mkfs.ext4 /sbin/mkfs.ext4; do
  [ -x "$c" ] && MKFS=$c && break
done
[ -n "$MKFS" ] || MKFS=$(command -v mkfs.ext4 2>/dev/null || true)
[ -n "$MKFS" ] || { echo "FORMAT_NO_MKFS need e2fsprogs (edge.bootstrap preset=usb-storage)"; ls /usr/sbin/mkfs* 2>/dev/null; exit 1; }
echo "FORMAT_MKFS=$MKFS"
rm -f /tmp/nd-mkfs.rc /tmp/nd-mkfs.log /tmp/nd-mkfs.part /tmp/nd-mkfs.before_uuid /tmp/nd-mkfs.pid /tmp/nd-mkfs.started
echo "$PART" > /tmp/nd-mkfs.part
echo "$MKFS" > /tmp/nd-mkfs.bin
block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1 > /tmp/nd-mkfs.before_uuid || true
# Dropbear SIGHUPs children when the SSH shell exits; plain nohup is not enough on OpenWrt.
START_SH=/tmp/nd-mkfs-run.sh
cat > "$START_SH" << 'EOS'
#!/bin/sh
trap '' HUP INT TERM
PART=$(cat /tmp/nd-mkfs.part)
MKFS=$(cat /tmp/nd-mkfs.bin 2>/dev/null || echo mkfs.ext4)
echo "start part=$PART mkfs=$MKFS" > /tmp/nd-mkfs.started
"$MKFS" -F -L netductor "$PART" > /tmp/nd-mkfs.log 2>&1
echo $? > /tmp/nd-mkfs.rc
EOS
chmod 755 "$START_SH"
if command -v setsid >/dev/null 2>&1; then
  setsid "$START_SH" </dev/null >/dev/null 2>&1 &
else
  ( trap '' HUP; "$START_SH" </dev/null >/dev/null 2>&1 ) &
fi
echo $! > /tmp/nd-mkfs.pid
sleep 1
if [ -f /tmp/nd-mkfs.rc ]; then
  echo "FORMAT_RC_EARLY=$(cat /tmp/nd-mkfs.rc)"
  tail -5 /tmp/nd-mkfs.log 2>/dev/null || true
elif [ -f /tmp/nd-mkfs.started ]; then
  echo "FORMAT_BG_OK pid=$(cat /tmp/nd-mkfs.pid 2>/dev/null)"
else
  echo "FORMAT_BG_FAIL trying foreground mkfs"
  "$MKFS" -F -L netductor "$PART" > /tmp/nd-mkfs.log 2>&1
  echo $? > /tmp/nd-mkfs.rc
  echo "FORMAT_RC_FG=$(cat /tmp/nd-mkfs.rc)"
  tail -8 /tmp/nd-mkfs.log 2>/dev/null || true
fi
echo FORMAT_STARTED part=$PART
BEFORE_UUID=$(cat /tmp/nd-mkfs.before_uuid 2>/dev/null || true)
echo FORMAT_BEFORE_UUID=$BEFORE_UUID
`, dev)
		out, err := runSSHOnPortLogged(port, pass, key, user, host, start, kp)
		o.logLine(strings.TrimSpace(stripSSHNoise(out)))
		if strings.Contains(out, "FORMAT_NO_MKFS") {
			return false, fmt.Errorf("storage format: mkfs.ext4 missing on router — run edge.bootstrap preset=usb-storage (%s)", stripSSHNoise(out))
		}
		if strings.Contains(out, "FORMAT_NO_BLOCK") {
			return false, fmt.Errorf("storage format: no block device (%s)", stripSSHNoise(out))
		}
		started := strings.Contains(out, "FORMAT_STARTED")
		if err != nil && !started {
			return false, fmt.Errorf("storage format start: %w (%s)", err, stripSSHNoise(out))
		}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "FORMAT_STARTED part=") {
				part = strings.TrimPrefix(line, "FORMAT_STARTED part=")
			}
		}
		if strings.Contains(out, "FORMAT_RC_EARLY=") || strings.Contains(out, "FORMAT_RC_FG=") {
			rcLine := "1"
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "FORMAT_RC_EARLY=") {
					rcLine = strings.TrimPrefix(line, "FORMAT_RC_EARLY=")
				}
				if strings.HasPrefix(line, "FORMAT_RC_FG=") {
					rcLine = strings.TrimPrefix(line, "FORMAT_RC_FG=")
				}
			}
			if rcLine != "0" {
				return false, fmt.Errorf("storage format mkfs rc=%s (%s)", rcLine, stripSSHNoise(out))
			}
			o.logLine("==> storage: FORMAT_OK part=" + part)
		} else {
			var done bool
			for i := 0; i < 180; i++ {
				poll := fmt.Sprintf(`
PART=%s
[ -f /tmp/nd-mkfs.part ] && PART=$(cat /tmp/nd-mkfs.part)
BEFORE=$(cat /tmp/nd-mkfs.before_uuid 2>/dev/null || true)
if [ -f /tmp/nd-mkfs.rc ]; then
  echo FORMAT_RC=$(cat /tmp/nd-mkfs.rc)
  echo FORMAT_PART=$PART
  tail -12 /tmp/nd-mkfs.log 2>/dev/null || true
  exit 0
fi
MKFS_ALIVE=0
if [ -f /tmp/nd-mkfs.pid ]; then
  pid=$(cat /tmp/nd-mkfs.pid 2>/dev/null)
  [ -n "$pid" ] && [ -d /proc/$pid ] && MKFS_ALIVE=1
fi
ps w 2>/dev/null | grep -F mkfs.ext4 | grep -v grep >/dev/null && MKFS_ALIVE=1
if [ "$MKFS_ALIVE" = "0" ]; then
  CUR=$(block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1)
  if block info "$PART" 2>/dev/null | grep -q 'TYPE="ext4"'; then
    if [ -z "$BEFORE" ] || [ -n "$CUR" -a "$CUR" != "$BEFORE" ]; then
      echo FORMAT_RC=0
      echo FORMAT_PART=$PART
      echo FORMAT_INFER_EXT4_UUID_CHANGED before=$BEFORE after=$CUR
      exit 0
    fi
  fi
  if [ %d -ge 3 ]; then
    echo FORMAT_DEAD_NO_RC
    echo FORMAT_PART=$PART
    echo "--- log ---"; cat /tmp/nd-mkfs.log 2>/dev/null || echo "(no log)"
    echo "--- started ---"; cat /tmp/nd-mkfs.started 2>/dev/null || echo "(no started marker)"
    echo "--- bin ---"; cat /tmp/nd-mkfs.bin 2>/dev/null; ls -l /usr/sbin/mkfs.ext4 /sbin/mkfs.ext4 2>/dev/null
    exit 0
  fi
fi
echo FORMAT_WAIT n=%d part=$PART mkfs_alive=$MKFS_ALIVE before=$BEFORE pid=$(cat /tmp/nd-mkfs.pid 2>/dev/null)
`, part, i, i)
				out, err = runSSHOnPortLogged(port, pass, key, user, host, poll, kp)
				text := stripSSHNoise(out)
				if strings.TrimSpace(text) != "" {
					o.logLine(strings.TrimSpace(text))
				}
				if strings.Contains(out, "FORMAT_DEAD_NO_RC") {
					return false, fmt.Errorf("storage format: mkfs never completed (SSH session killed background job or missing e2fsprogs): %s", text)
				}
				if strings.Contains(out, "FORMAT_RC=") {
					rcLine := ""
					for _, line := range strings.Split(out, "\n") {
						line = strings.TrimSpace(line)
						if strings.HasPrefix(line, "FORMAT_RC=") {
							rcLine = strings.TrimPrefix(line, "FORMAT_RC=")
						}
						if strings.HasPrefix(line, "FORMAT_PART=") {
							part = strings.TrimPrefix(line, "FORMAT_PART=")
						}
					}
					if rcLine == "0" {
						o.logLine("==> storage: FORMAT_OK part=" + part)
						done = true
						break
					}
					return false, fmt.Errorf("storage format mkfs rc=%s (%s)", rcLine, text)
				}
				if err != nil {
					o.logLine(fmt.Sprintf("==> storage: format poll SSH blip (will retry): %v", err))
				}
				select {
				case <-time.After(5 * time.Second):
				case <-currentSSHCtx().Done():
					return false, fmt.Errorf("storage format: cancelled")
				}
			}
			if !done {
				return false, fmt.Errorf("storage format: timeout waiting for mkfs on %s", part)
			}
		}
	}

	if s.PinFallback {
		if err := installPinSync(port, pass, key, user, host, kp); err != nil {
			return false, fmt.Errorf("storage pin-sync: %w", err)
		}
		o.logLine("==> storage: pin sync installed before extroot")
	}
	if s.Extroot {
		o.logLine("==> storage: configure extroot")
		script := fmt.Sprintf(`
set +e
PART=%s
DISK=/dev/%s
echo "EXTROOT_BEGIN part=$PART disk=$DISK"
# Prefer the node that already has our ext4 (whole-disk vs partition).
if [ -b "$PART" ] && block info "$PART" 2>/dev/null | grep -q 'TYPE="ext4"'; then
  :
elif [ -b "$DISK" ] && block info "$DISK" 2>/dev/null | grep -q 'TYPE="ext4"'; then
  PART=$DISK
  echo "EXTROOT_PART_RESOLVED $PART (whole disk)"
elif [ -b "${DISK}1" ] && block info "${DISK}1" 2>/dev/null | grep -q 'TYPE="ext4"'; then
  PART=${DISK}1
  echo "EXTROOT_PART_RESOLVED $PART"
elif [ -b "$PART" ]; then
  :
elif [ -b "$DISK" ]; then
  PART=$DISK
else
  echo "EXTROOT_NO_PART"; ls -l $DISK* 2>/dev/null; exit 1
fi
mkdir -p /mnt/netductor
umount /mnt/netductor 2>/dev/null || true
if ! mount "$PART" /mnt/netductor; then
  echo "EXTROOT_MOUNT_FAIL part=$PART"
  dmesg 2>/dev/null | tail -8
  exit 1
fi
echo "EXTROOT_MOUNTED $(df -h /mnt/netductor | tail -1)"

# Copy current overlay onto USB if not already staged.
if [ ! -f /mnt/netductor/.nd-extroot ]; then
  echo "EXTROOT_COPY_START"
  if cp -a /overlay/. /mnt/netductor/ 2>/tmp/nd-extroot-cp.err; then
    echo "EXTROOT_COPY_CP_OK"
  elif tar -C /overlay -cf - . 2>/dev/null | tar -C /mnt/netductor -xf - 2>/tmp/nd-extroot-tar.err; then
    echo "EXTROOT_COPY_TAR_OK"
  else
    echo "EXTROOT_COPY_FAIL"
    cat /tmp/nd-extroot-cp.err /tmp/nd-extroot-tar.err 2>/dev/null | tail -20
    umount /mnt/netductor 2>/dev/null || true
    exit 1
  fi
  if [ ! -d /mnt/netductor/upper/etc ] && [ ! -d /mnt/netductor/etc ]; then
    n=$(find /mnt/netductor -mindepth 1 -maxdepth 2 ! -name lost+found 2>/dev/null | wc -l)
    echo "EXTROOT_ENTRIES=$n"
    if [ "$n" -lt 3 ]; then
      echo "EXTROOT_EMPTY_AFTER_COPY"
      umount /mnt/netductor 2>/dev/null || true
      exit 1
    fi
  fi
  touch /mnt/netductor/.nd-extroot
fi

# UUID: block info only (OpenWrt 24+/apk often has no blkid binary).
UUID=$(block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1)
[ -n "$UUID" ] || UUID=$(block info | grep -F "$PART" | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1)
[ -n "$UUID" ] || UUID=$(tune2fs -l "$PART" 2>/dev/null | awk -F: '/Filesystem UUID/{gsub(/ /,"",$2); print $2; exit}')
[ -n "$UUID" ] || { echo "EXTROOT_NO_UUID part=$PART"; block info 2>&1 | head -20; exit 1; }
echo "EXTROOT_UUID=$UUID part=$PART"

# Never apk/opkg here: without WAN, apk can hang for many minutes and CombinedOutput
# shows nothing until it finishes. block-mount comes from edge.bootstrap usb-storage.
uci -q delete fstab.nd_extroot
if ! uci set fstab.nd_extroot=mount; then
  echo "EXTROOT_UCI_FAIL need block-mount (run edge.bootstrap preset=usb-storage first)"
  exit 1
fi
uci set fstab.nd_extroot.uuid="$UUID"
uci set fstab.nd_extroot.device="$PART"
uci set fstab.nd_extroot.target=/overlay
uci set fstab.nd_extroot.enabled=1
uci set fstab.nd_extroot.enabled_fsck=1
uci commit fstab || { echo "EXTROOT_UCI_COMMIT_FAIL"; exit 1; }

# Critical: live /etc is still INTERNAL overlay. USB copy was taken earlier and
# does not include this fstab — without this, after pivot uci show fstab.nd_extroot
# is missing and next boots may not remount extroot reliably.
for dest in \
  /mnt/netductor/upper/etc/config \
  /mnt/netductor/etc/config; do
  if [ -d "$(dirname "$dest")" ] || mkdir -p "$dest" 2>/dev/null; then
    if [ -d "$dest" ] || mkdir -p "$dest"; then
      cp -a /etc/config/fstab "$dest/fstab" 2>/dev/null && echo "EXTROOT_FSTAB_SYNCED $dest/fstab"
    fi
  fi
done

umount /mnt/netductor 2>/dev/null || true
echo EXTROOT_OK uuid=$UUID part=$PART
`, part, dev)
		out, err = runSSHOnPortLogged(port, pass, key, user, host, script, kp)
		if text := strings.TrimSpace(stripSSHNoise(out)); text != "" {
			o.logLine(text)
		}
		if err != nil {
			out = stripSSHNoise(out)
			return false, fmt.Errorf("storage extroot: %w (%s)", err, out)
		}
		needReboot = true
	} else if s.UseUSB {
		script := fmt.Sprintf(`
set +e
PART=%s
DISK=/dev/%s
if [ -b "$PART" ] && block info "$PART" 2>/dev/null | grep -q 'TYPE="ext4"'; then
  :
elif [ -b "$DISK" ] && block info "$DISK" 2>/dev/null | grep -q 'TYPE="ext4"'; then
  PART=$DISK
elif [ -b "${DISK}1" ]; then
  PART=${DISK}1
fi
mkdir -p /mnt/netductor
mountpoint -q /mnt/netductor || mount "$PART" /mnt/netductor || { echo DATA_MOUNT_FAIL; exit 1; }
UUID=$(block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1)
[ -n "$UUID" ] || UUID=$(tune2fs -l "$PART" 2>/dev/null | awk -F: '/Filesystem UUID/{gsub(/ /,"",$2); print $2; exit}')
if [ -n "$UUID" ]; then
  uci -q delete fstab.nd_data
  uci set fstab.nd_data=mount
  uci set fstab.nd_data.uuid="$UUID"
  uci set fstab.nd_data.device="$PART"
  uci set fstab.nd_data.target=/mnt/netductor
  uci set fstab.nd_data.enabled=1
  uci commit fstab
fi
echo DATA_MOUNT_OK part=$PART uuid=$UUID
`, part, dev)
		out, err = runSSHOnPort(port, pass, key, user, host, script, kp)
		if err != nil {
			return false, fmt.Errorf("storage mount: %w (%s)", err, stripSSHNoise(out))
		}
		fmt.Fprintln(os.Stderr, strings.TrimSpace(out))
	}

	// Always (re)install factory-reset wrap when USB/extroot is in play so LuCI/firstboot
	// wipes the stick. Safe to re-run: rewrites nd-factory-reset body and keeps wrappers.
	if s.UseUSB || s.Extroot {
		if err := installFactoryResetWrap(port, pass, key, user, host, kp); err != nil {
			o.logLine(fmt.Sprintf("warn: factory-reset wrap: %v", err))
		} else {
			o.logLine("==> storage: firstboot/factoryreset wrap installed (LuCI reset wipes extroot)")
		}
	}

	return needReboot, nil
}

func installPinSync(port, pass, key, user, host, kp string) error {
	// Pin lives on internal flash outside the active overlay when possible:
	// mount rootfs_data/jffs2 at /mnt/nd-pin-internal. After extroot, /etc is on USB;
	// the pin tree must stay on internal media for USB-loss recovery.
	remote := `cat > /usr/sbin/nd-pin-sync << 'PINEOF'
#!/bin/sh
set -e
PIN_ROOT=/mnt/nd-pin-internal
mkdir -p "$PIN_ROOT"
# Best-effort: mount original overlay volume if not already mounted.
if ! mountpoint -q "$PIN_ROOT" 2>/dev/null; then
  MTD=$(awk -F: '/rootfs_data/{gsub(/"/,"",$1); print $1; exit}' /proc/mtd 2>/dev/null || true)
  if [ -n "$MTD" ] && [ -b "/dev/$MTD" ]; then
    mount -t jffs2 "/dev/$MTD" "$PIN_ROOT" 2>/dev/null || \
      mount -t ubifs "ubi0:rootfs_data" "$PIN_ROOT" 2>/dev/null || true
  fi
fi
# Fallback: dedicated dir under /overlay only if internal mount failed and we are
# still on internal overlay (no extroot marker on current /overlay).
if ! mountpoint -q "$PIN_ROOT" 2>/dev/null; then
  if [ ! -f /overlay/.nd-extroot ] && [ ! -f /mnt/netductor/.nd-extroot ]; then
    PIN_ROOT=/overlay/.nd-pin-internal
    mkdir -p "$PIN_ROOT"
  else
    echo "PIN_NO_INTERNAL_MOUNT" >&2
    exit 1
  fi
fi
PIN="$PIN_ROOT/pin"
mkdir -p "$PIN/etc/netductor-agent" "$PIN/etc/config" "$PIN/usr/sbin"
fail=0
cp -a /etc/netductor-agent/config "$PIN/etc/netductor-agent/" || fail=1
cp -a /etc/netductor-agent/mtls "$PIN/etc/netductor-agent/" 2>/dev/null || true
cp -a /etc/config/network "$PIN/etc/config/" || fail=1
cp -a /etc/config/wireless "$PIN/etc/config/" 2>/dev/null || true
cp -a /etc/config/dhcp "$PIN/etc/config/" 2>/dev/null || true
cp -a /etc/config/firewall "$PIN/etc/config/" || fail=1
cp -a /etc/config/system "$PIN/etc/config/" 2>/dev/null || true
cp -a /usr/sbin/netductor-agent "$PIN/usr/sbin/" 2>/dev/null || true
[ -f "$PIN/etc/netductor-agent/config" ] || fail=1
[ -f "$PIN/etc/config/network" ] || fail=1
[ "$fail" = "0" ] || { echo "PIN_COPY_FAIL" >&2; exit 1; }
date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ > "$PIN/SYNCED_AT" 2>/dev/null || date > "$PIN/SYNCED_AT"
echo PIN_OK root=$PIN_ROOT
PINEOF
chmod 755 /usr/sbin/nd-pin-sync
grep -q nd-pin-sync /etc/crontabs/root 2>/dev/null || echo '0 */12 * * * /usr/sbin/nd-pin-sync' >> /etc/crontabs/root
/etc/init.d/cron enable 2>/dev/null || true
/etc/init.d/cron restart 2>/dev/null || true
/usr/sbin/nd-pin-sync
`

	out, err := runSSHOnPort(port, pass, key, user, host, remote, kp)
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	if strings.Contains(out, "PIN_COPY_FAIL") || strings.Contains(out, "PIN_NO_INTERNAL_MOUNT") {
		return fmt.Errorf("pin-sync failed: %s", strings.TrimSpace(out))
	}
	return nil
}

// installFactoryResetWrap makes LuCI System→Reset / firstboot wipe USB extroot.
// Stock OpenWrt only clears internal jffs2; with fstab.nd_extroot the old overlay
// comes back on the next boot.
//
// When already pivoted to USB (live root = USB overlay):
//   - never call stock jffs2reset/firstboot (they rm -rf /overlay and kill busybox/reboot);
//   - never rm -rf /overlay from our script either;
//   - clear fstab, wipe the USB block device, mtd-erase internal rootfs_data if any,
//     then force reboot so the next boot is internal flash without the old overlay.
// Without extroot: delegate to stock firstboot/jffs2reset (busybox applet or *.real).
func installFactoryResetWrap(port, pass, key, user, host, kp string) error {
	// Single remote script: body + wrappers. Avoid nested heredoc pitfalls on ash.
	remote := `
set -e
cat > /usr/sbin/nd-factory-reset << 'NDEOF'
#!/bin/sh
# netductor: factory reset safe with USB extroot. Never destroy live root via rm -rf.
set +e
echo "nd-factory-reset: begin"

# Stock firstboot.real is often a 1-liner that execs /sbin/jffs2reset — and jffs2reset
# is also our wrapper → infinite recursion. Refuse re-entry.
if [ -n "$ND_FACTORY_RESET_ACTIVE" ]; then
  echo "nd-factory-reset: re-entry ignored (avoid firstboot.real→jffs2reset wrap loop)"
  exit 0
fi
export ND_FACTORY_RESET_ACTIVE=1

want_yes=0
for a in "$@"; do
  case "$a" in
    -y|-Y|--yes) want_yes=1 ;;
  esac
done

BB=""
for b in /bin/busybox /sbin/busybox; do
  [ -x "$b" ] && BB=$b && break
done

force_reboot() {
  echo "nd-factory-reset: forcing reboot"
  sync
  if [ -n "$BB" ]; then
    "$BB" reboot -f 2>/dev/null &
  fi
  # Prefer busybox reboot: /sbin/reboot may be missing after partial overlay damage
  if [ -x /bin/busybox ]; then
    /bin/busybox reboot -f 2>/dev/null &
  fi
  sleep 1
  echo b > /proc/sysrq-trigger 2>/dev/null || true
}

# Clear current *internal* overlay without going through our wrappers.
# Never call /sbin/firstboot or /sbin/jffs2reset — those exec this script again.
# Prefer jffs2reset.real; skip firstboot.real if it only delegates to jffs2reset.
invoke_stock_overlay_reset() {
  # 1) Real jffs2reset binary/script saved beside the wrapper
  if [ -x /sbin/jffs2reset.real ] && ! grep -q nd-factory-reset /sbin/jffs2reset.real 2>/dev/null; then
    echo "nd-factory-reset: stock /sbin/jffs2reset.real"
    if [ "$want_yes" = "1" ]; then
      /sbin/jffs2reset.real -y 2>/dev/null || /sbin/jffs2reset.real 2>/dev/null || true
    else
      /sbin/jffs2reset.real "$@" 2>/dev/null || true
    fi
    return 0
  fi
  # 2) Busybox applet (does not go through /sbin/jffs2reset)
  if [ -n "$BB" ] && "$BB" --list 2>/dev/null | grep -qx jffs2reset; then
    echo "nd-factory-reset: busybox jffs2reset"
    if [ "$want_yes" = "1" ]; then
      "$BB" jffs2reset -y 2>/dev/null || "$BB" jffs2reset 2>/dev/null || true
    else
      "$BB" jffs2reset "$@" 2>/dev/null || true
    fi
    return 0
  fi
  # 3) firstboot.real only if it does NOT call wrapped /sbin/jffs2reset
  if [ -x /sbin/firstboot.real ] && ! grep -q nd-factory-reset /sbin/firstboot.real 2>/dev/null; then
    if grep -E -q '/sbin/jffs2reset([[:space:]]|$)|jffs2reset' /sbin/firstboot.real 2>/dev/null \
      && ! grep -q jffs2reset.real /sbin/firstboot.real 2>/dev/null; then
      echo "nd-factory-reset: skip firstboot.real (delegates to wrapped jffs2reset)"
    else
      echo "nd-factory-reset: stock /sbin/firstboot.real"
      if [ "$want_yes" = "1" ]; then
        /sbin/firstboot.real -y 2>/dev/null || /sbin/firstboot.real 2>/dev/null || true
      else
        /sbin/firstboot.real "$@" 2>/dev/null || true
      fi
      return 0
    fi
  fi
  if [ -x /sbin/factoryreset.real ] && ! grep -q nd-factory-reset /sbin/factoryreset.real 2>/dev/null; then
    echo "nd-factory-reset: stock /sbin/factoryreset.real"
    if [ "$want_yes" = "1" ]; then
      /sbin/factoryreset.real -y 2>/dev/null || /sbin/factoryreset.real 2>/dev/null || true
    else
      /sbin/factoryreset.real "$@" 2>/dev/null || true
    fi
    return 0
  fi
  return 1
}

# Erase internal flash overlay only (not the live USB mount).
erase_internal_rootfs_data() {
  if ! command -v mtd >/dev/null 2>&1; then
    return 1
  fi
  for part in rootfs_data overlay; do
    if grep -q "\"$part\"" /proc/mtd 2>/dev/null; then
      echo "nd-factory-reset: mtd erase $part"
      # Do not use mtd -r here: we control reboot after USB wipe.
      mtd erase "$part" 2>/dev/null && return 0
    fi
  done
  return 1
}

wipe_usb_extroot() {
  PART="$1"
  echo "nd-factory-reset: extroot path part=${PART:-unknown}"
  uci -q delete fstab.nd_extroot
  uci -q delete fstab.nd_data
  uci commit fstab 2>/dev/null || true
  # Keep /overlay mounted — only wipe the block device; do not rm through the mount.
  umount /mnt/netductor 2>/dev/null || true
  if [ -z "$PART" ]; then
    echo "nd-factory-reset: no PART to wipe (fstab cleared only)"
    return 0
  fi
  # Only umount PART if it is NOT the live root device.
  ROOT_SRC=$(mount 2>/dev/null | awk '/ on \/ /{print $1; exit}')
  OV_SRC=$(mount 2>/dev/null | awk '/ on \/overlay /{print $1; exit}')
  if [ -n "$PART" ] && [ "$PART" != "$ROOT_SRC" ] && [ "$PART" != "$OV_SRC" ]; then
    umount "$PART" 2>/dev/null || true
  fi
  sync
  if [ -b "$PART" ]; then
    dd if=/dev/zero of="$PART" bs=1M count=8 conv=notrunc 2>/dev/null \
      && echo "nd-factory-reset: wiped $PART" \
      || echo "nd-factory-reset: wipe failed $PART"
  fi
  # GPT/MBR header: only if PART is a partition node, not whole-disk live root.
  DISK=$(echo "$PART" | sed 's/[0-9]*$//')
  if [ -n "$DISK" ] && [ "$DISK" != "$PART" ] && [ -b "$DISK" ] && [ "$DISK" != "$ROOT_SRC" ]; then
    dd if=/dev/zero of="$DISK" bs=1M count=1 conv=notrunc 2>/dev/null || true
  fi
  sync
}

# live_on_usb: /overlay is the USB stick right now — stock jffs2reset would rm -rf live root.
# fstab_ext: leftover nd_extroot even when already booted from internal jffs2 after a failed wipe.
live_on_usb=0
fstab_ext=0
[ -f /overlay/.nd-extroot ] && live_on_usb=1
[ -f /.nd-extroot ] && live_on_usb=1
uci -q get fstab.nd_extroot >/dev/null 2>&1 && fstab_ext=1
OV_DEV=$(mount 2>/dev/null | awk '/ on \/overlay /{print $1; exit}')
case "$OV_DEV" in
  /dev/sd*) live_on_usb=1 ;;
  /dev/mmcblk*p[0-9]*)
    # mmc may be eMMC root; only treat as USB-style if fstab targets it as nd_extroot
    [ "$fstab_ext" = "1" ] && live_on_usb=1
    ;;
esac

PART=""
PART=$(uci -q get fstab.nd_extroot.device 2>/dev/null || true)
if [ "$live_on_usb" = "1" ]; then
  [ -n "$PART" ] || PART=$OV_DEV
fi
[ -n "$PART" ] || PART=$(block info 2>/dev/null | awk -F: '/MOUNT="\/overlay"/{gsub(/ /,"",$1); print $1; exit}')
if [ -z "$PART" ]; then
  for d in /sys/block/sd*; do
    [ -d "$d" ] || continue
    rem=$(cat "$d/removable" 2>/dev/null || echo 0)
    [ "$rem" = "1" ] || continue
    b=$(basename "$d")
    if [ -b "/dev/${b}1" ]; then PART=/dev/${b}1
    elif [ -b "/dev/$b" ]; then PART=/dev/$b
    fi
    [ -n "$PART" ] && break
  done
fi

if [ "$live_on_usb" = "1" ]; then
  echo "nd-factory-reset: live USB overlay — skip stock jffs2reset (would kill root)"
  wipe_usb_extroot "$PART"
  erase_internal_rootfs_data || true
else
  # Internal jffs2/overlayfs: stock jffs2reset.real / busybox is safe.
  # Still drop fstab.nd_extroot + wipe stick so next boot cannot pivot to old USB.
  if [ "$fstab_ext" = "1" ] || [ -n "$PART" ]; then
    wipe_usb_extroot "$PART"
  fi
  if ! invoke_stock_overlay_reset "$@"; then
    erase_internal_rootfs_data || echo "nd-factory-reset: no stock reset tool"
  fi
fi

echo "nd-factory-reset: done"
# LuCI runs firstboot -y && reboot; we always force reboot on -y so a wiped USB cannot hang the box.
if [ "$want_yes" = "1" ]; then
  force_reboot
fi
exit 0
NDEOF
chmod 755 /usr/sbin/nd-factory-reset

# Wrappers: jffs2reset FIRST so firstboot.real can be rewritten to call jffs2reset.real.
# Never mv busybox applet symlinks (argv0 breaks).
for name in jffs2reset factoryreset firstboot; do
  bin=/sbin/$name
  real=/sbin/${name}.real
  if [ -f "$bin" ] && [ ! -L "$bin" ] && grep -q nd-factory-reset "$bin" 2>/dev/null; then
    echo "WRAP_KEEP $name"
    continue
  fi
  if [ -L "$bin" ]; then
    tgt=$(readlink -f "$bin" 2>/dev/null || readlink "$bin" 2>/dev/null || true)
    case "$tgt" in
      *busybox*)
        rm -f "$bin"
        printf '%s\n' '#!/bin/sh' 'exec /usr/sbin/nd-factory-reset "$@"' > "$bin"
        chmod 755 "$bin"
        echo "WRAP_OK $name (busybox applet)"
        continue
        ;;
    esac
  fi
  if [ -e "$bin" ] && [ ! -e "$real" ]; then
    mv "$bin" "$real" 2>/dev/null || cp -a "$bin" "$real" 2>/dev/null || true
  fi
  printf '%s\n' '#!/bin/sh' 'exec /usr/sbin/nd-factory-reset "$@"' > "$bin"
  chmod 755 "$bin"
  echo "WRAP_OK $name"
done
# Stock firstboot often is: exec /sbin/jffs2reset — point .real at jffs2reset.real to break the loop.
if [ -f /sbin/firstboot.real ] && grep -E -q '/sbin/jffs2reset([[:space:]]|$)' /sbin/firstboot.real 2>/dev/null; then
  if [ -x /sbin/jffs2reset.real ]; then
    printf '%s\n' '#!/bin/sh' 'exec /sbin/jffs2reset.real "$@"' > /sbin/firstboot.real
    chmod 755 /sbin/firstboot.real
    echo "WRAP_PATCH firstboot.real -> jffs2reset.real"
  fi
fi
# LuCI always expects firstboot and/or jffs2reset — create even if stock never had them.
for name in firstboot jffs2reset; do
  if [ ! -x /sbin/$name ]; then
    printf '%s\n' '#!/bin/sh' 'exec /usr/sbin/nd-factory-reset "$@"' > /sbin/$name
    chmod 755 /sbin/$name
    echo "WRAP_OK $name (created)"
  fi
done

# Stage into USB overlay so post-pivot LuCI reset still hits our wrap.
# If we are already pivoted, live /sbin is the USB upper — still try mount for whole-disk copies.
PART=$(uci -q get fstab.nd_extroot.device 2>/dev/null || true)
[ -n "$PART" ] || PART=$(block info 2>/dev/null | awk -F: '/LABEL="netductor"|TYPE="ext4"/{gsub(/ /,"",$1); print $1; exit}')
mkdir -p /mnt/netductor
if [ -n "$PART" ] && [ -b "$PART" ]; then
  # Avoid mounting the live overlay device onto itself.
  OV_SRC=$(mount 2>/dev/null | awk '/ on \/overlay /{print $1; exit}')
  if [ "$PART" != "$OV_SRC" ]; then
    umount /mnt/netductor 2>/dev/null || true
    mount "$PART" /mnt/netductor 2>/dev/null || true
  fi
fi
for root in /mnt/netductor/upper /mnt/netductor; do
  [ -d "$root" ] || continue
  mkdir -p "$root/usr/sbin" "$root/sbin" 2>/dev/null || true
  cp -a /usr/sbin/nd-factory-reset "$root/usr/sbin/" 2>/dev/null && echo "WRAP_USB_BIN $root" || true
  for n in firstboot factoryreset jffs2reset; do
    [ -f /sbin/$n ] && cp -a /sbin/$n "$root/sbin/" 2>/dev/null || true
    [ -f /sbin/${n}.real ] && cp -a /sbin/${n}.real "$root/sbin/" 2>/dev/null || true
  done
done
umount /mnt/netductor 2>/dev/null || true
echo WRAP_DONE
`
	out, err := runSSHOnPort(port, pass, key, user, host, remote, kp)
	if err != nil {
		return fmt.Errorf("%w: %s", err, stripSSHNoise(out))
	}
	if !strings.Contains(out, "WRAP_DONE") && !strings.Contains(out, "WRAP_OK") {
		return fmt.Errorf("factory-reset wrap incomplete: %s", strings.TrimSpace(stripSSHNoise(out)))
	}
	return nil
}

// stageOperatorKeysOntoExtroot writes the operator pubkey into the USB overlay tree
// so authorized_keys survive pivot (live install alone is on internal flash).
func stageOperatorKeysOntoExtroot(o EdgeOpts, pub string) error {
	pub = strings.TrimSpace(pub)
	if pub == "" {
		return fmt.Errorf("empty pubkey")
	}
	esc := strings.ReplaceAll(pub, "'", `'"'"'`)
	script := fmt.Sprintf(`
set +e
PART=$(uci -q get fstab.nd_extroot.device 2>/dev/null || true)
[ -n "$PART" ] || PART=$(block info 2>/dev/null | awk -F: '/LABEL="netductor"|MOUNT="\/overlay"/{gsub(/ /,"",$1); print $1; exit}')
if [ -z "$PART" ]; then
  for d in /sys/block/sd*; do
    [ -d "$d" ] || continue
    rem=$(cat "$d/removable" 2>/dev/null || echo 0)
    [ "$rem" = "1" ] || continue
    b=$(basename "$d")
    if [ -b "/dev/${b}1" ]; then PART=/dev/${b}1
    elif [ -b "/dev/$b" ]; then PART=/dev/$b
    fi
    [ -n "$PART" ] && break
  done
fi
mkdir -p /mnt/netductor
umount /mnt/netductor 2>/dev/null || true
if [ -z "$PART" ] || ! mount "$PART" /mnt/netductor 2>/dev/null; then
  echo "STAGE_KEYS_MOUNT_FAIL part=${PART:-none}"
  exit 1
fi
n=0
for root in /mnt/netductor/upper /mnt/netductor; do
  [ -d "$root" ] || continue
  mkdir -p "$root/etc/dropbear" "$root/root/.ssh"
  for ak in "$root/etc/dropbear/authorized_keys" "$root/root/.ssh/authorized_keys"; do
    touch "$ak"
    chmod 600 "$ak"
    grep -qxF '%s' "$ak" 2>/dev/null || echo '%s' >> "$ak"
    n=$((n+1))
  done
  chmod 700 "$root/root/.ssh" 2>/dev/null || true
done
mkdir -p /etc/dropbear /root/.ssh
chmod 700 /root/.ssh
for ak in /etc/dropbear/authorized_keys /root/.ssh/authorized_keys; do
  touch "$ak"; chmod 600 "$ak"
  grep -qxF '%s' "$ak" 2>/dev/null || echo '%s' >> "$ak"
done
# Pivot-critical UCI already on live before copy; re-sync to USB upper in case copy was partial/old.
for cfg in dropbear system luci; do
  [ -f /etc/config/$cfg ] || continue
  for root in /mnt/netductor/upper /mnt/netductor; do
    [ -d "$root" ] || continue
    mkdir -p "$root/etc/config"
    cp -a /etc/config/$cfg "$root/etc/config/" 2>/dev/null && echo STAGE_CFG $cfg $root || true
  done
done
umount /mnt/netductor 2>/dev/null || true
echo STAGE_KEYS_OK files=$n part=$PART
`, esc, esc, esc, esc)
	out, err := runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, script, o.PrimaryKeyPassphrase)
	if err != nil {
		return fmt.Errorf("%w: %s", err, stripSSHNoise(out))
	}
	if !strings.Contains(out, "STAGE_KEYS_OK") {
		return fmt.Errorf("stage keys incomplete: %s", strings.TrimSpace(stripSSHNoise(out)))
	}
	return nil
}
