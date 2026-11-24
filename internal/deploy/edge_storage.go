package deploy

import (
	"fmt"
	"os"
	"strings"
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
	fmt.Fprintf(os.Stderr, "==> storage: disk=%s part=%s format=%v extroot=%v\n", dev, part, s.Format, s.Extroot)

	if s.Format {
		fmt.Fprintln(os.Stderr, "==> storage: FORMAT USB (destructive)")
		script := fmt.Sprintf(`
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
  # Travel OpenWrt often has neither parted nor fdisk — format whole disk.
  echo "FORMAT_WHOLE_DISK no parted/fdisk"
  PART=$DISK
fi
if [ -n "$PART" ] && [ "$PART" != "$DISK" ] && [ ! -b "$PART" ]; then
  echo "FORMAT_PART_MISSING $PART — falling back to whole disk"
  PART=$DISK
fi
[ -b "$PART" ] || { echo "FORMAT_NO_BLOCK part=$PART"; ls -l $DISK* 2>/dev/null; exit 1; }
mkfs.ext4 -F -L netductor "$PART"
echo FORMAT_OK part=$PART
`, dev)
		out, err = runSSHOnPort(port, pass, key, user, host, script, kp)
		if err != nil {
			return false, fmt.Errorf("storage format: %w (%s)", err, stripSSHNoise(out))
		}
		fmt.Fprintln(os.Stderr, strings.TrimSpace(out))
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "FORMAT_OK part=") {
				part = strings.TrimPrefix(line, "FORMAT_OK part=")
				fmt.Fprintln(os.Stderr, "==> storage: using part", part)
			}
		}
	}

	if s.PinFallback {
		if err := installPinSync(port, pass, key, user, host, kp); err != nil {
			return false, fmt.Errorf("storage pin-sync: %w", err)
		}
		fmt.Fprintln(os.Stderr, "==> storage: pin sync installed before extroot")
	}
	if s.Extroot {
		fmt.Fprintln(os.Stderr, "==> storage: configure extroot")
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

# block-mount provides /etc/config/fstab schema (apk or opkg).
if command -v apk >/dev/null 2>&1; then
  apk add block-mount 2>/dev/null || true
elif command -v opkg >/dev/null 2>&1; then
  opkg install block-mount 2>/dev/null || true
fi

uci -q delete fstab.nd_extroot
uci set fstab.nd_extroot=mount || { echo "EXTROOT_UCI_FAIL need block-mount"; exit 1; }
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
		out, err = runSSHOnPort(port, pass, key, user, host, script, kp)
		if err != nil {
			out = stripSSHNoise(out)
			return false, fmt.Errorf("storage extroot: %w (%s)", err, out)
		}
		fmt.Fprintln(os.Stderr, strings.TrimSpace(out))
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
