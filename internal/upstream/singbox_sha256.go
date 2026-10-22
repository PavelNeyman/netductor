package upstream

import "strings"

// SingBoxPinnedVersion is the release we ship/cache/install by default (no "v" prefix).
const SingBoxPinnedVersion = "1.11.15"

// SingBoxTarballSHA256: SHA-256 of official SagerNet release tarballs for SingBoxPinnedVersion.
// Key = linux-<arch> as in asset name. Recompute on pin bump.
var SingBoxTarballSHA256 = map[string]string{
	"linux-amd64":   "950af37eb2d7e55dddae34a18411cd617303fd99d2dc75bc76b6dd9fcd97d9c5",
	"linux-arm64":   "20a6a9cd259a95411599f811a5066513a98db63705a51121252ad27daf96c029",
	"linux-armv7":   "490461d5a107847080b896d575a6cb74b31b64e2ad8bbb47635f6d227e73d7c8",
	"linux-mipsle":  "c241c7dceeb687e602ce93068034b6924944f446fc8fa1346e617866c45ecf78",
	"linux-riscv64": "782b61e908d7cfd2b3ec7c232ccc320cfbe41ffae62a957b771769e54f04b3ed",
}

// SingBoxSHA returns expected tarball hash for release arch (amd64, arm64, armv7, mipsle, …).
func SingBoxSHA(linuxArch string) string {
	return SingBoxTarballSHA256["linux-"+linuxArch]
}

// MatchSHA reports case-insensitive equality of hex digests.
func MatchSHA(got, want string) bool {
	return strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want))
}
