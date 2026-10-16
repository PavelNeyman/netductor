package edgestorage

// ExpectUSBFlag is written on the edge device when operator expects USB/extroot.
const ExpectUSBFlag = "/etc/netductor-agent/expect_usb"

// Agent storage facts keys (heartbeat extra).
const (
	FactUSBPresent    = "usb_present"
	FactExtrootActive = "extroot_active"
	FactExpectUSB     = "expect_usb"
)
