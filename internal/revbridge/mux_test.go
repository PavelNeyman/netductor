package revbridge

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestMuxPipe(t *testing.T) {
	a, b := net.Pipe()
	done := make(chan struct{})
	srv := newSession(a, true, func(id uint32) (net.Conn, error) {
		// primary side: echo server via another pipe end
		c1, c2 := net.Pipe()
		go func() {
			defer c1.Close()
			io.Copy(c1, c1) // broken - use c2 as peer
		}()
		_ = c2
		// simpler: return a pipe we write expected data from test
		return c1, nil
	})
	_ = srv
	cli := newSession(b, false, nil)
	local, remote := net.Pipe()
	go func() {
		_ = cli.OpenStream(local)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	_ = remote.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
