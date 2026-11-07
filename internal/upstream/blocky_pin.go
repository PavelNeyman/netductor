package upstream

// BlockyPinnedTag is the default install target (with "v" prefix as on GitHub releases).
// Override: NETDUCTOR_BLOCKY_TAG or /etc/netductor/addons/blocky.tag
//
// Pin rationale (2026-11):
//   - v0.32.0 fixed GHSA/GO-2026-5745 DNSSEC validation bypass + cache-scope pollution
//   - v0.35.0 is current stable (metrics, refused block type, DNSSEC response hygiene)
const BlockyPinnedTag = "v0.35.0"

// BlockyTarballSHA256: official release checksums for BlockyPinnedTag (Linux only).
// Source: https://github.com/0xERR0R/blocky/releases/download/v0.35.0/blocky_checksums.txt
// Asset names: blocky_v0.35.0_Linux_<arch>.tar.gz
var BlockyTarballSHA256 = map[string]string{
	"x86_64": "97ae4257e4f2ea7ee4e0622fca152019eb92224d81b41b2549f064e4cef5584e",
	"arm64":  "9ac70e846f7919632e340607fb73d80a1fe280c9ceb876430d2a7d79743b4aae",
	"armv7":  "f9c2ebc500f4fa7b74dca4c9cc384ade3fe44a39f308e0f31731403917fe0d95",
	"armv6":  "8415fa2e7d3ee83da321c89a93686cf98a9d597c5cc5843e3aa95665803d88c9",
}

// BlockySHA returns expected tarball hash for release arch key (x86_64, arm64, …).
func BlockySHA(linuxArch string) string {
	return BlockyTarballSHA256[linuxArch]
}
