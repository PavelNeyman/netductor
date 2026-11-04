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

const maxTLSIPFallbackStageTimeout = 5 * time.Second

// installTLSIPFallback makes TLS establishment part of each address attempt.
// net/http's DialTLSContext contract requires the returned connection to have
// completed its handshake, which lets us retry when TCP succeeds but TLS fails.
func installTLSIPFallback(transport *http.Transport, allow map[string]struct{}, lookup ipLookupFunc, dial, normalDial contextDialFunc) {
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
		for i, target := range targets {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			var conn net.Conn
			if allowed {
				dialTimeout := tlsIPFallbackStageTimeout(ctx, len(targets)-i, 0)
				if dialTimeout <= 0 {
					return nil, context.DeadlineExceeded
				}
				dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
				conn, err = dial(dialCtx, network, target)
				cancel()
			} else {
				conn, err = normalDial(ctx, network, target)
			}
			if err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				failures = append(failures, fmt.Errorf("%s: TCP: %w", target, err))
				if !allowed {
					return nil, errors.Join(failures...)
				}
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

			handshakeCtx := ctx
			cancelHandshake := func() {}
			if allowed {
				handshakeTimeout := tlsIPFallbackStageTimeout(ctx, len(targets)-i, transport.TLSHandshakeTimeout)
				if handshakeTimeout <= 0 {
					_ = conn.Close()
					return nil, context.DeadlineExceeded
				}
				handshakeCtx, cancelHandshake = context.WithTimeout(ctx, handshakeTimeout)
			} else if transport.TLSHandshakeTimeout > 0 {
				handshakeCtx, cancelHandshake = context.WithTimeout(ctx, transport.TLSHandshakeTimeout)
			}

			tlsConn := tls.Client(conn, cfg)
			handshakeErr := tlsConn.HandshakeContext(handshakeCtx)
			cancelHandshake()
			if handshakeErr == nil {
				return tlsConn, nil
			}
			_ = conn.Close()
			failures = append(failures, fmt.Errorf("%s: TLS handshake: %w", target, handshakeErr))
			if !allowed {
				return nil, errors.Join(failures...)
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		return nil, fmt.Errorf("download: all addresses for %s failed: %w", host, errors.Join(failures...))
	}
}

// tlsIPFallbackStageTimeout bounds TCP and TLS separately while sharing the
// request's remaining deadline fairly across all remaining addresses.
func tlsIPFallbackStageTimeout(ctx context.Context, remainingAddresses int, configured time.Duration) time.Duration {
	timeout := maxTLSIPFallbackStageTimeout
	if configured > 0 && configured < timeout {
		timeout = configured
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0
		}
		fairShare := remaining / time.Duration(remainingAddresses*2)
		if fairShare < timeout {
			timeout = fairShare
		}
	}
	return timeout
}
