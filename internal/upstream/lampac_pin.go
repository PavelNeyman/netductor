package upstream

// LampacPinnedImage is the default Docker image reference (tag, not :latest).
// Override: NETDUCTOR_LAMPAC_IMAGE
//
// Pin rationale (2026-11): track lampac-nextgen release 1.87.8 (GHCR).
// Prefer explicit version over floating :latest so fleet installs are reproducible.
const LampacPinnedImage = "ghcr.io/lampac-nextgen/lampac:1.87.8"

// LampacPinnedTag is the image tag portion only (for state files / logs).
const LampacPinnedTag = "1.87.8"
