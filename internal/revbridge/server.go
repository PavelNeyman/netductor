package revbridge

import (
	"fmt"
	"os"
	"net"
	"strings"
	"sync"
	"time"
)

// Server runs on Secondary:
//   - public port: accepts Primary reverse session (TCP or WS)
//   - local 127.0.0.1 port: accepts sing-box outbound; opens mux stream toward Primary
type Server struct {
	plans     []PortPlan
	allowFrom string
	mu        sync.Mutex
	ln        []net.Listener
	stop      chan struct{}
}

var (
	sessMu   sync.Mutex
	sessions = map[Kind]*session{}
)

func NewServer(kinds []Kind) *Server {
	var plans []PortPlan
	for _, k := range kinds {
		if k == KindWG {
			continue
		}
		if p, ok := PlanFor(k); ok {
			plans = append(plans, p)
		}
	}
	return &Server{plans: plans, stop: make(chan struct{})}
}

func (s *Server) SetAllowFrom(ip string) {
	s.allowFrom = strings.TrimSpace(ip)
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var started int
	var lastErr error
	for _, p := range s.plans {
		pln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.PublicPort))
		if err != nil {
			lastErr = fmt.Errorf("revbridge public :%d: %w", p.PublicPort, err)
			fmt.Fprintln(os.Stderr, lastErr)
			continue
		}
		s.ln = append(s.ln, pln)
		go s.acceptPrimary(pln, p)

		lln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p.LocalPort))
		if err != nil {
			_ = pln.Close()
			lastErr = fmt.Errorf("revbridge local :%d: %w", p.LocalPort, err)
			fmt.Fprintln(os.Stderr, lastErr)
			continue
		}
		s.ln = append(s.ln, lln)
		go s.acceptLocal(lln, p)
		started++
		fmt.Fprintf(os.Stderr, "revbridge: listen kind=%s public=:%d local=127.0.0.1:%d\n", p.Kind, p.PublicPort, p.LocalPort)
	}
	if started == 0 {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("revbridge: no plans to listen")
	}
	return nil
}

func (s *Server) acceptPrimary(ln net.Listener, p PortPlan) {
	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-s.stop:
				return
			default:
				time.Sleep(200 * time.Millisecond)
				continue
			}
		}
		go s.handlePrimary(c, p)
	}
}

func (s *Server) handlePrimary(c net.Conn, p PortPlan) {
	if s.allowFrom != "" {
		host, _, err := net.SplitHostPort(c.RemoteAddr().String())
		if err != nil {
			host = c.RemoteAddr().String()
		}
		if host != s.allowFrom {
			fmt.Fprintf(os.Stderr, "revbridge: reject %s kind=%s (allowFrom=%s)\n", host, p.Kind, s.allowFrom)
			_ = c.Close()
			return
		}
	}
	fmt.Fprintf(os.Stderr, "revbridge: primary session kind=%s from %s\n", p.Kind, c.RemoteAddr().String())
	uc := net.Conn(c)
	var err error
	if p.Transport == "ws" {
		uc, err = upgradeServerWS(c)
		if err != nil {
			_ = c.Close()
			return
		}
	}
	sess := newSession(uc, false, nil)
	sessMu.Lock()
	if old := sessions[p.Kind]; old != nil {
		old.Close()
	}
	sessions[p.Kind] = sess
	sessMu.Unlock()
}

func (s *Server) acceptLocal(ln net.Listener, p PortPlan) {
	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-s.stop:
				return
			default:
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
		go func(c net.Conn) {
			sessMu.Lock()
			sess := sessions[p.Kind]
			sessMu.Unlock()
			if sess == nil {
				fmt.Fprintf(os.Stderr, "revbridge: local dial kind=%s but no primary session\n", p.Kind)
				_ = c.Close()
				return
			}
			if err := sess.OpenStream(c); err != nil {
				fmt.Fprintf(os.Stderr, "revbridge: OpenStream kind=%s: %v\n", p.Kind, err)
				_ = c.Close()
			}
		}(c)
	}
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *Server) stopLocked() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	for _, ln := range s.ln {
		_ = ln.Close()
	}
	s.ln = nil
	sessMu.Lock()
	for k, sess := range sessions {
		sess.Close()
		delete(sessions, k)
	}
	sessMu.Unlock()
}
