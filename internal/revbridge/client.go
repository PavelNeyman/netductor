package revbridge

import (
	"fmt"
	"os"
	"net"
	"sync"
	"time"
)

// Client runs on Primary: dials Secondary public ports and services local targets per stream.
type Client struct {
	host  string
	plans []PortPlan
	stop  chan struct{}
	wg    sync.WaitGroup
}

func NewClient(secondaryHost string, kinds []Kind) *Client {
	var plans []PortPlan
	for _, k := range kinds {
		if k == KindWG {
			continue
		}
		if p, ok := PlanFor(k); ok {
			plans = append(plans, p)
		}
	}
	return &Client{host: secondaryHost, plans: plans, stop: make(chan struct{})}
}

func (c *Client) Start() {
	for _, p := range c.plans {
		p := p
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.loopKind(p)
		}()
	}
}

func (c *Client) loopKind(p PortPlan) {
	backoff := time.Second
	for {
		select {
		case <-c.stop:
			return
		default:
		}
		err := c.sessionOnce(p)
		if err != nil {
			select {
			case <-c.stop:
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (c *Client) sessionOnce(p PortPlan) error {
	select {
	case <-c.stop:
		return fmt.Errorf("stopped")
	default:
	}
	addr := fmt.Sprintf("%s:%d", c.host, p.PublicPort)
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "revbridge: dial %s kind=%s: %v\n", addr, p.Kind, err)
		return err
	}
	uc := net.Conn(conn)
	if p.Transport == "ws" {
		uc, err = dialClientWS(conn, c.host)
		if err != nil {
			fmt.Fprintf(os.Stderr, "revbridge: ws upgrade %s: %v\n", addr, err)
			_ = conn.Close()
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "revbridge: session up kind=%s -> %s dial=%s\n", p.Kind, addr, p.PrimaryDial)
	// Block until session dies (readLoop → Close).
	done := make(chan struct{})
	sess := newSession(uc, true, func(id uint32) (net.Conn, error) {
		return net.DialTimeout("tcp", p.PrimaryDial, 10*time.Second)
	})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			if sess.closed.Load() {
				close(done)
				return
			}
			select {
			case <-c.stop:
				sess.Close()
				close(done)
				return
			case <-ticker.C:
			}
		}
	}()
	<-done
	return fmt.Errorf("session ended")
}

func (c *Client) Stop() {
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
	c.wg.Wait()
}
