// Package update — release fetch/apply helpers and the unified update contract (Phase 1).
//
// Contract types define component identity, job state machine, and asset naming.
// Existing apply paths (Ship, agent_update, op self-update) remain the implementation
// until Phase 2 wraps them behind this model.
package update

import (
	"fmt"
	"strings"
)

// ComponentID is a stable logical product component (not a hostname).
type ComponentID string

const (
	ComponentNode  ComponentID = "node"  // netductor control-plane binary (primary or secondary host)
	ComponentTG    ComponentID = "tg"    // netductor-tg
	ComponentAgent ComponentID = "agent" // netductor-agent (edge device or secondary agent unit)
	ComponentOp    ComponentID = "op"    // netductor-op (macOS operator)
)

// ValidComponent reports whether id is a supported update target in the unified catalog.
func ValidComponent(id ComponentID) bool {
	switch id {
	case ComponentNode, ComponentTG, ComponentAgent, ComponentOp:
		return true
	default:
		return false
	}
}

// Role of the host or device instance.
type Role string

const (
	RolePrimary   Role = "primary"
	RoleSecondary Role = "secondary"
	RoleEdge      Role = "edge"
	RoleOperator  Role = "operator"
)

// InstanceID identifies where a component runs.
// Examples: primary host id, secondary agent id, edge device_id, "local" for operator.
type InstanceID string

// JobState is the unified update operation state machine (Phase 1).
type JobState string

const (
	StateUnknown            JobState = "unknown"
	StateUnsupported        JobState = "unsupported"
	StateIdle               JobState = "idle"
	StateChecking           JobState = "checking"
	StateAvailable          JobState = "available"
	StateQueued             JobState = "queued"
	StateDownloading        JobState = "downloading"
	StateVerifying          JobState = "verifying"
	StateInstalling         JobState = "installing"
	StateRestarting         JobState = "restarting"
	StateHealthChecking     JobState = "health_checking"
	StateSucceeded          JobState = "succeeded"
	StateFailed             JobState = "failed"
	StateRollbackInProgress JobState = "rollback_in_progress"
	StateRolledBack         JobState = "rolled_back"
	StateCancelled          JobState = "cancelled"
)

// allowedTransitions is the directed graph of legal JobState moves.
// Terminal-ish states may still move to checking for a new operation on the same instance.
var allowedTransitions = map[JobState][]JobState{
	StateUnknown:            {StateChecking, StateUnsupported, StateIdle},
	StateUnsupported:        {StateChecking}, // re-probe after catalog change
	StateIdle:               {StateChecking},
	StateChecking:           {StateAvailable, StateIdle, StateUnsupported, StateFailed},
	StateAvailable:          {StateQueued, StateDownloading, StateIdle, StateChecking},
	StateQueued:             {StateDownloading, StateCancelled, StateFailed},
	StateDownloading:        {StateVerifying, StateFailed, StateCancelled},
	StateVerifying:          {StateInstalling, StateFailed, StateCancelled},
	StateInstalling:         {StateRestarting, StateHealthChecking, StateFailed, StateRollbackInProgress},
	StateRestarting:         {StateHealthChecking, StateFailed, StateRollbackInProgress},
	StateHealthChecking:     {StateSucceeded, StateFailed, StateRollbackInProgress},
	StateSucceeded:          {StateIdle, StateChecking},
	StateFailed:             {StateIdle, StateChecking, StateRollbackInProgress, StateQueued},
	StateRollbackInProgress: {StateRolledBack, StateFailed},
	StateRolledBack:         {StateIdle, StateChecking},
	StateCancelled:          {StateIdle, StateChecking},
}

// CanTransition reports whether moving from → to is allowed by the contract.
func CanTransition(from, to JobState) bool {
	if from == to {
		return true
	}
	for _, s := range allowedTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// ValidateTransition returns an error if from → to is illegal.
func ValidateTransition(from, to JobState) error {
	if CanTransition(from, to) {
		return nil
	}
	return fmt.Errorf("update: illegal state transition %q → %q", from, to)
}

// GOOS/GOARCH pair for asset selection (clients must not invent asset names).
type Platform struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	// GOARM / GOMIPS when relevant (agent arm, mipsle softfloat).
	Extra string `json:"extra,omitempty"`
}

// AssetName returns the canonical release asset basename for component+platform.
// Empty string means unsupported combination.
func AssetName(comp ComponentID, p Platform) string {
	os := strings.ToLower(p.GOOS)
	arch := strings.ToLower(p.GOARCH)
	switch comp {
	case ComponentNode:
		if os != "linux" {
			return ""
		}
		if arch == "amd64" || arch == "arm64" {
			return "netductor-linux-" + arch
		}
	case ComponentTG:
		if os != "linux" {
			return ""
		}
		if arch == "amd64" || arch == "arm64" {
			return "netductor-tg-linux-" + arch
		}
	case ComponentAgent:
		if os != "linux" {
			return ""
		}
		switch arch {
		case "amd64", "arm64", "arm", "mipsle", "riscv64":
			return "netductor-agent-linux-" + arch
		}
	case ComponentOp:
		if os != "darwin" {
			return ""
		}
		if arch == "amd64" || arch == "arm64" {
			return "netductor-op-darwin-" + arch
		}
	}
	return ""
}

// ComponentInstance is the catalog row for one installed target.
type ComponentInstance struct {
	Component       ComponentID `json:"component"`
	InstanceID      InstanceID  `json:"instance_id"`
	Role            Role        `json:"role"`
	Platform        Platform    `json:"platform"`
	InstalledVersion string     `json:"installed_version,omitempty"` // version file / package metadata
	RunningVersion  string      `json:"running_version,omitempty"`   // observed from process when known
	BuildCommit     string      `json:"build_commit,omitempty"`
	DesiredVersion  string      `json:"desired_version,omitempty"`
	AvailableVersion string     `json:"available_version,omitempty"`
	State           JobState    `json:"state"`
	HealthOK        *bool       `json:"health_ok,omitempty"`
	LastError       string      `json:"last_error,omitempty"`
	UpdatedAtUnix   int64       `json:"updated_at_unix,omitempty"`
}

// UpdateJob is one operation against a ComponentInstance.
type UpdateJob struct {
	JobID          string      `json:"job_id"`
	Component      ComponentID `json:"component"`
	InstanceID     InstanceID  `json:"instance_id"`
	FromVersion    string      `json:"from_version,omitempty"`
	TargetVersion  string      `json:"target_version"`
	TargetCommit   string      `json:"target_commit,omitempty"`
	Asset          string      `json:"asset"` // must equal AssetName(...); clients do not invent
	AssetSHA256    string      `json:"asset_sha256,omitempty"`
	State          JobState    `json:"state"`
	ProgressPct    int         `json:"progress_pct,omitempty"`
	LastError      string      `json:"last_error,omitempty"`
	CreatedAtUnix  int64       `json:"created_at_unix"`
	UpdatedAtUnix  int64       `json:"updated_at_unix"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
}

// Contract summary constants for API docs / operators.
const (
	// AuthSession — Web/API session cookie or bearer as used by existing /api/* routes.
	AuthSession = "session"
	// AuthCLI — local root/operator on the host running netductor CLI.
	AuthCLI = "local_cli"
	// AuthTG — Telegram operator allowlist + server-side callback re-auth.
	AuthTG = "telegram_operator"

	// DefaultJobTimeoutSec — soft timeout for a single update job before marking failed/stale.
	DefaultJobTimeoutSec = 900
	// StaleCallbackSec — TG/Web clients must not apply ops older than this without refresh.
	StaleCallbackSec = 300
)
