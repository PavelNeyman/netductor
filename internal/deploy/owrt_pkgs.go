package deploy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// OpenWrt packages required for USB mass-storage + extroot (router has no internet).
// Mac host downloads during offline-prep or on-demand at deploy; install is always local via apk/opkg.

const openWrtDownloads = "https://downloads.openwrt.org"

// OpenWrtMeta is probed from the router (/etc/openwrt_release + uname).
type OpenWrtMeta struct {
	Release string // 25.12.5
	Target  string // ramips/mt76x8
	Arch    string // mipsel_24kc
	Kernel  string // 6.12.94
	PkgMgr  string // apk | opkg
}

// OpenWrtPkgCacheDir is ~/.cache/netductor/offline/edge/owrt/<release>/<target-sanitized>/
func OpenWrtPkgCacheDir(release, target string) string {
	home, _ := os.UserHomeDir()
	safe := strings.ReplaceAll(target, "/", "_")
	return filepath.Join(home, ".cache", "netductor", "offline", "edge", "owrt", release, safe)
}

// ProbeOpenWrtMeta reads release/target/arch/kernel/pkg manager over SSH.
func ProbeOpenWrtMeta(password, keyPath, user, host, keyPassphrase string) (OpenWrtMeta, error) {
	// BusyBox: `command -v missing` exits 127 — always end with true so SSH is not false-failed.
	script := `
[ -f /etc/openwrt_release ] && . /etc/openwrt_release
echo RELEASE=${DISTRIB_RELEASE:-}
echo TARGET=${DISTRIB_TARGET:-}
echo ARCH=${DISTRIB_ARCH:-}
uname -r
command -v apk >/dev/null 2>&1 && echo PKGMGR=apk
command -v opkg >/dev/null 2>&1 && echo PKGMGR=opkg
true
`
	out, err := runSSHOnPort(factorySSHPort(), password, keyPath, user, host, script, keyPassphrase)
	var m OpenWrtMeta
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "RELEASE="):
			m.Release = strings.TrimPrefix(line, "RELEASE=")
		case strings.HasPrefix(line, "TARGET="):
			m.Target = strings.TrimPrefix(line, "TARGET=")
		case strings.HasPrefix(line, "ARCH="):
			m.Arch = strings.TrimPrefix(line, "ARCH=")
		case strings.HasPrefix(line, "PKGMGR="):
			v := strings.TrimPrefix(line, "PKGMGR=")
			if m.PkgMgr == "" || v == "apk" {
				m.PkgMgr = v
			}
		case regexp.MustCompile(`^\d+\.\d+`).MatchString(line):
			m.Kernel = line
		}
	}
	if m.Release == "" || m.Target == "" {
		if err != nil {
			return m, fmt.Errorf("probe openwrt meta: %w (%s)", err, strings.TrimSpace(out))
		}
		return m, fmt.Errorf("not OpenWrt or incomplete /etc/openwrt_release (got release=%q target=%q)", m.Release, m.Target)
	}
	_ = err // data present; ignore BusyBox exit 127 from command -v
	if m.PkgMgr == "" {
		if strings.HasPrefix(m.Release, "24.") || strings.HasPrefix(m.Release, "23.") {
			m.PkgMgr = "opkg"
		} else {
			m.PkgMgr = "apk"
		}
	}
	return m, nil
}

// usbPkgBaseNames are logical package names (version suffix resolved from index).
// Include e2fsprogs runtime deps: router often has no WAN, so apk/opkg cannot fetch them.
var usbPkgBaseNames = []string{
	"kmod-usb-storage",
	"kmod-scsi-core",
	"kmod-fs-ext4",
	"kmod-lib-crc16",
	"block-mount",
	"libuuid1",
	"libblkid1",
	"libcomerr2",
	"libss2",
	"libe2p2",
	"libext2fs2",
	"e2fsprogs",
}

