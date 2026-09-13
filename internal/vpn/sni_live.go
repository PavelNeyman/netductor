package vpn

// SetActiveSNI is the live-SNI entrypoint (presets resolve in CLI/API).
func SetActiveSNI(sniName string) error {
	return SetSNI(sniName)
}

// ActiveSNI returns current handshake name.
func ActiveSNI() string {
	return sni()
}
