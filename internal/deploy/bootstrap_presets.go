package deploy

import (
	"fmt"
	"strings"
)

// BootstrapPreset is a fixed OpenWrt package set installed only via edge.bootstrap
// (never via probe or free-form lists). Deploy does not install these packages.
type BootstrapPreset struct {
	ID              string   `json:"id"`
	TitleEN         string   `json:"title_en"`
	TitleRU         string   `json:"title_ru"`
	DescriptionEN   string   `json:"description_en,omitempty"`
	DescriptionRU   string   `json:"description_ru,omitempty"`
	Packages        []string `json:"packages"`
	RequiresUSBSlot bool     `json:"requires_usb_slot,omitempty"`
	// MinOverlayFreeKB: skip if overlay free space is known and below this (0 = no gate).
	MinOverlayFreeKB int64 `json:"min_overlay_free_kb,omitempty"`
	// MinUSBSizeKB: when a USB disk is visible, require at least this size (0 = no gate).
	MinUSBSizeKB int64 `json:"min_usb_size_kb,omitempty"`
}

// PresetStatus is probe/bootstrap UI state for one preset.
type PresetStatus struct {
	ID            string   `json:"id"`
	TitleEN       string   `json:"title_en"`
	TitleRU       string   `json:"title_ru"`
	DescriptionEN string   `json:"description_en,omitempty"`
	DescriptionRU string   `json:"description_ru,omitempty"`
	Packages      []string `json:"packages"`
	Installed     bool     `json:"installed"`
	Missing       []string `json:"missing,omitempty"`
	Selectable    bool     `json:"selectable"`
	ReasonEN      string   `json:"reason_en,omitempty"`
	ReasonRU      string   `json:"reason_ru,omitempty"`
	USBVisible    bool     `json:"usb_visible,omitempty"`
}

var builtinBootstrapPresets = []BootstrapPreset{
	{
		ID:              "usb-storage",
		TitleEN:         "USB mass storage",
		TitleRU:         "USB-накопитель",
		DescriptionEN:   "kmod + e2fsprogs so the router can see a USB stick and prepare extroot/overlay. Required before overlay/extroot deploy on stock OpenWrt.",
		DescriptionRU:   "kmod и e2fsprogs, чтобы роутер увидел USB и можно было сделать extroot/overlay. Нужен до deploy с overlay на стоковом OpenWrt.",
		Packages:         nil, // filled in BootstrapPresets()
		RequiresUSBSlot:  true,
		MinOverlayFreeKB: 4096, // need room to unpack kmods on flash
	},
}

// BootstrapPresets returns the built-in preset catalog (copy).
func BootstrapPresets() []BootstrapPreset {
	out := make([]BootstrapPreset, len(builtinBootstrapPresets))
	copy(out, builtinBootstrapPresets)
	for i := range out {
		if out[i].ID == "usb-storage" {
			out[i].Packages = DefaultUSBPkgNames()
		}
	}
	return out
}

// BootstrapPresetByID returns a preset or error.
func BootstrapPresetByID(id string) (BootstrapPreset, error) {
	id = strings.TrimSpace(strings.ToLower(id))
	for _, p := range BootstrapPresets() {
		if p.ID == id {
			return p, nil
		}
	}
	return BootstrapPreset{}, fmt.Errorf("unknown bootstrap preset %q (known: usb-storage)", id)
}

// BootstrapEvalOpts optional size gates from last probe.
type BootstrapEvalOpts struct {
	USBVisible     bool
	OverlayFreeKB  int64 // 0 = unknown
	USBSizeKB      int64 // largest USB disk; 0 = unknown / none
}

// EvaluateBootstrapPresets builds UI status from installed packages and optional size facts.
func EvaluateBootstrapPresets(installed map[string]bool, opt BootstrapEvalOpts) []PresetStatus {
	var out []PresetStatus
	for _, p := range BootstrapPresets() {
		st := PresetStatus{
			ID:            p.ID,
			TitleEN:       p.TitleEN,
			TitleRU:       p.TitleRU,
			DescriptionEN: p.DescriptionEN,
			DescriptionRU: p.DescriptionRU,
			Packages:      append([]string(nil), p.Packages...),
			USBVisible:    opt.USBVisible,
		}
		var missing []string
		for _, pkg := range p.Packages {
			if !packageSatisfied(installed, pkg) {
				missing = append(missing, pkg)
			}
		}
		st.Missing = missing
		st.Installed = len(missing) == 0
		// size gates
		if p.MinOverlayFreeKB > 0 && opt.OverlayFreeKB > 0 && opt.OverlayFreeKB < p.MinOverlayFreeKB {
			st.Selectable = false
			st.ReasonEN = fmt.Sprintf("overlay free %d KB < need %d KB", opt.OverlayFreeKB, p.MinOverlayFreeKB)
			st.ReasonRU = fmt.Sprintf("свободно overlay %d KB < нужно %d KB", opt.OverlayFreeKB, p.MinOverlayFreeKB)
			out = append(out, st)
			continue
		}
		if p.MinUSBSizeKB > 0 && opt.USBVisible && opt.USBSizeKB > 0 && opt.USBSizeKB < p.MinUSBSizeKB {
			st.Selectable = false
			st.ReasonEN = fmt.Sprintf("USB %d KB < need %d KB", opt.USBSizeKB, p.MinUSBSizeKB)
			st.ReasonRU = fmt.Sprintf("USB %d KB < нужно %d KB", opt.USBSizeKB, p.MinUSBSizeKB)
			out = append(out, st)
			continue
		}
		switch {
		case st.Installed && p.ID == "usb-storage" && !opt.USBVisible:
			st.Selectable = true
			st.ReasonEN = "packages installed; plug USB stick and re-probe if disk not listed"
			st.ReasonRU = "пакеты стоят; вставьте USB и сделайте probe, если диск не виден"
		case st.Installed:
			st.Selectable = false
			st.ReasonEN = "already installed"
			st.ReasonRU = "уже установлено"
		default:
			st.Selectable = true
		}
		out = append(out, st)
	}
	return out
}

// EvalOptsFromFacts builds BootstrapEvalOpts from edge.DeviceFacts storage.
func EvalOptsFromFacts(usbVisible bool, overlayFreeKB, usbSizeKB int64) BootstrapEvalOpts {
	return BootstrapEvalOpts{USBVisible: usbVisible, OverlayFreeKB: overlayFreeKB, USBSizeKB: usbSizeKB}
}

func packageSatisfied(installed map[string]bool, pkg string) bool {
	if installed[pkg] {
		return true
	}
	for _, alt := range usbPkgAlternates(pkg) {
		if installed[alt] {
			return true
		}
	}
	if strings.HasPrefix(pkg, "kmod-") {
		mod := strings.ReplaceAll(strings.TrimPrefix(pkg, "kmod-"), "-", "_")
		if installed["mod:"+mod] || installed["mod:"+strings.TrimPrefix(pkg, "kmod-")] {
			return true
		}
	}
	return false
}
