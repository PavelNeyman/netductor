package svcpaths

import "testing"

func TestValidateUnitExecStart(t *testing.T) {
	tests := []struct {
		name      string
		unit      string
		execStart string
		wantErr   bool
	}{
		{
			name:      "valid absolute command with arguments",
			unit:      "nd-wss-sp-server-nd-svc-sp1",
			execStart: "/usr/local/bin/wstunnel server wss://0.0.0.0:8454 --restrict-to 127.0.0.1:51840",
		},
		{name: "empty command", unit: "nd-wss", execStart: "  ", wantErr: true},
		{name: "relative executable", unit: "nd-wss", execStart: "wstunnel server", wantErr: true},
		{name: "newline injection", unit: "nd-wss", execStart: "/usr/bin/true\nExecStart=/bin/sh", wantErr: true},
		{name: "carriage return", unit: "nd-wss", execStart: "/usr/bin/true\rExecStart=/bin/sh", wantErr: true},
		{name: "nul byte", unit: "nd-wss", execStart: "/usr/bin/true\x00", wantErr: true},
		{name: "systemd specifier", unit: "nd-wss", execStart: "/usr/bin/echo %n", wantErr: true},
		{name: "unit path traversal", unit: "../nd-wss", execStart: "/usr/bin/true", wantErr: true},
		{name: "unit slash", unit: "nd/wss", execStart: "/usr/bin/true", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUnitExecStart(tt.unit, tt.execStart)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateUnitExecStart() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