// EnsureOpenWrtUSBPackagesCached downloads required .apk/.ipk into cache for this board release.
// Mac needs internet; router does not.
func EnsureOpenWrtUSBPackagesCached(m OpenWrtMeta) (dir string, files []string, err error) {
	if m.Release == "" || m.Target == "" {
		return "", nil, fmt.Errorf("openwrt meta incomplete")
	}
	dir = OpenWrtPkgCacheDir(m.Release, m.Target)
	_ = os.MkdirAll(dir, 0o755)

	ext := ".apk"
	if m.PkgMgr == "opkg" {
		ext = ".ipk"
	}

	// Offline-first: if every package is already in cache, never touch the network.
	// (Probe runs on LAN-only Mac↔router; Mac may have no route to downloads.openwrt.org.)
	if got, ok := listCachedUSBPackages(dir, ext); ok {
		for _, f := range got {
			fmt.Fprintln(os.Stderr, "==> using cached OpenWrt pkg", filepath.Base(f))
		}
		return dir, got, nil
	}

	kmodBase, err := resolveKmodBaseURL(m)
	if err != nil {
		// Last chance: any partial cache is better than hard fail with EOF
		if got, ok := listCachedUSBPackages(dir, ext); ok {
			fmt.Fprintln(os.Stderr, "warn: kmod index unreachable; using partial/full local cache")
			return dir, got, nil
		}
		if got := listAnyCachedUSBPackages(dir, ext); len(got) > 0 {
			fmt.Fprintln(os.Stderr, "warn: kmod index unreachable; installing", len(got), "cached packages (may be incomplete)")
			return dir, got, nil
		}
		return dir, nil, err
	}
	pkgBase := fmt.Sprintf("%s/releases/%s/targets/%s/packages/", openWrtDownloads, m.Release, m.Target)
	branch := openWrtBranch(m.Release)
	archBase := fmt.Sprintf("%s/releases/packages-%s/%s/base/", openWrtDownloads, branch, m.Arch)
	if m.Arch == "" {
		archBase = ""
	}

	var got []string
	for _, base := range usbPkgBaseNames {
		path, err := fetchPkgByPrefix(dir, base, ext, []string{kmodBase, pkgBase, archBase})
		if err != nil {
			return dir, got, fmt.Errorf("%s: %w", base, err)
		}
		got = append(got, path)
		fmt.Fprintln(os.Stderr, "==> cached OpenWrt pkg", filepath.Base(path))
	}
	return dir, got, nil
}

// listCachedUSBPackages returns true only when every required package name has a matching file.
func listCachedUSBPackages(dir, ext string) ([]string, bool) {
	var got []string
	for _, base := range usbPkgBaseNames {
		matches, _ := filepath.Glob(filepath.Join(dir, base+"*"+ext))
		if len(matches) == 0 {
			return nil, false
		}
		got = append(got, matches[0])
	}
	return got, true
}

func listAnyCachedUSBPackages(dir, ext string) []string {
	var got []string
	for _, base := range usbPkgBaseNames {
		matches, _ := filepath.Glob(filepath.Join(dir, base+"*"+ext))
		if len(matches) > 0 {
			got = append(got, matches[0])
		}
	}
	return got
}

func openWrtBranch(release string) string {
	// 25.12.5 → 25.12 ; 24.10.2 → 24.10
	parts := strings.Split(release, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return release
}

func resolveKmodBaseURL(m OpenWrtMeta) (string, error) {
	indexURL := fmt.Sprintf("%s/releases/%s/targets/%s/kmods/", openWrtDownloads, m.Release, m.Target)
	body, err := httpGetString(indexURL)
	if err != nil {
		return "", fmt.Errorf("list kmods: %w", err)
	}
	// href="6.12.94-1-2fe195f1…/"
	re := regexp.MustCompile(`href="([0-9][^"/]+)/"`)
	matches := re.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no kmod dirs under %s", indexURL)
	}
	pick := matches[0][1]
	if m.Kernel != "" {
		for _, sm := range matches {
			if strings.HasPrefix(sm[1], m.Kernel) {
				pick = sm[1]
				break
			}
		}
	}
	return indexURL + pick + "/", nil
}

func fetchPkgByPrefix(dir, baseName, ext string, bases []string) (string, error) {
	// already cached?
	if matches, _ := filepath.Glob(filepath.Join(dir, baseName+"*"+ext)); len(matches) > 0 {
		return matches[0], nil
	}
	var lastErr error
	for _, base := range bases {
		if base == "" {
			continue
		}
		body, err := httpGetString(base)
		if err != nil {
			lastErr = err
			continue
		}
		re := regexp.MustCompile(`href="(` + regexp.QuoteMeta(baseName) + `[^"]*` + regexp.QuoteMeta(ext) + `)"`)
		sm := re.FindStringSubmatch(body)
		if len(sm) < 2 {
			lastErr = fmt.Errorf("not found in %s", base)
			continue
		}
		name := sm[1]
		url := base + name
		dest := filepath.Join(dir, name)
		if err := httpDownloadFile(url, dest); err != nil {
			lastErr = err
			continue
		}
		return dest, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("package %s not found", baseName)
	}
	return "", lastErr
}

func httpGetString(url string) (string, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return string(b), err
}

func httpDownloadFile(url, dest string) error {
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, url)
	}
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

// RouterNeedsUSBStorage returns true if usb-storage module is not available on device.
func RouterNeedsUSBStorage(password, keyPath, user, host, keyPassphrase string) (bool, string, error) {
	script := `
if lsmod 2>/dev/null | grep -q usb_storage; then echo HAVE=yes; exit 0; fi
if [ -d /sys/module/usb_storage ]; then echo HAVE=yes; exit 0; fi
KO=$(find /lib/modules -name 'usb-storage.ko*' 2>/dev/null | head -1)
if [ -n "$KO" ]; then echo HAVE=ko path=$KO; exit 0; fi
echo HAVE=no
true
`
	out, err := runSSHOnPort(factorySSHPort(), password, keyPath, user, host, script, keyPassphrase)
	if err != nil && !strings.Contains(out, "HAVE=") {
		return true, out, err
	}
	s := strings.TrimSpace(out)
	if strings.Contains(s, "HAVE=yes") || strings.Contains(s, "HAVE=ko") {
		return false, s, nil
	}
	return true, s, nil
}

