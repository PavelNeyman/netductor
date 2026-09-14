package vpn

// DefaultRealitySNI is the handshake name used when no secret/env override exists.
// From openlibrecommunity/twl + commercial WL profiles: VK API SNI is widely usable under carrier WL.
// IP of the entry host still must be L3-whitelisted — SNI alone cannot open a non-WL address.
const DefaultRealitySNI = "api.vk.me"

// DefaultUTLSFingerprint matches working commercial WL client profiles.
const DefaultUTLSFingerprint = "firefox"
