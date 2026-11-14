package revbridge

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Client runs on Primary: dials Secondary public ports and services local targets per stream.
type Client struct {
	host  string // secondary IP
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
	addr := fmt.Sprintf("%s:%d", c.host, p.PublicPort)
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return err
	}
	uc := net.Conn(conn)
	if p.Transport == "ws" {
		uc, err = dialClientWS(conn, c.host)
		if err != nil {
			_ = conn.Close()
			return err
		}
	}
	done := make(chan struct{})
	sess := newSession(uc, true, func(id uint32) (net.Conn, error) {
		return net.DialTimeout("tcp", p.PrimaryDial, 10*time.Second)
	})
	go func() {
		// Wait until session dies (readLoop closes)
		for !sess.closed.Load() {
			time.Sleep(500 * time.Millisecond)
			select {
			case <-c.stop:
				sess.Close()
				close(done)
				return
			default:
			}
		}
		close(done)
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
