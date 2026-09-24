package operator

import (
	"github.com/PavelNeyman/netductor/internal/deploy"
)

// DeployPrimary bootstraps a primary VPS (operator machine → SSH).
func DeployPrimary(s PrimarySpec) error {
	ApplyDomainFlags(&s)
	return deploy.DeployPrimary(s.toDeploy())
}

// DeploySecondary provisions secondary Mac-direct.
func DeploySecondary(s SecondarySpec) error {
	return deploy.DeploySecondary(s.toDeploy())
}

// CollectCredentials pulls secrets to ~/.netductor/credentials on the operator machine.
func CollectCredentials(role, user, host, keyPath, keyPass string) (string, error) {
	return deploy.CollectOperatorSecrets(role, user, host, keyPath, keyPass)
}


// FleetDeploy runs primary then secondary in order. Credentials are collected inside each Deploy*.
func FleetDeploy(f FleetSpec) error {
	return FleetDeployWithReport(f, nil)
}

func defaultOperatorKeyPath() string {
	// mirror deploy default without importing private helpers
	return expandHome("~/.ssh/netductor_primary")
}
