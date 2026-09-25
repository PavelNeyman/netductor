package deploy

import "github.com/PavelNeyman/netductor/internal/version"

// Release is the pinned product version for downloads and deploy defaults.
const Release = version.Release

// NodeAsset returns the GitHub release asset name for the VPS/node binary.
// Operator must never download netductor-op onto a node.
func NodeAsset(goos, goarch string) string {
	return "netductor-" + goos + "-" + goarch
}

// OperatorAsset returns the workstation operator binary asset name.
func OperatorAsset(goos, goarch string) string {
	return "netductor-op-" + goos + "-" + goarch
}
