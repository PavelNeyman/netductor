package mikrotik

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnOpts one-shot SSH credentials (not persisted by netductor).
type ConnOpts struct {
	Host       string
	User       string
	Password   string
	PrivateKey []byte
	Port       int
}

func (c ConnOpts) port() int {
	if c.Port <= 0 {
		return 22
	}
	return c.Port
}

func (c ConnOpts) user() string {
	if c.User == "" {
		return "admin"
	}
	return c.User
}

func dial(c ConnOpts) (*ssh.Client, error) {
	if c.Host == "" {
		return nil, fmt.Errorf("host required")
	}
	var auth []ssh.AuthMethod
	if len(c.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(c.PrivateKey)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if c.Password != "" {
		auth = append(auth, ssh.Password(c.Password))
	}
	if len(auth) == 0 {
		return nil, fmt.Errorf("password or private key required")
	}
	cfg := &ssh.ClientConfig{
		User:            c.user(),
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
	}
	return ssh.Dial("tcp", net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.port())), cfg)
}

// Run runs a single RouterOS command; returns combined output.
func Run(c ConnOpts, command string) (string, error) {
	client, err := dial(c)
	if err != nil {
		return "", err
	}
	defer client.Close()
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(command)
	return string(out), err
}

// PushRSC applies a multi-line RSC: skips empty/# lines, runs each command separately.
func PushRSC(host, user, password string, privateKey []byte, rsc string, port int) error {
	c := ConnOpts{Host: host, User: user, Password: password, PrivateKey: privateKey, Port: port}
	if rsc == "" {
		return fmt.Errorf("rsc required")
	}
	client, err := dial(c)
	if err != nil {
		return err
	}
	defer client.Close()
	nl := string([]byte{10})
	crlf := string([]byte{13, 10})
	body := strings.ReplaceAll(rsc, crlf, nl)
	var errs []string
	for _, line := range strings.Split(body, nl) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sess, err := client.NewSession()
		if err != nil {
			return err
		}
		out, err := sess.CombinedOutput(line)
		sess.Close()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v %s", line, err, string(out)))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("partial failures: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Identity reads /system identity.
func Identity(c ConnOpts) (string, error) {
	return Run(c, "/system identity print")
}

// Resource reads CPU/memory snapshot.
func Resource(c ConnOpts) (string, error) {
	return Run(c, "/system resource print")
}

// Routes prints IP routes.
func Routes(c ConnOpts) (string, error) {
	return Run(c, "/ip route print")
}

// PingFromMT runs ping from router.
func PingFromMT(c ConnOpts, target string) (string, error) {
	if target == "" {
		target = "1.1.1.1"
	}
	return Run(c, fmt.Sprintf("/ping %s count=3", target))
}
