package vpn

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sub profiles control what the subscription URL serves (client auto-refresh).
const (
	SubSecondary = "secondary" // default for normal users — RU entry only
	SubPrimary   = "primary"   // abroad core only (vacation RU-exit via primary→… depends on routing)
	SubBoth      = "both"      // operator / explicit dual list
)

func NormalizeSubProfile(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case SubPrimary, "core":
		return SubPrimary
	case SubBoth, "all", "dual":
		return SubBoth
	default:
		return SubSecondary
	}
}

// EffectiveSubProfile: empty → secondary; operator defaults to both if unset historically dual.
func EffectiveSubProfile(u *UserRecord) string {
	if u == nil {
		return SubSecondary
	}
	if p := NormalizeSubProfile(u.SubProfile); u.SubProfile != "" {
		return p
	}
	if strings.EqualFold(u.Name, "operator") {
		return SubBoth
	}
	return SubSecondary
}

// SetSubProfile stores profile and refreshes subscription files (links UUID unchanged).
func SetSubProfile(name, profile string) error {
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	u := findUser(r, name)
	if u == nil {
		return fmt.Errorf("user not found")
	}
	u.SubProfile = NormalizeSubProfile(profile)
	if err := writeRegistry(r); err != nil {
		return err
	}
	return WriteSubscriptionFiles(name)
}

// SubscriptionBody newline-separated VLESS URIs according to user sub_profile.
func SubscriptionBody(name string) (string, error) {
	r, err := loadRegistry()
	if err != nil {
		return "", err
	}
	u := findUser(r, name)
	if u == nil {
		return "", fmt.Errorf("user not found")
	}
	nl := string([]byte{10})
	sec := strings.TrimSpace(PreferredVLESSLink(name, u.UUID))
	core := strings.TrimSpace(VLESSLink(name, u.UUID))
	var parts []string
	switch EffectiveSubProfile(u) {
	case SubPrimary:
		if core != "" {
			parts = append(parts, core)
		}
	case SubBoth:
		if sec != "" {
			parts = append(parts, sec)
		}
		if core != "" && core != sec {
			parts = append(parts, core)
		}
	default: // secondary
		if sec != "" {
			parts = append(parts, sec)
		} else if core != "" {
			// fallback if secondary offline — still one URI
			parts = append(parts, core)
		}
	}
	return strings.Join(parts, nl) + nl, nil
}

func SubscriptionBase64(name string) (string, error) {
	body, err := SubscriptionBody(name)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(body)), nil
}

func WriteSubscriptionFiles(name string) error {
	body, err := SubscriptionBody(name)
	if err != nil {
		return err
	}
	dir := filepath.Join(Clients(), name)
	_ = os.MkdirAll(dir, 0o700)
	nl := string([]byte{10})
	_ = os.WriteFile(filepath.Join(dir, "subscription.txt"), []byte(body), 0o600)
	b64 := base64.StdEncoding.EncodeToString([]byte(body))
	_ = os.WriteFile(filepath.Join(dir, "subscription.b64"), []byte(b64+nl), 0o600)
	return nil
}

// SubProfileOf returns effective profile for UI.
func SubProfileOf(name string) string {
	r, err := loadRegistry()
	if err != nil {
		return SubSecondary
	}
	return EffectiveSubProfile(findUser(r, name))
}

// CycleSubProfile secondary → primary → both → secondary.
func CycleSubProfile(name string) (string, error) {
	cur := SubProfileOf(name)
	next := SubSecondary
	switch cur {
	case SubSecondary:
		next = SubPrimary
	case SubPrimary:
		next = SubBoth
	default:
		next = SubSecondary
	}
	if err := SetSubProfile(name, next); err != nil {
		return "", err
	}
	return next, nil
}
