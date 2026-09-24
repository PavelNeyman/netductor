package operator

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

// DeployPrimary bootstraps a primary VPS (operator machine → SSH).
func DeployPrimary(s PrimarySpec) error {
	ApplyDomainFlags(&s)
	if !ValidHost(s.Host) {
		return fmt.Errorf("invalid primary host")
	}
	if s.User == "" {
		s.User = "root"
	}
	if !ValidUser(s.User) {
		return fmt.Errorf("invalid primary user")
	}
	return deploy.DeployPrimary(s.toDeploy())
}

// DeploySecondary provisions secondary Mac-direct.
func DeploySecondary(s SecondarySpec) error {
	if !ValidHost(s.SecondaryHost) {
		return fmt.Errorf("invalid secondary host")
	}
	if s.PrimaryHost != "" && !ValidHost(s.PrimaryHost) {
		return fmt.Errorf("invalid primary host")
	}
	if s.SecondaryUser == "" {
		s.SecondaryUser = "root"
	}
	if s.PrimaryUser == "" {
		s.PrimaryUser = "root"
	}
	if !ValidUser(s.SecondaryUser) || !ValidUser(s.PrimaryUser) {
		return fmt.Errorf("invalid ssh user")
	}
	return deploy.DeploySecondary(s.toDeploy())
}

// DeployEdge provisions OpenWrt/RPi agent from the operator machine (Mac-direct).
func DeployEdge(s EdgeSpec) error {
	if !ValidHost(s.RouterHost) {
		return fmt.Errorf("invalid router host")
	}
	if s.PrimaryHost != "" && !ValidHost(s.PrimaryHost) {
		return fmt.Errorf("invalid primary host")
	}
	if s.RouterUser == "" {
		s.RouterUser = "root"
	}
	if s.PrimaryUser == "" {
		s.PrimaryUser = "root"
	}
	if !ValidUser(s.RouterUser) || !ValidUser(s.PrimaryUser) {
		return fmt.Errorf("invalid ssh user")
	}
	return deploy.DeployEdge(s.toDeploy())
}

// CollectCredentials pulls secrets to ~/.netductor/credentials on the operator machine.
func CollectCredentials(role, user, host, keyPath, keyPass string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "primary", "secondary", "node":
	default:
		role = "node"
	}
	if !ValidHost(host) {
		return "", fmt.Errorf("invalid host")
	}
	if user == "" {
		user = "root"
	}
	if !ValidUser(user) {
		return "", fmt.Errorf("invalid user")
	}
	return deploy.CollectOperatorSecrets(role, user, host, keyPath, keyPass)
}

// FleetDeploy runs primary then secondary in order. Credentials are collected inside each Deploy*.
func FleetDeploy(f FleetSpec) error {
	return FleetDeployWithReport(f, nil)
}

func defaultOperatorKeyPath() string {
	return expandHome("~/.ssh/netductor_primary")
}
