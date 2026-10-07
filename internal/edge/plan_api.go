package edge

import (
	"fmt"
	"time"
)

// PlanRequest is the JSON body for POST /api/edge/plan (and local op library).
// Same shape for Web, TUI, CLI — no UI-specific fields.
type PlanRequest struct {
	Preset    string          `json:"preset,omitempty"`
	Facts     *DeviceFacts    `json:"facts,omitempty"`
	DeviceID  string          `json:"device_id,omitempty"` // if facts omitted, use stored
	Intent    DeployIntent    `json:"intent"`
	Selection ModuleSelection `json:"selection"`
}

// PlanResponse is returned to all thin UIs.
type PlanResponse struct {
	Preset    string       `json:"preset"`
	Suggested string       `json:"suggested_preset"`
	Facts     DeviceFacts  `json:"facts"`
	Plan      DeployPlan   `json:"plan"`
	Card      DeviceCard   `json:"card"`
}

// DeviceCard is a stable view-model for UI (derived from facts only).
type DeviceCard struct {
	Model      string   `json:"model,omitempty"`
	Board      string   `json:"board,omitempty"`
	Serial     string   `json:"serial,omitempty"`
	Arch       string   `json:"arch,omitempty"`
	OS         string   `json:"os,omitempty"`
	LANIP      string   `json:"lan_ip,omitempty"`
	WANPresent bool     `json:"wan_present"`
	WANCapable bool     `json:"wan_capable"`
	EthCount   int      `json:"eth_count"`
	Bands      []string `json:"bands,omitempty"`
	SSIDNow    []string `json:"ssid_now,omitempty"`
	MemAvailKB int64    `json:"mem_avail_kb,omitempty"`
	MemTotalKB int64    `json:"mem_total_kb,omitempty"`
	HasMMC     bool     `json:"has_mmc,omitempty"`
	HasUSBDisk bool     `json:"has_usb_disk,omitempty"`
	ExpandHint bool     `json:"expand_hint,omitempty"`
}

// CardFromFacts builds UI card from facts.
func CardFromFacts(f DeviceFacts) DeviceCard {
	c := DeviceCard{
		Model: f.Model, Board: f.Board, Serial: f.Serial, Arch: f.Arch, OS: f.OS,
		LANIP: f.UCI.LANIP, WANPresent: f.UCI.WANPresent, WANCapable: f.WANCapable(),
		EthCount: f.EthernetCount(), Bands: f.Bands(),
		MemAvailKB: f.MemAvailKB, MemTotalKB: f.MemTotalKB,
		HasMMC: f.Storage.HasMMC, HasUSBDisk: f.Storage.HasUSBDisk, ExpandHint: f.Storage.ExpandHint,
	}
	if c.Model == "" {
		c.Model = f.Board
	}
	for _, s := range f.SSIDs {
		if s.SSID != "" && !s.Disabled {
			c.SSIDNow = append(c.SSIDNow, s.SSID)
		}
	}
	return c
}

// ResolvePlan builds plan from request; loads stored facts when device_id set and facts nil.
func ResolvePlan(req PlanRequest) (PlanResponse, error) {
	var f DeviceFacts
	if req.Facts != nil {
		f = *req.Facts
	} else if id := req.DeviceID; id != "" {
		d, ok := GetDevice(id)
		if !ok {
			return PlanResponse{}, fmt.Errorf("unknown device")
		}
		if d.Facts == nil {
			return PlanResponse{}, fmt.Errorf("no facts stored for device (probe or agent report first)")
		}
		f = *d.Facts
	} else {
		return PlanResponse{}, fmt.Errorf("facts or device_id required")
	}
	preset := req.Preset
	if preset == "" {
		preset = SuggestPreset(f)
	}
	if !ValidPreset(preset) {
		return PlanResponse{}, fmt.Errorf("unknown preset %q", preset)
	}
	plan := BuildPlan(preset, f, req.Intent)
	plan = ApplySelection(plan, req.Selection, f)
	return PlanResponse{
		Preset:    plan.Preset,
		Suggested: SuggestPreset(f),
		Facts:     f,
		Plan:      plan,
		Card:      CardFromFacts(f),
	}, nil
}

// SetDeviceFacts stores probe/agent facts on the device record (operator or agent).
func SetDeviceFacts(deviceID string, f DeviceFacts) error {
	mu.Lock()
	defer mu.Unlock()
	m := loadDevices()
	d, ok := m[deviceID]
	if !ok {
		return fmt.Errorf("unknown device")
	}
	cp := f
	d.Facts = &cp
	d.FactsAt = time.Now().Unix()
	if f.Board != "" {
		d.Board = f.Board
	}
	if f.Arch != "" {
		d.Arch = f.Arch
	}
	m[deviceID] = d
	return saveDevices(m)
}
