package operator

import (
	"fmt"
	"strings"

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
	if !f.DoPrimary && !f.DoSecondary {
		return fmt.Errorf("fleet: nothing selected (primary and/or secondary)")
	}
	var primaryHost, primaryUser, primaryKey, keyPass string
	if f.DoPrimary {
		s := f.Primary
		ApplyDomainFlags(&s)
		if err := DeployPrimary(s); err != nil {
			return fmt.Errorf("fleet primary: %w", err)
		}
		primaryHost = strings.TrimSpace(s.Host)
		primaryUser = strings.TrimSpace(s.User)
		if primaryUser == "" {
			primaryUser = "root"
		}
		primaryKey = strings.TrimSpace(s.SSHPrivateKey)
		keyPass = s.KeyPassphrase
		if primaryKey == "" {
			primaryKey = defaultOperatorKeyPath()
		}
		// ensure absolute for secondary leg
		primaryKey = expandHome(primaryKey)
	}
	if f.DoSecondary {
		sec := f.Secondary
		if sec.PrimaryHost == "" {
			sec.PrimaryHost = primaryHost
		}
		if sec.PrimaryUser == "" {
			sec.PrimaryUser = primaryUser
		}
		if sec.PrimaryKey == "" {
			sec.PrimaryKey = primaryKey
		}
		if sec.PrimaryKeyPassphrase == "" {
			sec.PrimaryKeyPassphrase = keyPass
		}
		if sec.SNI == "" && f.Primary.SNI != "" {
			sec.SNI = f.Primary.SNI
		}
		if sec.PrimaryHost == "" || sec.PrimaryKey == "" {
			return fmt.Errorf("fleet secondary: primary host/key required (deploy primary first or set fields)")
		}
		if err := DeploySecondary(sec); err != nil {
			return fmt.Errorf("fleet secondary: %w", err)
		}
	}
	return nil
}

func defaultOperatorKeyPath() string {
	// mirror deploy default without importing private helpers
	return expandHome("~/.ssh/netductor_primary")
}
