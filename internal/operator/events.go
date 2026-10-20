package operator

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

// Step is a stable progress ID for TUI/Web (not free-form log lines).
type Step struct {
	ID      string // e.g. primary, secondary, credentials
	Message string
	Done    bool
	Err     string
}

// Reporter receives structured steps. nil = no-op.
type Reporter func(Step)

func report(r Reporter, id, msg string) {
	if r != nil {
		r(Step{ID: id, Message: msg})
	}
}

func reportDone(r Reporter, id, msg string) {
	if r != nil {
		r(Step{ID: id, Message: msg, Done: true})
	}
}

func reportErr(r Reporter, id string, err error) {
	if r != nil && err != nil {
		r(Step{ID: id, Message: err.Error(), Err: err.Error()})
	}
}

// FleetDeployWithReport is FleetDeploy plus step callbacks.
func FleetDeployWithReport(f FleetSpec, r Reporter) error {
	if !f.DoPrimary && !f.DoSecondary && !f.DoBuild {
		return fmt.Errorf("fleet: nothing selected (primary and/or secondary and/or build)")
	}
	var primaryHost, primaryUser, primaryKey, keyPass string
	if f.DoPrimary {
		report(r, "primary", "deploy primary")
		s := f.Primary
		ApplyDomainFlags(&s)
		if err := DeployPrimary(s); err != nil {
			reportErr(r, "primary", err)
			return fmt.Errorf("fleet primary: %w", err)
		}
		reportDone(r, "primary", "primary ok")
		primaryHost = s.Host
		primaryUser = s.User
		if primaryUser == "" {
			primaryUser = "root"
		}
		primaryKey = s.SSHPrivateKey
		keyPass = s.KeyPassphrase
		if primaryKey == "" {
			primaryKey = defaultOperatorKeyPath()
		}
		primaryKey = expandHome(primaryKey)
	}
	if f.DoSecondary {
		report(r, "secondary", "deploy secondary")
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
			err := fmt.Errorf("fleet secondary: primary host/key required")
			reportErr(r, "secondary", err)
			return err
		}
		if err := DeploySecondary(sec); err != nil {
			reportErr(r, "secondary", err)
			return fmt.Errorf("fleet secondary: %w", err)
		}
		reportDone(r, "secondary", "secondary ok")
	}
	if f.DoBuild {
		report(r, "build-host", "deploy build-host")
		if primaryHost == "" {
			primaryHost = f.Primary.Host
		}
		if primaryKey == "" {
			primaryKey = f.Primary.SSHPrivateKey
			if primaryKey == "" {
				primaryKey = defaultOperatorKeyPath()
			}
			primaryKey = expandHome(primaryKey)
			keyPass = f.Primary.KeyPassphrase
		}
		if primaryUser == "" {
			primaryUser = "root"
		}
		if primaryHost == "" || primaryKey == "" {
			err := fmt.Errorf("fleet build-host: primary host/key required")
			reportErr(r, "build-host", err)
			return err
		}
		if strings.TrimSpace(f.Build.Host) == "" {
			err := fmt.Errorf("fleet build-host: host required")
			reportErr(r, "build-host", err)
			return err
		}
		bh := f.Build
		if bh.User == "" {
			bh.User = "root"
		}
		if bh.Label == "" {
			bh.Label = "build-1"
		}
		err := deploy.DeployBuildHost(deploy.BuildHostOpts{
			PrimaryHost: primaryHost, PrimaryUser: primaryUser, PrimaryKey: primaryKey,
			PrimaryKeyPassphrase: keyPass,
			BuildHost:            bh.Host, BuildUser: bh.User, BuildPass: bh.Password, Label: bh.Label,
		})
		if err != nil {
			reportErr(r, "build-host", err)
			return fmt.Errorf("fleet build-host: %w", err)
		}
		reportDone(r, "build-host", "build-host ok")
	}
	reportDone(r, "done", "fleet complete")
	return nil
}
