package download

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTLSIPFallbackRetriesAfterHandshakeFailure(t *testing.T) {
	const host = "release-assets.githubusercontent.com"

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: host},
		DNSNames:              []string{host},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &serverKey.PublicKey, serverKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		t.Fatal(err)
	}
	parsedCert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(parsedCert)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	sni := make(chan string, 1)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			sni <- hello.ServerName
			return nil, nil
		},
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "fallback-ok")
	})}
	defer server.Close()
	go func() {
		_ = server.Serve(tls.NewListener(listener, tlsConfig))
	}()

	var attempted []string
	transport := newTransportWithIPFallback(&http.Transport{
		TLSHandshakeTimeout: 2 * time.Second,
		TLSClientConfig:     &tls.Config{RootCAs: roots},
	}, map[string]struct{}{host: {}}, func(ctx context.Context, gotHost string) ([]net.IPAddr, error) {
		if gotHost != host {
			return nil, fmt.Errorf("lookup host = %q, want %q", gotHost, host)
		}
		return []net.IPAddr{
			{IP: net.ParseIP("192.0.2.10")},
			{IP: net.ParseIP("192.0.2.20")},
		}, nil
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		attempted = append(attempted, address)
		if strings.HasPrefix(address, "192.0.2.10:") {
			clientConn, serverConn := net.Pipe()
			go func() {
				defer serverConn.Close()
				// Drain the ClientHello before sending a fatal TLS alert. net.Pipe
				// writes block until the peer reads, so writing first deadlocks
				// the client's handshake.
				buf := make([]byte, 4096)
				if _, readErr := serverConn.Read(buf); readErr != nil {
					return
				}
				_, _ = serverConn.Write([]byte{21, 3, 3, 0, 2, 2, 40})
			}()
			return clientConn, nil
		}
		if strings.HasPrefix(address, "192.0.2.20:") {
			return net.Dial(network, listener.Addr().String())
		}
		return nil, fmt.Errorf("unexpected dial address %q", address)
	})
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	resp, err := client.Get("https://" + host + "/asset")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if resp.StatusCode != http.StatusOK || string(body) != "fallback-ok" {
		t.Fatalf("unexpected response: status=%d body=%q", resp.StatusCode, body)
	}
	if len(attempted) != 2 || attempted[0] != "192.0.2.10:443" || attempted[1] != "192.0.2.20:443" {
		t.Fatalf("unexpected dial sequence: %v", attempted)
	}
	select {
	case gotSNI := <-sni:
		if gotSNI != host {
			t.Fatalf("TLS SNI = %q, want original hostname %q", gotSNI, host)
		}
	case <-time.After(time.Second):
		t.Fatal("TLS server did not receive ClientHello")
	}
}

func TestTLSIPFallbackPreservesBaseDialerForOtherHosts(t *testing.T) {
	baseDialCalls := 0
	fallbackDialCalls := 0
	lookupCalls := 0
	base := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			baseDialCalls++
			clientConn, serverConn := net.Pipe()
			go func() {
				_, _ = serverConn.Write([]byte("not a TLS record"))
				_ = serverConn.Close()
			}()
			return clientConn, nil
		},
	}
	transport := newTransportWithIPFallback(base, map[string]struct{}{
		"release-assets.githubusercontent.com": {},
	}, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		lookupCalls++
		return nil, fmt.Errorf("unexpected DNS lookup for %q", host)
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		fallbackDialCalls++
		return nil, fmt.Errorf("unexpected fallback dial for %q", address)
	})

	_, err := transport.DialTLSContext(context.Background(), "tcp", "github.com:443")
	if err == nil {
		t.Fatal("expected TLS handshake error from test connection")
	}
	if baseDialCalls != 1 {
		t.Fatalf("base dial calls = %d, want 1", baseDialCalls)
	}
	if fallbackDialCalls != 0 {
		t.Fatalf("fallback dial calls = %d, want 0", fallbackDialCalls)
	}
	if lookupCalls != 0 {
		t.Fatalf("DNS lookup calls = %d, want 0", lookupCalls)
	}
}
