package revbridge

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Simple stream mux over one net.Conn.
// Frame: u32be length (payload only) | u32be streamID | u8 flag | payload
// flag: 1=open 2=data 3=close

const (
	flagOpen  = 1
	flagData  = 2
	flagClose = 3
	maxFrame  = 1 << 20
)

type session struct {
	conn     net.Conn
	mu       sync.Mutex
	streams  map[uint32]net.Conn
	nextID   uint32
	inbound  bool // true = server (secondary)
	onOpen   func(id uint32) (net.Conn, error)
	closed   atomic.Bool
	writeMu  sync.Mutex
}

func newSession(conn net.Conn, inbound bool, onOpen func(uint32) (net.Conn, error)) *session {
	s := &session{
		conn:    conn,
		streams: make(map[uint32]net.Conn),
		inbound: inbound,
		onOpen:  onOpen,
		nextID:  1,
	}
	go s.readLoop()
	return s
}

func (s *session) writeFrame(id uint32, flag byte, payload []byte) error {
	if s.closed.Load() {
		return io.ErrClosedPipe
	}
	if len(payload) > maxFrame {
		return fmt.Errorf("frame too large")
	}
	hdr := make([]byte, 9)
	binary.BigEndian.PutUint32(hdr[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(hdr[4:8], id)
	hdr[8] = flag
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := s.conn.Write(hdr); err != nil {
		return err
	}
	if len(payload) > 0 {
		_, err := s.conn.Write(payload)
		return err
	}
	return nil
}

func (s *session) readLoop() {
	defer s.Close()
	hdr := make([]byte, 9)
	for {
		if _, err := io.ReadFull(s.conn, hdr); err != nil {
			return
		}
		n := binary.BigEndian.Uint32(hdr[0:4])
		id := binary.BigEndian.Uint32(hdr[4:8])
		flag := hdr[8]
		if n > maxFrame {
			return
		}
		var payload []byte
		if n > 0 {
			payload = make([]byte, n)
			if _, err := io.ReadFull(s.conn, payload); err != nil {
				return
			}
		}
		switch flag {
		case flagOpen:
			if !s.inbound {
				continue
			}
			go s.handleOpen(id)
		case flagData:
			s.mu.Lock()
			c := s.streams[id]
			s.mu.Unlock()
			if c != nil && len(payload) > 0 {
				_, _ = c.Write(payload)
			}
		case flagClose:
			s.mu.Lock()
			c := s.streams[id]
			delete(s.streams, id)
			s.mu.Unlock()
			if c != nil {
				_ = c.Close()
			}
		}
	}
}

func (s *session) handleOpen(id uint32) {
	if s.onOpen == nil {
		_ = s.writeFrame(id, flagClose, nil)
		return
	}
	c, err := s.onOpen(id)
	if err != nil || c == nil {
		_ = s.writeFrame(id, flagClose, nil)
		return
	}
	s.mu.Lock()
	s.streams[id] = c
	s.mu.Unlock()
	s.pipeConn(id, c)
}

// OpenStream is used by secondary local accept: notify primary to dial target.
func (s *session) OpenStream(local net.Conn) error {
	id := atomic.AddUint32(&s.nextID, 1)
	s.mu.Lock()
	s.streams[id] = local
	s.mu.Unlock()
	if err := s.writeFrame(id, flagOpen, nil); err != nil {
		s.mu.Lock()
		delete(s.streams, id)
		s.mu.Unlock()
		return err
	}
	s.pipeConn(id, local)
	return nil
}

func (s *session) pipeConn(id uint32, c net.Conn) {
	defer func() {
		_ = s.writeFrame(id, flagClose, nil)
		s.mu.Lock()
		delete(s.streams, id)
		s.mu.Unlock()
		_ = c.Close()
	}()
	buf := make([]byte, 32*1024)
	for {
		n, err := c.Read(buf)
		if n > 0 {
			if werr := s.writeFrame(id, flagData, buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (s *session) Close() {
	if s.closed.Swap(true) {
		return
	}
	_ = s.conn.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, c := range s.streams {
		_ = c.Close()
		delete(s.streams, id)
	}
}
