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
for d in /sys/block/sd*; do
  [ -d "$d" ] || continue
  echo DEV=$(basename "$d")
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
if [ ! -b %s ]; then
  if command -v parted >/dev/null 2>&1; then
    parted -s $DISK -- mklabel gpt mkpart primary ext4 1MiB 100%%
  elif command -v fdisk >/dev/null 2>&1; then
    echo -e 'o\nn\np\n1\n\n\nw\n' | fdisk $DISK
  fi
  sleep 2
fi
PART=%s
[ -b "$PART" ] || PART=$DISK
umount "$PART" 2>/dev/null || true
mkfs.ext4 -F -L netductor "$PART"
echo FORMAT_OK part=$PART
`, dev, part, part)
		out, err = runSSHOnPort(port, pass, key, user, host, script, kp)
		if err != nil {
			return false, fmt.Errorf("storage format: %w (%s)", err, out)
		}
		fmt.Fprintln(os.Stderr, strings.TrimSpace(out))
	}

	if s.Extroot {
		fmt.Fprintln(os.Stderr, "==> storage: configure extroot")
		script := fmt.Sprintf(`
set -e
PART=%s
[ -b "$PART" ] || PART=/dev/%s
mkdir -p /mnt/netductor
umount /mnt/netductor 2>/dev/null || true
mount "$PART" /mnt/netductor
if [ ! -f /mnt/netductor/.nd-extroot ]; then
  tar -C /overlay -cf - . 2>/dev/null | tar -C /mnt/netductor -xf - || cp -a /overlay/. /mnt/netductor/ 2>/dev/null || true
  touch /mnt/netductor/.nd-extroot
fi
UUID=$(block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1)
[ -n "$UUID" ] || UUID=$(blkid -s UUID -o value "$PART" 2>/dev/null || true)
[ -n "$UUID" ] || { echo "no UUID for $PART"; exit 1; }
opkg install block-mount kmod-fs-ext4 kmod-usb-storage 2>/dev/null || true
uci -q delete fstab.nd_extroot
uci set fstab.nd_extroot=mount
uci set fstab.nd_extroot.uuid="$UUID"
uci set fstab.nd_extroot.target=/overlay
uci set fstab.nd_extroot.enabled=1
uci set fstab.nd_extroot.enabled_fsck=1
uci commit fstab
umount /mnt/netductor 2>/dev/null || true
echo EXTROOT_OK uuid=$UUID
`, part, dev)
		out, err = runSSHOnPort(port, pass, key, user, host, script, kp)
		if err != nil {
			return false, fmt.Errorf("storage extroot: %w (%s)", err, out)
		}
		fmt.Fprintln(os.Stderr, strings.TrimSpace(out))
		needReboot = true
	} else if s.UseUSB {
		script := fmt.Sprintf(`
set -e
PART=%s
[ -b "$PART" ] || PART=/dev/%s
mkdir -p /mnt/netductor
mountpoint -q /mnt/netductor || mount "$PART" /mnt/netductor
UUID=$(block info "$PART" 2>/dev/null | sed -n 's/.*UUID="\([^"]*\)".*/\1/p' | head -1)
[ -n "$UUID" ] || UUID=$(blkid -s UUID -o value "$PART" 2>/dev/null || true)
if [ -n "$UUID" ]; then
  uci -q delete fstab.nd_data
  uci set fstab.nd_data=mount
  uci set fstab.nd_data.uuid="$UUID"
  uci set fstab.nd_data.target=/mnt/netductor
  uci set fstab.nd_data.enabled=1
  uci commit fstab
fi
echo DATA_MOUNT_OK
`, part, dev)
		out, err = runSSHOnPort(port, pass, key, user, host, script, kp)
		if err != nil {
			return false, fmt.Errorf("storage mount: %w (%s)", err, out)
		}
		fmt.Fprintln(os.Stderr, strings.TrimSpace(out))
	}

	if s.PinFallback {
		if err := installPinSync(port, pass, key, user, host, kp); err != nil {
			fmt.Fprintln(os.Stderr, "warn: pin sync install:", err)
		} else {
			fmt.Fprintln(os.Stderr, "==> storage: pin sync script installed")
		}
	}
	return needReboot, nil
}

func installPinSync(port, pass, key, user, host, kp string) error {
	remote := `cat > /usr/sbin/nd-pin-sync << 'PINEOF'
#!/bin/sh
PIN=/etc/netductor-agent/pin
mkdir -p "$PIN/etc/netductor-agent" "$PIN/etc/config" "$PIN/usr/sbin"
cp -a /etc/netductor-agent/config "$PIN/etc/netductor-agent/" 2>/dev/null || true
cp -a /etc/netductor-agent/mtls "$PIN/etc/netductor-agent/" 2>/dev/null || true
cp -a /etc/config/network "$PIN/etc/config/" 2>/dev/null || true
cp -a /etc/config/wireless "$PIN/etc/config/" 2>/dev/null || true
cp -a /etc/config/dhcp "$PIN/etc/config/" 2>/dev/null || true
cp -a /etc/config/firewall "$PIN/etc/config/" 2>/dev/null || true
cp -a /etc/config/system "$PIN/etc/config/" 2>/dev/null || true
cp -a /usr/sbin/netductor-agent "$PIN/usr/sbin/" 2>/dev/null || true
date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ > "$PIN/SYNCED_AT" 2>/dev/null || date > "$PIN/SYNCED_AT"
PINEOF
chmod 755 /usr/sbin/nd-pin-sync
grep -q nd-pin-sync /etc/crontabs/root 2>/dev/null || echo '0 */12 * * * /usr/sbin/nd-pin-sync' >> /etc/crontabs/root
/etc/init.d/cron enable 2>/dev/null || true
/etc/init.d/cron restart 2>/dev/null || true
/usr/sbin/nd-pin-sync
echo PIN_OK
`
	out, err := runSSHOnPort(port, pass, key, user, host, remote, kp)
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}
