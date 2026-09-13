package mikrotik

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// PushRSC applies a RouterOS script over SSH (password or key).
// Safe default: does not enable password permanently; caller supplies one-shot credentials.
func PushRSC(host, user, password string, privateKey []byte, rsc string, port int) error {
	if host == "" || rsc == "" {
		return fmt.Errorf("host and rsc required")
	}
	if port <= 0 {
		port = 22
	}
	if user == "" {
		user = "admin"
	}
	var auth []ssh.AuthMethod
	if len(privateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			return err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if password != "" {
		auth = append(auth, ssh.Password(password))
	}
	if len(auth) == 0 {
		return fmt.Errorf("password or private key required")
	}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	// RouterOS accepts commands line by line
	cmd := strings.ReplaceAll(rsc, "\r\n", "\n")
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}
