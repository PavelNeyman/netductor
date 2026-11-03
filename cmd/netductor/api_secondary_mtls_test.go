package main

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecondaryAgentCertificateMatchesTokenIdentity(t *testing.T) {
	const id = "secondary-0123456789abcdef"

	tests := []struct {
		name string
		req  *http.Request
		want bool
	}{
		{name: "missing TLS", req: httptest.NewRequest(http.MethodGet, "/", nil), want: false},
		{
			name: "missing peer certificate",
			req: &http.Request{TLS: &tls.ConnectionState{}},
			want: false,
		},
		{
			name: "matching agent identity",
			req: &http.Request{TLS: &tls.ConnectionState{PeerCertificates: []*x509.Certificate{
				{Subject: pkix.Name{CommonName: "netductor-agent-" + id}},
			}}},
			want: true,
		},
		{
			name: "different agent identity",
			req: &http.Request{TLS: &tls.ConnectionState{PeerCertificates: []*x509.Certificate{
				{Subject: pkix.Name{CommonName: "netductor-agent-secondary-fedcba9876543210"}},
			}}},
			want: false,
		},
		{
			name: "default shared certificate",
			req: &http.Request{TLS: &tls.ConnectionState{PeerCertificates: []*x509.Certificate{
				{Subject: pkix.Name{CommonName: "netductor-agent-client"}},
			}}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := secondaryAgentCertificateMatches(tt.req, id); got != tt.want {
				t.Fatalf("secondaryAgentCertificateMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
