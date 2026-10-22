package upstream

import "strings"

// SingBoxPinnedVersion is the release we ship/cache by default (no "v" prefix).
const SingBoxPinnedVersion = "1.11.7"

// SingBoxTarballSHA256: SHA-256 of official SagerNet release tarballs for SingBoxPinnedVersion.
// Key = linux-<arch> as in asset name. Recompute on pin bump.
var SingBoxTarballSHA256 = map[string]string{
	"linux-amd64":   "30420c7e1a0e4b9c7ee2ff3992c53257be85dec2bdc93074594c8b92d19d4d71",
	"linux-arm64":   "d309eec0b2251223227be9bef09c07c348df31d55677e2a785415aa542911951",
	"linux-armv7":   "ba783a28de32c4ffa43aaa21b5c2b471018fcbc400c5162a13f6be8c37f6abf1",
	"linux-mipsle":  "b0e777b33d41ab66b22aae2053ef3706ca9defd1778912ce6ccbdbb0925b2c2d",
	"linux-riscv64": "f97e01378dfd6c40e6e23967f1d622ea2d8efeec6b66dde69478cc4f271d5a85",
}

// SingBoxSHA returns expected tarball hash for release arch (amd64, arm64, armv7, mipsle, …).
func SingBoxSHA(linuxArch string) string {
	return SingBoxTarballSHA256["linux-"+linuxArch]
}

// MatchSHA reports case-insensitive equality of hex digests.
func MatchSHA(got, want string) bool {
	return strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want))
}