// EnsureUSBStorageOnRouter checks modules; if missing, installs from Mac cache (download if needed).
// Always safe to call on OpenWrt; no-op when modules already present.
func EnsureUSBStorageOnRouter(password, keyPath, user, host, keyPassphrase string) error {
	need, note, err := RouterNeedsUSBStorage(password, keyPath, user, host, keyPassphrase)
	if err != nil {
		return fmt.Errorf("usb-storage check: %w (%s)", err, note)
	}
	if !need {
		fmt.Fprintln(os.Stderr, "==> usb-storage already present on router")
		// still try modprobe in case ko exists but not loaded
		_, _ = runSSHOnPort(factorySSHPort(), password, keyPath, user, host,
			`modprobe usb-storage 2>/dev/null; modprobe sd_mod 2>/dev/null; true`, keyPassphrase)
		return nil
	}
	fmt.Fprintln(os.Stderr, "==> usb-storage missing on router — installing packages from Mac cache")
	fmt.Fprintln(os.Stderr, "==> (probe installs kmods so the card can show USB / Format / overlay)")

	meta, err := ProbeOpenWrtMeta(password, keyPath, user, host, keyPassphrase)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "==> OpenWrt %s target=%s arch=%s kernel=%s mgr=%s\n",
		meta.Release, meta.Target, meta.Arch, meta.Kernel, meta.PkgMgr)
	fmt.Fprintln(os.Stderr, "==> uploading packages to router /tmp/nd-owrt-pkgs …")

	dir, files, err := EnsureOpenWrtUSBPackagesCached(meta)
	if err != nil {
		return fmt.Errorf("cache OpenWrt USB packages: %w (Mac needs internet once)", err)
	}
	_ = dir

	// scp each package to /tmp/nd-owrt-pkgs/
	remoteDir := "/tmp/nd-owrt-pkgs"
	_, _ = runSSHOnPort(factorySSHPort(), password, keyPath, user, host,
		"rm -rf "+remoteDir+"; mkdir -p "+remoteDir, keyPassphrase)
	for _, f := range files {
		if err := scpToRouter(password, keyPath, user, host, keyPassphrase, f, remoteDir+"/"+filepath.Base(f)); err != nil {
			return fmt.Errorf("scp %s: %w", filepath.Base(f), err)
		}
	}

	var installScript string
	if meta.PkgMgr == "opkg" {
		installScript = `
set -e
cd /tmp/nd-owrt-pkgs
for f in *.ipk; do
  [ -f "$f" ] || continue
  opkg install --force-depends "$f" || opkg install "$f" || true
done
modprobe usb-storage 2>/dev/null || true
modprobe sd_mod 2>/dev/null || true
lsmod | grep -E 'usb_storage|sd_mod' || true
echo INSTALL_DONE
`
	} else {
		// Offline: no WAN on factory router. --no-network + empty repos so apk does not wget indexes.
		installScript = `
set -e
cd /tmp/nd-owrt-pkgs
# prevent apk from refreshing remote indexes (wget "Operation not permitted" without WAN)
: > /tmp/nd-apk-empty-repos
apk add --allow-untrusted --no-network --repositories-file=/tmp/nd-apk-empty-repos ./*.apk
modprobe usb-storage 2>/dev/null || true
modprobe sd_mod 2>/dev/null || true
lsmod | grep -E 'usb_storage|sd_mod' || true
echo INSTALL_DONE
`
	}
	out, err := runSSHOnPort(factorySSHPort(), password, keyPath, user, host, installScript, keyPassphrase)
	fmt.Fprintln(os.Stderr, strings.TrimSpace(out))
	if err != nil {
		return fmt.Errorf("install packages on router: %w (%s)", err, out)
	}
	if !strings.Contains(out, "INSTALL_DONE") {
		return fmt.Errorf("install packages on router: unexpected output: %s", out)
	}
	// verify
	need, note, _ = RouterNeedsUSBStorage(password, keyPath, user, host, keyPassphrase)
	if need {
		return fmt.Errorf("usb-storage still missing after install (%s)", note)
	}
	fmt.Fprintln(os.Stderr, "==> usb-storage ready")
	return nil
}

// scpToRouter copies local file to remote path using existing SSH helpers.
func scpToRouter(password, keyPath, user, host, keyPassphrase, local, remote string) error {
	return runSCPOnPort(factorySSHPort(), password, keyPath, user, host, local, remote, keyPassphrase)
}

// offlineOpenWrtUSBPresets boards we pre-cache during offline-prep (Mac online).
// Deploy always probes the live router and refreshes cache for that exact kernel.
func offlineOpenWrtUSBPresets() []OpenWrtMeta {
	return []OpenWrtMeta{
		{Release: "25.12.5", Target: "ramips/mt76x8", Arch: "mipsel_24kc", Kernel: "6.12.94", PkgMgr: "apk"},
		{Release: "24.10.4", Target: "ramips/mt76x8", Arch: "mipsel_24kc", PkgMgr: "opkg"},
	}
}
