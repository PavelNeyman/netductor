package version

// Release is the product version. Overridden at link time:
//
//	-X github.com/PavelNeyman/netductor/internal/version.Release=0.9.xxx
var Release = "0.9.629"

// Commit is the git SHA when set at link time:
//
//	-X github.com/PavelNeyman/netductor/internal/version.Commit=abc1234
var Commit = ""
