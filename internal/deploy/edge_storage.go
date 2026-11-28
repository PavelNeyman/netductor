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
		// Start mkfs under nohup, then poll with short SSH sessions.
		start := fmt.Sprintf(`
set -e
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
rm -f /tmp/nd-mkfs.rc /tmp/nd-mkfs.log /tmp/nd-mkfs.part /tmp/nd-mkfs.before_uuid
echo "$PART" > /tmp/nd-mkfs.part
# Record pre-mkfs UUID so we never treat the OLD ext4 as format success (FORMAT_INFER race).
block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1 > /tmp/nd-mkfs.before_uuid || true
# Background so SSH disconnect does not kill mkfs.
nohup sh -c 'mkfs.ext4 -F -L netductor "$(cat /tmp/nd-mkfs.part)" > /tmp/nd-mkfs.log 2>&1; echo $? > /tmp/nd-mkfs.rc' >/dev/null 2>&1 &
echo FORMAT_STARTED part=$PART
BEFORE_UUID=$(cat /tmp/nd-mkfs.before_uuid 2>/dev/null || true)
echo FORMAT_BEFORE_UUID=$BEFORE_UUID
`, dev)
		out, err := runSSHOnPortLogged(port, pass, key, user, host, start, kp)
		o.logLine(strings.TrimSpace(stripSSHNoise(out)))
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
		// Poll up to ~15 minutes (large USB on slow mips).
		var done bool
		for i := 0; i < 180; i++ {
			poll := fmt.Sprintf(`
PART=%s
[ -f /tmp/nd-mkfs.part ] && PART=$(cat /tmp/nd-mkfs.part)
BEFORE=$(cat /tmp/nd-mkfs.before_uuid 2>/dev/null || true)
if [ -f /tmp/nd-mkfs.rc ]; then
  echo FORMAT_RC=$(cat /tmp/nd-mkfs.rc)
  echo FORMAT_PART=$PART
  tail -8 /tmp/nd-mkfs.log 2>/dev/null || true
  exit 0
fi
# Only infer success if mkfs is no longer running AND UUID changed (or was empty).
# Never accept pre-existing ext4 while mkfs still running or UUID unchanged — that was the race.
MKFS_ALIVE=0
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
fi
echo FORMAT_WAIT n=%d part=$PART mkfs_alive=$MKFS_ALIVE before=$BEFORE
ps w 2>/dev/null | grep -F mkfs.ext4 | grep -v grep || true
`, part, i)
			out, err = runSSHOnPortLogged(port, pass, key, user, host, poll, kp)
			text := stripSSHNoise(out)
			if strings.TrimSpace(text) != "" {
				o.logLine(strings.TrimSpace(text))
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
				// Transient SSH blip during mkfs — keep polling.
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
		// LuCI System→Reset must wipe USB extroot too; stock firstboot only clears jffs2.
		if err := installFactoryResetWrap(port, pass, key, user, host, kp); err != nil {
			o.logLine(fmt.Sprintf("warn: factory-reset wrap: %v", err))
		} else {
			o.logLine("==> storage: firstboot/factoryreset wrap installed (LuCI reset wipes extroot)")
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
// Critical when already pivoted to USB:
//   - do NOT rm -rf /overlay while root is the USB overlay (kills live busybox/reboot);
//   - resolve stock firstboot/jffs2reset (busybox applet or *.real) BEFORE wipe;
//   - clear fstab, wipe USB, then force reboot so next boot is internal flash.
func installFactoryResetWrap(port, pass, key, user, host, kp string) error {
	// Single remote script: body + wrappers. Avoid nested heredoc pitfalls on ash.
	remote := `
set -e
cat > /usr/sbin/nd-factory-reset << 'NDEOF'
#!/bin/sh
# netductor: factory reset with USB extroot. Must not destroy the live root mid-run.
set +e
echo "nd-factory-reset: begin"

want_yes=0
for a in "$@"; do
  [ "$a" = "-y" ] || [ "$a" = "-Y" ] || [ "$a" = "--yes" ] && want_yes=1
done

# --- locate stock reset BEFORE any destructive work ---
BB=""
for b in /bin/busybox /sbin/busybox; do
  [ -x "$b" ] && BB=$b && break
done

invoke_stock() {
  # Prefer renamed real binaries (non-busybox scripts).
  for c in /sbin/jffs2reset.real /sbin/firstboot.real /sbin/factoryreset.real; do
    if [ -x "$c" ] && ! grep -q nd-factory-reset "$c" 2>/dev/null; then
      echo "nd-factory-reset: stock $c"
      if [ "$want_yes" = "1" ]; then
        "$c" -y 2>/dev/null || "$c" 2>/dev/null || true
      else
        "$c" "$@" 2>/dev/null || true
      fi
      return 0
    fi
  done
  # Busybox applets (argv0 matters — call via busybox NAME).
  if [ -n "$BB" ]; then
    for applet in jffs2reset firstboot factoryreset; do
      if "$BB" --list 2>/dev/null | grep -qx "$applet" || "$BB" "$applet" --help >/dev/null 2>&1; then
        echo "nd-factory-reset: busybox $applet"
        if [ "$want_yes" = "1" ]; then
          "$BB" "$applet" -y 2>/dev/null || "$BB" "$applet" 2>/dev/null || true
        else
          "$BB" "$applet" "$@" 2>/dev/null || true
        fi
        return 0
      fi
    done
  fi
  # Unwrapped sbin script (not our wrap, not a busybox symlink we already replaced).
  for c in /sbin/jffs2reset /sbin/firstboot /sbin/factoryreset; do
    if [ -f "$c" ] && [ ! -L "$c" ] && ! grep -q nd-factory-reset "$c" 2>/dev/null; then
      echo "nd-factory-reset: stock file $c"
      if [ "$want_yes" = "1" ]; then
        "$c" -y 2>/dev/null || "$c" 2>/dev/null || true
      else
        "$c" "$@" 2>/dev/null || true
      fi
      return 0
    fi
  done
  # Last resort: erase rootfs_data MTD if present (internal overlay).
  if command -v mtd >/dev/null 2>&1; then
    for part in rootfs_data overlay; do
      if grep -q "\"$part\"" /proc/mtd 2>/dev/null; then
        echo "nd-factory-reset: mtd erase $part"
        mtd -r erase "$part" 2>/dev/null && return 0
        mtd erase "$part" 2>/dev/null && return 0
      fi
    done
  fi
  echo "nd-factory-reset: no stock firstboot/jffs2reset (extroot wipe + reboot only)"
  return 1
}

has_ext=0
[ -f /overlay/.nd-extroot ] && has_ext=1
[ -f /.nd-extroot ] && has_ext=1
uci -q get fstab.nd_extroot >/dev/null 2>&1 && has_ext=1
PART=""
PART=$(uci -q get fstab.nd_extroot.device 2>/dev/null || true)
[ -n "$PART" ] || PART=$(mount 2>/dev/null | awk '/ on \/overlay /{print $1; exit}')
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

# 1) Stock internal reset first (while binaries still intact).
invoke_stock "$@"

# 2) Detach + wipe USB extroot so next boot cannot remount old overlay.
if [ "$has_ext" = "1" ] || [ -n "$PART" ]; then
  echo "nd-factory-reset: extroot path part=${PART:-unknown}"
  uci -q delete fstab.nd_extroot
  uci -q delete fstab.nd_data
  uci commit fstab 2>/dev/null || true
  # Keep /overlay mounted: wiping the block under a live root is OK for next boot;
  # deleting files through the mount kills busybox/reboot in this shell.
  umount /mnt/netductor 2>/dev/null || true
  if [ -n "$PART" ]; then
    umount "$PART" 2>/dev/null || true
    case "$PART" in
      /dev/sd*[0-9]) umount "${PART%%[0-9]*}" 2>/dev/null || true ;;
    esac
    sync
    if [ -b "$PART" ]; then
      dd if=/dev/zero of="$PART" bs=1M count=8 2>/dev/null && echo "nd-factory-reset: wiped $PART" || echo "nd-factory-reset: wipe failed $PART"
    fi
    DISK=$(echo "$PART" | sed 's/[0-9]*$//')
    if [ -n "$DISK" ] && [ "$DISK" != "$PART" ] && [ -b "$DISK" ]; then
      dd if=/dev/zero of="$DISK" bs=1M count=2 2>/dev/null || true
    fi
    sync
  else
    echo "nd-factory-reset: no PART to wipe (fstab cleared only)"
  fi
fi

echo "nd-factory-reset: done"
# LuCI / firstboot -y expect a reboot; force it (busybox reboot may already be broken after wipe).
if [ "$want_yes" = "1" ]; then
  echo "nd-factory-reset: forcing reboot"
  sync
  if [ -n "$BB" ]; then
    "$BB" reboot -f 2>/dev/null &
  fi
  reboot -f 2>/dev/null &
  sleep 2
  echo b > /proc/sysrq-trigger 2>/dev/null || true
fi
exit 0
NDEOF
chmod 755 /usr/sbin/nd-factory-reset

# Wrappers: busybox applets must not be mv'd (argv0 breaks); replace symlink with script.
for name in firstboot factoryreset jffs2reset; do
  bin=/sbin/$name
  real=/sbin/${name}.real
  [ -e "$bin" ] || [ -e "$real" ] || [ -L "$bin" ] || continue
  if [ -f "$bin" ] && [ ! -L "$bin" ] && grep -q nd-factory-reset "$bin" 2>/dev/null; then
    echo "WRAP_SKIP $name"
    continue
  fi
  if [ -L "$bin" ]; then
    tgt=$(readlink -f "$bin" 2>/dev/null || readlink "$bin" 2>/dev/null || true)
    case "$tgt" in
      *busybox*)
        # Drop applet symlink; stock path is busybox $name (handled in nd-factory-reset).
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

# Stage into USB overlay so post-pivot LuCI reset still hits our wrap.
PART=$(uci -q get fstab.nd_extroot.device 2>/dev/null || true)
[ -n "$PART" ] || PART=$(block info 2>/dev/null | awk -F: '/LABEL="netductor"|TYPE="ext4"/{gsub(/ /,"",$1); print $1; exit}')
mkdir -p /mnt/netductor
if [ -n "$PART" ] && [ -b "$PART" ]; then
  umount /mnt/netductor 2>/dev/null || true
  mount "$PART" /mnt/netductor 2>/dev/null || true
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
[ -n "$PART" ] || PART=/dev/sda
mkdir -p /mnt/netductor
umount /mnt/netductor 2>/dev/null || true
if ! mount "$PART" /mnt/netductor 2>/dev/null; then
  echo "STAGE_KEYS_MOUNT_FAIL part=$PART"
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
