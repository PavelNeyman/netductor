package download

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// installTLSIPFallback makes TLS establishment part of each address attempt.
// net/http's DialTLSContext contract requires the returned connection to have
// completed its handshake, which lets us retry when TCP succeeds but TLS fails.
func installTLSIPFallback(transport *http.Transport, allow map[string]struct{}, lookup ipLookupFunc, dial contextDialFunc) {
	transport.DialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("download: split TLS address %q: %w", address, err)
		}

		_, allowed := allow[strings.ToLower(strings.TrimSuffix(host, "."))]
		targets := []string{address}
		if allowed {
			ips, lookupErr := lookup(ctx, host)
			if lookupErr != nil {
				return nil, fmt.Errorf("download: resolve %s: %w", host, lookupErr)
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("download: no addresses returned for %s", host)
			}
			targets = make([]string, 0, len(ips))
			for _, ip := range ips {
				targets = append(targets, net.JoinHostPort(ip.String(), port))
			}
		}

		var failures []error
		for _, target := range targets {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			conn, dialErr := dial(ctx, network, target)
			if dialErr != nil {
				failures = append(failures, fmt.Errorf("%s: TCP: %w", target, dialErr))
				continue
			}

			cfg := &tls.Config{}
			if transport.TLSClientConfig != nil {
				cfg = transport.TLSClientConfig.Clone()
			}
			if cfg.ServerName == "" {
				cfg.ServerName = strings.TrimSuffix(host, ".")
			}
			if cfg.NextProtos == nil {
				if transport.ForceAttemptHTTP2 {
					cfg.NextProtos = []string{"h2", "http/1.1"}
				} else {
					cfg.NextProtos = []string{"http/1.1"}
				}
			}

			tlsConn := tls.Client(conn, cfg)
			handshakeCtx := ctx
			cancel := func() {}
			if transport.TLSHandshakeTimeout > 0 {
				handshakeCtx, cancel = context.WithTimeout(ctx, transport.TLSHandshakeTimeout)
			}
			handshakeErr := tlsConn.HandshakeContext(handshakeCtx)
			cancel()
			if handshakeErr == nil {
				return tlsConn, nil
			}
			_ = conn.Close()
			failures = append(failures, fmt.Errorf("%s: TLS handshake: %w", target, handshakeErr))
			if !allowed {
				return nil, errors.Join(failures...)
			}
		}
		return nil, fmt.Errorf("download: all addresses for %s failed: %w", host, errors.Join(failures...))
	}
}

// Keep the time import tied to the explicit timeout semantics above.
var _ = time.Second
