package revbridge

import "strings"

// Kind is a reverse underlay type Primary dials toward Secondary.
type Kind string

const (
	KindWS     Kind = "ws"
	KindGRPC   Kind = "grpc"
	KindAnyTLS Kind = "anytls"
	KindNomux  Kind = "nomux"
	KindWG     Kind = "wg" // handled by sing-box endpoints, not this bridge
)

// PortPlan public (S listen / P dial) and local (S sing-box dial).
type PortPlan struct {
	Kind        Kind
	PublicPort  int    // Secondary listens; Primary dials
	LocalPort   int    // Secondary 127.0.0.1 for sing-box outbound
	PrimaryDial string // Primary side target after stream open, e.g. 127.0.0.1:10443
	Transport   string // "tcp" | "ws"
}

func DefaultPlans() []PortPlan {
	return []PortPlan{
		{Kind: KindWS, PublicPort: 11543, LocalPort: 12543, PrimaryDial: "127.0.0.1:10443", Transport: "ws"},
		{Kind: KindGRPC, PublicPort: 11544, LocalPort: 12544, PrimaryDial: "127.0.0.1:10444", Transport: "tcp"},
		{Kind: KindAnyTLS, PublicPort: 11545, LocalPort: 12545, PrimaryDial: "127.0.0.1:10445", Transport: "tcp"},
		{Kind: KindNomux, PublicPort: 11546, LocalPort: 12546, PrimaryDial: "127.0.0.1:443", Transport: "tcp"},
	}
}

func PlanFor(k Kind) (PortPlan, bool) {
	for _, p := range DefaultPlans() {
		if p.Kind == k {
			return p, true
		}
	}
	return PortPlan{}, false
}

func ParseKinds(list []string) []Kind {
	var out []Kind
	seen := map[Kind]bool{}
	for _, s := range list {
		k := Kind(strings.ToLower(strings.TrimSpace(s)))
		switch k {
		case KindWS, KindGRPC, KindAnyTLS, KindNomux, KindWG:
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}
