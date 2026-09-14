package vpn

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SubscriptionBody returns newline-separated primary URIs (relay + core), no HY2.
func SubscriptionBody(name string) (string, error) {
	r, err := loadRegistry()
	if err != nil {
		return "", err
	}
	u := findUser(r, name)
	if u == nil {
		return "", fmt.Errorf("user not found")
	}
	vless := PreferredVLESSLink(name, u.UUID)
	core := VLESSLink(name, u.UUID)
	nl := string([]byte{10})
	body := strings.TrimSpace(vless) + nl + strings.TrimSpace(core) + nl
	return body, nil
}

// SubscriptionBase64 is the Shadowrocket-friendly subscription document (base64 of URI list).
func SubscriptionBase64(name string) (string, error) {
	body, err := SubscriptionBody(name)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(body)), nil
}

// WriteSubscriptionFiles refreshes subscription.txt + subscription.b64 under client dir.
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
