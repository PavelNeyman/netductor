package main

import (
	qrcode "github.com/skip2/go-qrcode"
)

func guestQRPNG(payload string) ([]byte, error) {
	return qrcode.Encode(payload, qrcode.Medium, 256)
}
