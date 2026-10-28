package main

import (
	"fmt"
	"net"
	"strings"
	"unicode"
)

// validMAC returns canonical lower aa:bb:cc:dd:ee:ff or error.
func validMAC(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "\r\n") {
		return "", fmt.Errorf("invalid mac")
	}
	s = strings.ToLower(strings.ReplaceAll(s, "-", ":"))
	hw, err := net.ParseMAC(s)
	if err != nil {
		return "", fmt.Errorf("invalid mac")
	}
	// Require 6-byte EUI-48.
	if len(hw) != 6 {
		return "", fmt.Errorf("invalid mac length")
	}
	return hw.String(), nil
}

func validIPv4(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "\r\n") {
		return "", fmt.Errorf("invalid ip")
	}
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("invalid ipv4")
	}
	return ip.To4().String(), nil
}

// validDHCPName: hostname-like, no CR/LF, limited charset.
func validDHCPName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty name")
	}
	if len(s) > 63 || strings.ContainsAny(s, "\r\n'\"`") {
		return "", fmt.Errorf("invalid name")
	}
	for i, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		if (r == '-' || r == '_') && i > 0 {
			continue
		}
		return "", fmt.Errorf("invalid name char")
	}
	return s, nil
}

// safeUCIValue rejects CR/LF and quotes that break UCI batch lines.
func safeUCIValue(s string) (string, error) {
	if strings.ContainsAny(s, "\r\n'") {
		return "", fmt.Errorf("unsafe uci value")
	}
	if len(s) > 256 {
		return "", fmt.Errorf("uci value too long")
	}
	return s, nil
}

// parseGatewayCIDR returns gateway IP and *net.IPNet from "192.168.50.1/24".
func parseGatewayCIDR(s string) (gw string, n *net.IPNet, err error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "\r\n") {
		return "", nil, fmt.Errorf("invalid cidr")
	}
	ip, network, err := net.ParseCIDR(s)
	if err != nil {
		return "", nil, fmt.Errorf("invalid cidr")
	}
	if ip.To4() == nil {
		return "", nil, fmt.Errorf("ipv4 cidr required")
	}
	if !network.Contains(ip) {
		return "", nil, fmt.Errorf("gateway outside network")
	}
	ones, bits := network.Mask.Size()
	if bits != 32 || ones < 8 || ones > 30 {
		return "", nil, fmt.Errorf("invalid mask")
	}
	return ip.To4().String(), network, nil
}
