package version

// Release is the product version. Overridden at link time:
//
//	-X github.com/PavelNeyman/netductor/internal/version.Release=0.9.xxx
//
// Keep the default equal to the VERSION file at the tip of main.
var Release = "0.9.518"
