package revbridge

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestMuxOpenAck(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(io.Discard, c)
			}(c)
		}
	}()
	addr := ln.Addr().String()

	a, b := net.Pipe()
	_ = newSession(a, true, func(id uint32) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 2*time.Second)
	})
	sec := newSession(b, false, nil)

	local, peer := net.Pipe()
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, _ = peer.Write([]byte("ping"))
		time.Sleep(50 * time.Millisecond)
		_ = peer.Close()
	}()

	done := make(chan error, 1)
	go func() {
		done <- sec.OpenStream(local)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("OpenStream: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OpenStream timeout")
	}
}
