package revbridge

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Frame: u32be length | u32be streamID | u8 flag | payload
// flag: 1=open 2=data 3=close 4=open-ack

const (
	flagOpen    = 1
	flagData    = 2
	flagClose   = 3
	flagOpenAck = 4
	maxFrame    = 1 << 20
	streamQ     = 64 // per-stream ordered write queue
)

type streamSlot struct {
	conn     net.Conn
	wch      chan []byte
	closed   atomic.Bool
	closeOnce sync.Once
}

type session struct {
	conn    net.Conn
	mu      sync.Mutex
	streams map[uint32]*streamSlot
	nextID  uint32
	inbound bool
	onOpen  func(id uint32) (net.Conn, error)
	closed  atomic.Bool
	writeMu sync.Mutex
	ackWait map[uint32]chan struct{}
}

func newSession(conn net.Conn, inbound bool, onOpen func(uint32) (net.Conn, error)) *session {
	s := &session{
		conn:    conn,
		streams: make(map[uint32]*streamSlot),
		inbound: inbound,
		onOpen:  onOpen,
		nextID:  1,
		ackWait: make(map[uint32]chan struct{}),
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

func (s *session) attachStream(id uint32, c net.Conn) *streamSlot {
	slot := &streamSlot{conn: c, wch: make(chan []byte, streamQ)}
	s.mu.Lock()
	s.streams[id] = slot
	s.mu.Unlock()
	go s.streamWriter(id, slot)
	return slot
}

func (s *session) streamWriter(id uint32, slot *streamSlot) {
	for buf := range slot.wch {
		if slot.closed.Load() {
			continue
		}
		if _, err := slot.conn.Write(buf); err != nil {
			slot.closed.Store(true)
			s.mu.Lock()
			delete(s.streams, id)
			s.mu.Unlock()
			_ = slot.conn.Close()
			return
		}
	}
}

func (s *session) dropStream(id uint32) {
	s.mu.Lock()
	slot := s.streams[id]
	delete(s.streams, id)
	s.mu.Unlock()
	if slot == nil {
		return
	}
	slot.closed.Store(true)
	slot.closeOnce.Do(func() {
		close(slot.wch)
		_ = slot.conn.Close()
	})
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
		case flagOpenAck:
			s.mu.Lock()
			ch := s.ackWait[id]
			if ch != nil {
				select {
				case <-ch:
				default:
					close(ch)
				}
			}
			s.mu.Unlock()
		case flagData:
			s.mu.Lock()
			slot := s.streams[id]
			s.mu.Unlock()
			if slot == nil || len(payload) == 0 {
				continue
			}
			buf := append([]byte(nil), payload...)
			select {
			case slot.wch <- buf:
			default:
				// queue full — drop stream rather than stall mux or reorder
				s.dropStream(id)
				_ = s.writeFrame(id, flagClose, nil)
			}
		case flagClose:
			s.dropStream(id)
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
	s.attachStream(id, c)
	_ = s.writeFrame(id, flagOpenAck, nil)
	s.pipeConn(id, c)
}

func (s *session) OpenStream(local net.Conn) error {
	id := atomic.AddUint32(&s.nextID, 1)
	ack := make(chan struct{})
	s.mu.Lock()
	s.ackWait[id] = ack
	s.mu.Unlock()
	s.attachStream(id, local)
	if err := s.writeFrame(id, flagOpen, nil); err != nil {
		s.mu.Lock()
		delete(s.ackWait, id)
		s.mu.Unlock()
		s.dropStream(id)
		return err
	}
	select {
	case <-ack:
	case <-time.After(15 * time.Second):
		s.mu.Lock()
		delete(s.ackWait, id)
		s.mu.Unlock()
		s.dropStream(id)
		_ = s.writeFrame(id, flagClose, nil)
		return fmt.Errorf("revbridge open ack timeout stream %d", id)
	}
	s.mu.Lock()
	delete(s.ackWait, id)
	s.mu.Unlock()
	s.pipeConn(id, local)
	return nil
}

func (s *session) pipeConn(id uint32, c net.Conn) {
	defer func() {
		_ = s.writeFrame(id, flagClose, nil)
		s.dropStream(id)
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
	ids := make([]uint32, 0, len(s.streams))
	for id := range s.streams {
		ids = append(ids, id)
	}
	acks := s.ackWait
	s.ackWait = map[uint32]chan struct{}{}
	s.mu.Unlock()
	for _, id := range ids {
		s.dropStream(id)
	}
	for _, ch := range acks {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
}
