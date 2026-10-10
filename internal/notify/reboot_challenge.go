package notify

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

type rebootChal struct {
	Code   string `json:"code"`
	Target string `json:"target"` // "primary" or secondary id
	Exp    int64  `json:"exp"`
}

func rebootChalPath() string {
	return filepath.Join(paths.StateDir(), "reboot-challenge.json")
}

// BeginRebootChallenge stores a 4-digit code valid for 3 minutes.
func BeginRebootChallenge(target string) (code string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	if err != nil {
		return "", err
	}
	code = fmt.Sprintf("%04d", n.Int64()+1000)
	c := rebootChal{Code: code, Target: target, Exp: time.Now().Add(3 * time.Minute).Unix()}
	_ = os.MkdirAll(filepath.Dir(rebootChalPath()), 0o700)
	b, _ := json.Marshal(c)
	return code, os.WriteFile(rebootChalPath(), b, 0o600)
}

// ConsumeRebootChallenge validates code+target and deletes the file (one-shot).
func ConsumeRebootChallenge(target, code string) error {
	b, err := os.ReadFile(rebootChalPath())
	if err != nil {
		return fmt.Errorf("no pending reboot challenge — request again")
	}
	var c rebootChal
	if json.Unmarshal(b, &c) != nil {
		return fmt.Errorf("invalid challenge file")
	}
	_ = os.Remove(rebootChalPath())
	if time.Now().Unix() > c.Exp {
		return fmt.Errorf("challenge expired — request again")
	}
	if c.Target != target || c.Code != strings.TrimSpace(code) {
		return fmt.Errorf("code or target mismatch")
	}
	return nil
}
