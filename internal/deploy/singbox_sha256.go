package deploy

import "github.com/PavelNeyman/netductor/internal/upstream"

func pinnedSingBoxSHA(sbArch string) string {
	return upstream.SingBoxSHA(sbArch)
}

func matchSHA256(got, want string) bool {
	return upstream.MatchSHA(got, want)
}
